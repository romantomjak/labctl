package k8s

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Interact with kubernetes cluster",
	}

	cmd.AddCommand(dashboard)

	return cmd
}
