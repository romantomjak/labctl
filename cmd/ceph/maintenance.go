package ceph

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/config"
	"github.com/romantomjak/labctl/ssh"
)

var enterMaintenanceLong = strings.Trim(`
Host maintenance commands

  Place hosts into maintenance mode during troubleshooting or OSD maintenance to
  avoid CRUSH algorithm automatically rebalance data across OSDs when an OSD fails
  or is stopped.
`, "\n")

var maintenance = &cobra.Command{
	Use:          "maintenance [command]",
	Short:        "Host maintenance commands",
	Long:         enterMaintenanceLong,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
}

var enterMaintenance = &cobra.Command{
	Use:          "enter [flags] <host>",
	Short:        "Place host in maintenance",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         enterMaintenanceCommandFunc,
}

var exitMaintenance = &cobra.Command{
	Use:          "exit [flags] <host>",
	Short:        "Return host from maintenance",
	Args:         cobra.ExactArgs(1),
	SilenceUsage: true,
	RunE:         exitMaintenanceCommandFunc,
}

func enterMaintenanceCommandFunc(cmd *cobra.Command, args []string) error {
	host, err := loadHostConfiguration(args[0])
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	fmt.Println("🔒 Connecting to cluster")

	sshClient, err := sshToRandomClusterNodeExcept(host.Name)
	if err != nil {
		return fmt.Errorf("ssh: %w", err)
	}
	defer sshClient.Close()

	fmt.Printf("🚧 Placing %s into maintenance mode\n", host.Name)

	if err := sshClient.CephEnterMaintenance(host.Name); err != nil {
		return fmt.Errorf("enter maintenance: %w", err)
	}

	fmt.Println("✅ OK")

	return nil
}

func exitMaintenanceCommandFunc(cmd *cobra.Command, args []string) error {
	host, err := loadHostConfiguration(args[0])
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	fmt.Println("🔒 Connecting to cluster")

	sshClient, err := sshToRandomClusterNodeExcept(host.Name)
	if err != nil {
		return fmt.Errorf("ssh: %w", err)
	}
	defer sshClient.Close()

	fmt.Printf("♻️  Returning %s from maintenance mode\n", host.Name)

	if err := sshClient.CephExitMaintenance(host.Name); err != nil {
		return fmt.Errorf("exit maintenance: %w", err)
	}

	fmt.Println("✅ OK")

	return nil
}

func sshToRandomClusterNodeExcept(hostname string) (*ssh.Client, error) {
	cfg, err := config.FromFile("~/.labctl.hcl")
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}

	switch n := len(cfg.Ceph.Nodes); {
	case n > 1:
		for {
			node := cfg.Ceph.Nodes[rand.Intn(n)]

			// We can return early if we don't need to filter hosts.
			if hostname == "" {
				return ssh.New(node)
			}

			// Exclude hosts matching the filter.
			if strings.EqualFold(node.Name, hostname) {
				continue
			}

			return ssh.New(node)
		}

	case n == 1:
		node := cfg.Ceph.Nodes[0]

		if strings.EqualFold(node.Name, hostname) {
			return nil, fmt.Errorf("only one host defined in configuration and it was excluded by hostname filter")
		}

		return ssh.New(node)

	default:
		return nil, fmt.Errorf("no hosts defined in configuration")
	}
}

func loadHostConfiguration(hostname string) (config.Node, error) {
	cfg, err := config.FromFile("~/.labctl.hcl")
	if err != nil {
		return config.Node{}, err
	}

	for _, n := range cfg.Ceph.Nodes {
		if strings.EqualFold(n.Name, hostname) {
			return n, nil
		}
	}

	return config.Node{}, fmt.Errorf("host %q is not defined in configuration", hostname)
}
