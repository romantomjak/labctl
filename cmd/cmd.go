package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/cmd/ps"
	"github.com/romantomjak/labctl/cmd/start"
	"github.com/romantomjak/labctl/cmd/stop"
)

func Execute() {
	cmd := &cobra.Command{
		Use:   "labctl",
		Short: "labctl controls roman’s homelab",
	}

	cmd.AddCommand(ps.Command())
	cmd.AddCommand(start.Command())
	cmd.AddCommand(stop.Command())

	if err := cmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
