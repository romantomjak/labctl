package proxmox

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/luthermonson/go-proxmox"
	"golang.org/x/sync/errgroup"
)

var cluster = &multiClient{
	mu:            sync.Mutex{},
	clientsByNode: map[string]*proxmox.Client{},
}

// multiClient makes it possible to interact with virtual machines on separate
// hosts as if they all are part of the same cluster.
type multiClient struct {
	mu            sync.Mutex
	clientsByNode map[string]*proxmox.Client
}

func (c *multiClient) Resources(ctx context.Context) (proxmox.ClusterResources, error) {
	// Comma separated list of nodes and their credentials.
	nodes := os.Getenv("PROXMOX_NODES")

	// If no nodes are given - check for existing proxmox API credentials.
	if len(nodes) == 0 {
		addr := os.Getenv("PROXMOX_ADDR")
		user := os.Getenv("PROXMOX_USER")
		password := os.Getenv("PROXMOX_PASSWORD")

		if addr != "" && user != "" {
			nodes = fmt.Sprintf("https://%s:%s@%s", user, password, addr)
		}
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no proxmox addrs found")
	}

	g, gCtx := errgroup.WithContext(ctx)

	var (
		rsMu      sync.Mutex
		resources proxmox.ClusterResources
	)

	for _, node := range strings.Split(nodes, ",") {
		u, err := url.Parse(node)
		if err != nil {
			return nil, fmt.Errorf("invalid proxmox addr: %w", err)
		}

		// Grab the username and password and remove them from the url.
		user := u.User.Username()
		password, _ := u.User.Password()
		u.User = &url.Userinfo{}

		// Parsed addrs won't have a path, so set it now.
		u.Path = "/api2/json"

		addr := u.String()

		if addr == "" || user == "" {
			continue // ignore invalid configs
		}

		g.Go(func() error {
			client := proxmox.NewClient(addr,
				proxmox.WithHTTPClient(&http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true,
						},
					},
				}),
				proxmox.WithCredentials(&proxmox.Credentials{
					Username: user,
					Password: password,
					Realm:    "pam",
				}),
			)

			clientCluster, err := client.Cluster(gCtx)
			if err != nil {
				return err
			}

			// Remember client for each node. We'll use this to interact with
			// virtual machines on that node.
			c.mu.Lock()
			for _, node := range clientCluster.Nodes {
				c.clientsByNode[node.ID] = client
			}
			c.mu.Unlock()

			// Query virtual machines on each node and aggregate them into a
			// single response.
			rs, err := clientCluster.Resources(gCtx, "vm")
			if err != nil {
				return err
			}

			rsMu.Lock()
			resources = append(resources, rs...)
			rsMu.Unlock()

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return resources, nil
}

// Client returns a proxmox client for interacting with the given virtual machine.
//
// Client will always be the same for all hosts in the same cluster. Hosts that are
// specified via the PROXMOX_NODES environment variable (and thus are not part of a
// cluster) will receive a new client per host. This makes it possible to interact
// with virtual machines as if they all were part of the same cluster.
func (c *multiClient) Client(vm VirtualMachine) (*proxmox.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for node, client := range c.clientsByNode {
		if node == vm.Node {
			return client, nil
		}
	}

	return nil, fmt.Errorf("no client found for node %q", vm.Node)
}
