package ssh

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/romantomjak/labctl/config"
)

const DaemonStatusStopped = 0

var (
	ErrAlreadyInMaintenance = errors.New("already in maintenance")
	ErrNotInMaintenance     = errors.New("not in maintenance")
)

type Client struct {
	node config.Node
	ssh  *ssh.Client
	buf  *bytes.Buffer
}

func New(node config.Node) (*Client, error) {
	privateKeyFile, err := expandTilde(node.PrivateKeyFile)
	if err != nil {
		return nil, err
	}

	pem, err := os.ReadFile(privateKeyFile)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	key, err := ssh.ParsePrivateKey(pem)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}

	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(node.HostKey))
	if err != nil {
		return nil, fmt.Errorf("parse host key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: node.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	client, err := ssh.Dial("tcp", node.Addr, config)
	if err != nil {
		return nil, err
	}

	return &Client{node, client, &bytes.Buffer{}}, nil
}

func (c *Client) SnapshotETCD(filename string) error {
	// Only root can read certs for connecting to the etcd cluster.
	cmd := "sudo etcdctl --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key snapshot save " + filename
	if _, err := c.run(cmd); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	// Update file permissions to allow scp'ing the snapshot back to local machine.
	cmd = fmt.Sprintf("sudo chown %s:%s %s", c.node.Username, c.node.Username, filename)
	if _, err := c.run(cmd); err != nil {
		return fmt.Errorf("chown: %w", err)
	}

	return nil
}

func (c *Client) Compress(filename string) error {
	if _, err := c.run("zstd --rm " + filename); err != nil {
		return fmt.Errorf("zstd: %w", err)
	}
	return nil
}

func (c *Client) Copy(src, dst string) error {
	sftp, err := sftp.NewClient(c.ssh)
	if err != nil {
		return fmt.Errorf("sftp client: %w", err)
	}
	defer sftp.Close()

	srcFile, err := sftp.Open(src)
	if err != nil {
		return fmt.Errorf("sftp open: %w", err)
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		return fmt.Errorf("copy: %w", err)
	}

	return nil
}

func (c *Client) SHA512Sum(filename string) (string, error) {
	out, err := c.run("sha512sum " + filename)
	if err != nil {
		return "", fmt.Errorf("sha512sum: %w", err)
	}

	hash := strings.SplitN(out, " ", 2)[0] // format: <hash><space><filename>

	return hash, nil
}

func (c *Client) Delete(filename string) error {
	if _, err := c.run("rm " + filename); err != nil {
		return fmt.Errorf("remove snapshot: %w", err)
	}
	return nil
}

func (c *Client) CephEnterMaintenance(hostname string) error {
	// Redirect stderr to stdout so we can inspect the output and return
	// a more specialised error if host is already in maintenance mode.
	out, err := c.run("sudo ceph orch host maintenance enter " + hostname + " 2>&1")
	if err != nil {
		if strings.Contains(out, "already in maintenance") {
			return ErrAlreadyInMaintenance
		}
		return err
	}
	return nil
}

func (c *Client) CephExitMaintenance(hostname string) error {
	// Redirect stderr to stdout so we can inspect the output and return
	// a more specialised error if host is not in maintenance mode.
	out, err := c.run("sudo ceph orch host maintenance exit " + hostname + " 2>&1")
	if err != nil {
		if strings.Contains(out, "not in maintenance mode") {
			return ErrNotInMaintenance
		}
		return err
	}
	return nil
}

func (c *Client) CephInMaintenance(hostname string) (bool, error) {
	out, err := c.run("sudo ceph orch host ls --format json --host_pattern " + hostname)
	if err != nil {
		return false, err
	}
	return strings.Contains(out, `"status": "maintenance"`), nil
}

func (c *Client) CephHealth() (string, error) {
	out, err := c.run("sudo ceph health")
	if err != nil {
		return "", fmt.Errorf("ceph health: %w", err)
	}
	return strings.TrimSpace(out), nil
}

func (c *Client) SetOSDFlag(flag string) error {
	// For some reason, the output is written to stderr, so
	// we must redirect stderr to stdout ¯\_(ツ)_/¯
	out, err := c.run("sudo ceph osd set " + flag + " 2>&1")
	if err != nil {
		return fmt.Errorf("ceph osd set: %w", err)
	}

	// We could run another command to check the key was set, but
	// instead we'll check if the command returned expected output.
	out = strings.TrimSpace(out)

	sentinel := flag + " is set"
	if flag == "pause" {
		// Weirdly, setting this flag actually sets two flags - one to
		// pause reads and one to pause writes!
		sentinel = "pauserd,pausewr is set"
	}

	if out != sentinel {
		return fmt.Errorf("ceph osd set: %v", out)
	}

	return nil
}

func (c *Client) UnsetOSDFlag(flag string) error {
	// For some reason, the output is written to stderr, so
	// we must redirect stderr to stdout ¯\_(ツ)_/¯
	out, err := c.run("sudo ceph osd unset " + flag + " 2>&1")
	if err != nil {
		return fmt.Errorf("ceph osd unset: %w", err)
	}

	// We could run another command to check the key was unset, but
	// instead we'll check if the command returned expected output.
	out = strings.TrimSpace(out)

	sentinel := flag + " is unset"
	if flag == "pause" {
		// Weirdly, unseting this flag actually unsets two flags - one that
		// pauses reads and one that pauses writes!
		sentinel = "pauserd,pausewr is unset"
	}

	if out != sentinel {
		return fmt.Errorf("ceph osd unset: %v", out)
	}

	return nil
}

func (c *Client) StopCephService(name string) error {
	if _, err := c.run("sudo ceph orch stop " + name); err != nil {
		return fmt.Errorf("ceph orch stop: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			daemons, err := c.CephStatusByServiceName(name)
			if err != nil {
				return fmt.Errorf("daemon status: %w", err)
			}

			allStopped := true
			for _, daemon := range daemons {
				if daemon.Status != DaemonStatusStopped {
					allStopped = false
				}
			}

			if allStopped {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("ceph orch stop: %w", ctx.Err())
		}
	}
}

func (c *Client) CephStatusByServiceName(name string) ([]CephDaemon, error) {
	return c.cephOrchPs("--service_name " + name)
}

type CephDaemon struct {
	Type   string `json:"daemon_type"`
	Host   string `json:"hostname"`
	Status int    `json:"status"`
	ID     string `json:"daemon_id"`
}

func (c *Client) cephOrchPs(filter string) ([]CephDaemon, error) {
	out, err := c.run("sudo ceph orch ps -f json " + filter)
	if err != nil {
		return nil, fmt.Errorf("ceph orch ps: %w", err)
	}

	var daemons []CephDaemon
	if err := json.Unmarshal([]byte(out), &daemons); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}

	return daemons, nil
}

func (c *Client) CephStatusByDaemonType(daemonType string) ([]CephDaemon, error) {
	return c.cephOrchPs("--daemon_type " + daemonType)
}

func (c *Client) StopCephDaemon(name string) error {
	if _, err := c.run("sudo ceph orch daemon stop " + name); err != nil {
		return fmt.Errorf("ceph orch daemon stop: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			daemons, err := c.CephStatusByDaemonName(name)
			if err != nil {
				return fmt.Errorf("daemon status: %w", err)
			}

			allStopped := true
			for _, daemon := range daemons {
				if daemon.Status != DaemonStatusStopped {
					allStopped = false
				}
			}

			if allStopped {
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("ceph orch daemon stop: %w", ctx.Err())
		}
	}
}

func (c *Client) CephStatusByDaemonName(name string) ([]CephDaemon, error) {
	parts := strings.SplitN(name, ".", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("%q is not a valid daemon name", name)
	}
	return c.cephOrchPs("--daemon_type " + parts[0] + " --daemon_id " + parts[1])
}

func (c *Client) CephFSID() (string, error) {
	out, err := c.run("sudo ceph fsid")
	if err != nil {
		return "", fmt.Errorf("ceph fsid: %w", err)
	}
	return strings.TrimSpace(out), nil
}

type CephService struct {
	Name   string `json:"service_name"`
	Status struct {
		Running int `json:"running"`
		Size    int `json:"size"`
	} `json:"status"`
}

func (c *Client) ListCephServices() ([]CephService, error) {
	out, err := c.run("sudo ceph orch ls -f json")
	if err != nil {
		return nil, fmt.Errorf("ceph orch ls: %w", err)
	}

	var services []CephService
	if err := json.Unmarshal([]byte(out), &services); err != nil {
		return nil, fmt.Errorf("json: %w", err)
	}

	return services, nil
}

func (c *Client) StopSystemdService(name string) error {
	if _, err := c.run("sudo systemctl stop " + name); err != nil {
		return fmt.Errorf("systemctl stop: %w", err)
	}
	return nil
}

func (c *Client) Shutdown() error {
	if _, err := c.run("sudo shutdown"); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}

func (c *Client) Close() error {
	return c.ssh.Close()
}

func (c *Client) run(cmd string) (string, error) {
	sess, err := c.ssh.NewSession()
	if err != nil {
		return "", fmt.Errorf("new session: %w", err)
	}
	defer sess.Close()

	c.buf.Reset()

	sess.Stdout = c.buf

	if err := sess.Run(cmd); err != nil {
		// Output can be incomplete or missing, but return everything we have
		// to allow inspecting output for specific errors or sentinel values.
		return c.buf.String(), fmt.Errorf("run command: %w", err)
	}

	return c.buf.String(), nil
}

func expandTilde(filename string) (string, error) {
	if !strings.HasPrefix(filename, "~") {
		return filename, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}

	return strings.Replace(filename, "~", home, 1), nil
}
