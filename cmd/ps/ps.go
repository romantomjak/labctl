package ps

import (
	"context"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/proxmox"
	"github.com/romantomjak/labctl/table"
)

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "ps",
		Short: "List VMs and their statuses",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			vms, err := proxmox.ListVMs(ctx, &proxmox.ListOptions{
				Filters: []proxmox.Filter{
					proxmox.FilterIsVM(),
				},
			})
			if err != nil {
				return err
			}

			if len(vms) == 0 {
				fmt.Println("No VMs are running at the moment 🙅‍♀️")
				return nil
			}

			t := table.New("ID", "NAME", "NODE", "STATUS", "UPTIME", "MEM", "CPU")
			for _, vm := range vms {
				t.AddRow(
					fmt.Sprintf("%d", vm.ID),
					vm.Name,
					vm.Node,
					vm.Status,
					time.Duration(vm.Uptime*uint64(time.Second)).String(),
					humanize.Bytes(vm.Mem),
					humanize.Ftoa(vm.CPU),
				)
			}
			if err := t.Print(cmd.OutOrStdout()); err != nil {
				return err
			}

			return nil
		},
	}
}
