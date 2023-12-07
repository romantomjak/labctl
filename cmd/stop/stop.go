package stop

import (
	"bufio"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sourcegraph/conc/iter"
	"github.com/spf13/cobra"

	"github.com/romantomjak/labctl/proxmox"
)

var (
	flagVMIDs bool
	flagTags  bool
)

func Command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [flags] [args]",
		Short: "Stop VMs",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var (
				vms []proxmox.VirtualMachine
				err error
			)
			switch {
			case flagVMIDs:
				vms, err = proxmox.ListVMsWithIDs(ctx, args...)
				if err != nil {
					return err
				}
			case flagTags:
				vms, err = proxmox.ListVMsWithTags(ctx, args...)
				if err != nil {
					return err
				}
			default:
				vms, err = proxmox.ListVMsWithNames(ctx, args...)
				if err != nil {
					return err
				}
			}

			if len(vms) == 0 {
				fmt.Println("No VMs matched the specified arguments 💔")
				return nil
			}

			fmt.Println("🚦 Will stop the VMs in the following order:")
			for _, vm := range vms {
				fmt.Printf("  - %s\n", vm.Name)
			}

			fmt.Printf("❓ Do you want to continue? [y/n] ")
			reader := bufio.NewReader(cmd.InOrStdin())
			input, _, err := reader.ReadLine()
			if err != nil {
				return err
			}

			switch strings.ToLower(string(input)) {
			case "y":
				fmt.Println("🏁 Stopping the VMs")
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()
				iter.ForEach(vms, stopVM(ctx, cmd))
			case "n":
				fmt.Println("🙅‍♀️ Aborted")
				return nil
			default:
				return fmt.Errorf("invalid input: %s", input)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&flagTags, "tags", false, "")
	cmd.Flags().BoolVar(&flagVMIDs, "ids", false, "")

	return cmd
}

func stopVM(ctx context.Context, cmd *cobra.Command) func(*proxmox.VirtualMachine) {
	return func(vm *proxmox.VirtualMachine) {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s... ", vm.Name)

		isStopped, err := proxmox.IsStopped(ctx, *vm)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s ❌\n", err.Error())
			return
		}

		if isStopped {
			fmt.Fprintln(cmd.OutOrStdout(), "already stopped ✅")
			return
		}

		if err := proxmox.StopVM(ctx, *vm); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s ❌\n", err.Error())
			return
		}

		fmt.Fprintln(cmd.OutOrStdout(), "OK ✅")
	}
}
