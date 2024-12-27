package ceph

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var bootExample = strings.Trim(`
  # Boot the whole cluster
  labctl ceph boot
`, "\n")

var boot = &cobra.Command{
	Use:          "boot [flags]",
	Short:        "Start ceph nodes",
	Example:      bootExample,
	SilenceUsage: true,
	RunE:         bootCommandFunc,
}

func bootCommandFunc(cmd *cobra.Command, args []string) error {
	fmt.Println("✅ All done!")

	return nil
}
