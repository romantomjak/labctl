package pve

import "github.com/spf13/cobra"

var (
	flagVMIDs bool
	flagTags  bool
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pve",
		Short: "Interact with proxmox cluster",
	}

	ps.Flags().BoolVar(&flagTags, "tags", false, "")
	ps.Flags().BoolVar(&flagVMIDs, "ids", false, "")
	cmd.AddCommand(ps)

	start.Flags().BoolVar(&flagTags, "tags", false, "")
	start.Flags().BoolVar(&flagVMIDs, "ids", false, "")
	cmd.AddCommand(start)

	stop.Flags().BoolVar(&flagTags, "tags", false, "")
	stop.Flags().BoolVar(&flagVMIDs, "ids", false, "")
	cmd.AddCommand(stop)

	return cmd
}
