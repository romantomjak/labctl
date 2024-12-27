package ceph

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/config"
	"github.com/romantomjak/labctl/ssh"
)

const CephStatusHealthy = "HEALTH_OK"

const (
	Reset       = "\033[0m"
	BrightBlack = "\033[90m"
)

var poweroffExample = strings.Trim(`
  # Shutdown the whole cluster
  labctl ceph poweroff
`, "\n")

var poweroff = &cobra.Command{
	Use:          "poweroff [flags]",
	Short:        "Shut down ceph nodes",
	Example:      poweroffExample,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE:         poweroffCommandFunc,
}

func poweroffCommandFunc(cmd *cobra.Command, args []string) error {
	cfg, err := config.FromFile("~/.labctl.hcl")
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	// Select a random ceph node for performing operations on the cluster.
	randomInt := rand.Intn(len(cfg.Ceph.Nodes))
	randomNode := cfg.Ceph.Nodes[randomInt]

	sshClient, err := ssh.New(randomNode)
	if err != nil {
		return fmt.Errorf("ssh: %w", err)
	}
	defer sshClient.Close()

	fsid, err := sshClient.CephFSID()
	if err != nil {
		return fmt.Errorf("fsid: %w", err)
	}
	fmt.Println("👉 Connected to " + fsid)

	fmt.Println("⛑️  Checking cluster health")
	health, err := sshClient.CephHealth()
	if err != nil {
		return fmt.Errorf("ceph health: %w", err)
	}
	if health != CephStatusHealthy {
		fmt.Println(BrightBlack + " ↳ Cluster is not healthy, aborting" + Reset)
		return nil
	}
	fmt.Println(BrightBlack + " ↳ Cluster is healthy" + Reset)

	fmt.Println("🚩 Setting cluster-wide OSD flags")
	flags := []string{"noout", "nodown", "nobackfill", "norecover", "norebalance", "pause"}
	for _, flag := range flags {
		if err := sshClient.SetOSDFlag(flag); err != nil {
			return fmt.Errorf("set flag: %w", err)
		}
		fmt.Println(BrightBlack + " ↳ " + flag + Reset)
	}

	// TODO: Bring down CephFS cluster

	// TODO: Stop MDS service

	// TODO: Stop RADOS Gateway services

	fmt.Println("💥 Stopping crash service")
	if err := sshClient.StopCephService("crash"); err != nil {
		return fmt.Errorf("stop service: %w", err)
	}

	fmt.Println("🗄️  Stopping OSDs")
	daemons, err := sshClient.CephStatusByDaemonType("osd")
	if err != nil {
		return fmt.Errorf("daemon status: %w", err)
	}
	for _, daemon := range daemons {
		name := daemon.Type + "." + daemon.ID

		if err := sshClient.StopCephDaemon(name); err != nil {
			return fmt.Errorf("stop daemon: %w", err)
		}

		fmt.Println(BrightBlack + " ↳ " + name + Reset)
	}

	fmt.Println("👀 Stopping monitors")
	daemons, err = sshClient.CephStatusByDaemonType("mon")
	if err != nil {
		return fmt.Errorf("daemon status: %w", err)
	}
	for _, daemon := range daemons {
		for _, node := range cfg.Ceph.Nodes {
			// Monitors can't be stopped using ceph orchestrator, so we must
			// ssh into the nodes and stop them using systemd services.
			if !strings.EqualFold(daemon.Host, node.Name) {
				continue
			}

			nodeSSHClient, err := ssh.New(node)
			if err != nil {
				return fmt.Errorf("ssh: %w", err)
			}
			defer nodeSSHClient.Close()

			name := daemon.Type + "." + daemon.ID

			if err := nodeSSHClient.StopSystemdService(fmt.Sprintf("ceph-%s@%s", fsid, name)); err != nil {
				return fmt.Errorf("stop service: %w", err)
			}

			fmt.Println(BrightBlack + " ↳ " + name + Reset)
		}
	}

	fmt.Println("⚡️ Scheduling power off in 1 minute")
	for _, node := range cfg.Ceph.Nodes {
		nodeSSHClient, err := ssh.New(node)
		if err != nil {
			return fmt.Errorf("ssh: %w", err)
		}
		defer nodeSSHClient.Close()

		if err := nodeSSHClient.Shutdown(); err != nil {
			return fmt.Errorf("shutdown: %w", err)
		}

		fmt.Println(BrightBlack + " ↳ " + node.Name + Reset)
	}

	fmt.Println("✅ All done!")

	return nil
}
