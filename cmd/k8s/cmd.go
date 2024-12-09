package k8s

import (
	"github.com/spf13/cobra"
)

var (
	flagCompressBackup bool
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "Interact with kubernetes cluster",
	}

	cmd.AddCommand(dashboard)

	backup.Flags().BoolVar(&flagCompressBackup, "compress", false, "compress backup with zstd")
	cmd.AddCommand(backup)

	return cmd
}
