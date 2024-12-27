package ceph

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/config"
	"github.com/romantomjak/labctl/ssh"
	"github.com/romantomjak/labctl/wakeonlan"
)

var bootExample = strings.Trim(`
  # Boot the whole cluster
  labctl ceph boot
`, "\n")

var boot = &cobra.Command{
	Use:          "boot [flags]",
	Short:        "Start ceph nodes",
	Example:      bootExample,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         bootCommandFunc,
}

func bootCommandFunc(cmd *cobra.Command, args []string) error {
	cfg, err := config.FromFile("~/.labctl.hcl")
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	fmt.Println("📡 Broadcasting Wake-on-LAN packets")
	for _, node := range cfg.Ceph.Nodes {
		if err := wakeonlan.Broadcast(node.MAC); err != nil {
			return fmt.Errorf("wake on lan: %w", err)
		}
	}

	fmt.Println("⏳ Waiting for ssh to become available")
	var node config.Node
SSHLOOP:
	for {
		select {
		case <-time.Tick(time.Second):
			if n, ok := firstAvailable(cfg.Ceph.Nodes); ok {
				fmt.Println(BrightBlack + " ↳ Node " + n.Name + " is first available" + Reset)
				node = n
				break SSHLOOP
			}
		case <-time.After(time.Minute):
			return fmt.Errorf("timed out waiting for ssh")
		}
	}

	sshClient, err := ssh.New(node)
	if err != nil {
		return fmt.Errorf("ssh: %w", err)
	}
	defer sshClient.Close()

	fmt.Println("⏳ Waiting for all services to start")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

LOOP:
	for {
		select {
		case <-ticker.C:
			services, err := sshClient.ListCephServices()
			if err != nil {
				return fmt.Errorf("list services: %w", err)
			}

			allRunning := true
			for _, service := range services {
				if service.Status.Running != service.Status.Size {
					allRunning = false
				}
			}

			if allRunning {
				break LOOP
			}
		case <-ctx.Done():
			return fmt.Errorf("list services: %w", ctx.Err())
		}
	}

	fmt.Println("🚩 Unsetting cluster-wide OSD flags")
	flags := []string{"noout", "nodown", "nobackfill", "norecover", "norebalance", "pause"}
	for _, flag := range flags {
		fmt.Println(BrightBlack + " ↳ " + flag + Reset)
		if err := sshClient.UnsetOSDFlag(flag); err != nil {
			return fmt.Errorf("unset flag: %w", err)
		}
	}

	// TODO: Bring the CephFS cluster back up

	fmt.Println("⛑️  Waiting for cluster to become healthy")
	ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ticker = time.NewTicker(1 * time.Second)
	defer ticker.Stop()

LOOP2:
	for {
		select {
		case <-ticker.C:
			health, err := sshClient.CephHealth()
			if err != nil {
				return fmt.Errorf("ceph health: %w", err)
			}

			if health == CephStatusHealthy {
				fmt.Println(BrightBlack + " ↳ Cluster is healthy" + Reset)
				break LOOP2
			}
		case <-ctx.Done():
			fmt.Println(BrightBlack + " ↳ Cluster is not healthy" + Reset)
			break LOOP2
		}
	}

	fmt.Println("✅ All done!")

	return nil
}

func firstAvailable(nodes []config.Node) (config.Node, bool) {
	timeout := 300 * time.Millisecond

	c := make(chan config.Node)
	checkSSH := func(node config.Node) {
		conn, err := net.DialTimeout("tcp", node.Addr, timeout)
		if err != nil {
			return
		}
		c <- node
		conn.Close()
	}

	for _, node := range nodes {
		go checkSSH(node)
	}

	select {
	case n := <-c:
		return n, true
	case <-time.After(300 * time.Millisecond):
		return config.Node{}, false
	}
}
