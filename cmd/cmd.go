package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/cmd/k8s"
	"github.com/romantomjak/labctl/cmd/pve"
)

func Execute() {
	cmd := &cobra.Command{
		Use:   "labctl",
		Short: "labctl controls roman’s homelab",
	}

	cmd.AddCommand(pve.Command())
	cmd.AddCommand(k8s.Command())

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
