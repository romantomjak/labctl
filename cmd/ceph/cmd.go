package ceph

import "github.com/spf13/cobra"

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ceph",
		Short: "Interact with ceph cluster",
	}

	cmd.AddCommand(poweroff)

	return cmd
}
