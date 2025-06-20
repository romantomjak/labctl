package ceph

import "github.com/spf13/cobra"

var (
	flagAssumeYes bool
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ceph",
		Short: "Interact with ceph cluster",
	}

	cmd.AddCommand(boot)

	cmd.AddCommand(poweroff)

	install.Flags().BoolVarP(&flagAssumeYes, "assume-yes", "y", false, `assume "yes" as answer to all prompts`)
	cmd.AddCommand(install)

	return cmd
}
