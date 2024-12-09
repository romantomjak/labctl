package ssh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"

	"github.com/romantomjak/labctl/config"
)

type Client struct {
	cfg *config.Config
	ssh *ssh.Client
	buf *bytes.Buffer
}

func New(cfg *config.Config) (*Client, error) {
	privateKeyFile, err := expandTilde(cfg.Kubernetes.SSH.PrivateKeyFile)
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

	hostKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(cfg.Kubernetes.SSH.HostKey))
	if err != nil {
		return nil, fmt.Errorf("parse host key: %w", err)
	}

	config := &ssh.ClientConfig{
		User: cfg.Kubernetes.SSH.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	client, err := ssh.Dial("tcp", cfg.Kubernetes.SSH.Addr, config)
	if err != nil {
		return nil, err
	}

	return &Client{cfg, client, &bytes.Buffer{}}, nil
}

func (c *Client) SnapshotETCD(filename string) error {
	// Only root can read certs for connecting to the etcd cluster.
	cmd := "sudo etcdctl --cacert=/etc/kubernetes/pki/etcd/ca.crt --cert=/etc/kubernetes/pki/etcd/server.crt --key=/etc/kubernetes/pki/etcd/server.key snapshot save " + filename
	if _, err := c.run(cmd); err != nil {
		return fmt.Errorf("save: %w", err)
	}

	// Update file permissions to allow scp'ing the snapshot back to local machine.
	cmd = fmt.Sprintf("sudo chown %s:%s %s", c.cfg.Kubernetes.SSH.Username, c.cfg.Kubernetes.SSH.Username, filename)
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
		return "", fmt.Errorf("run command: %w", err)
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
