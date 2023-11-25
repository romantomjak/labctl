package start

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

func Command() *cobra.Command {
	return &cobra.Command{
		Use:   "start [tags]",
		Short: "Start VMs with matching tags",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			vms, err := proxmox.ListVMsWithTags(ctx, args...)
			if err != nil {
				return err
			}

			if len(vms) == 0 {
				fmt.Println("No VMs matched the specified tags 💔")
				return nil
			}

			fmt.Println("🚦 Will start the VMs in the following order:")
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
				fmt.Println("🚀 Starting the VMs")
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
				defer cancel()
				iter.ForEach(vms, startVM(ctx, cmd))
			case "n":
				fmt.Println("🙅‍♀️ Aborted")
				return nil
			default:
				return fmt.Errorf("invalid input: %s", input)
			}

			return nil
		},
	}
}

func startVM(ctx context.Context, cmd *cobra.Command) func(*proxmox.VirtualMachine) {
	return func(vm *proxmox.VirtualMachine) {
		fmt.Fprintf(cmd.OutOrStdout(), "  - %s... ", vm.Name)

		isRunning, err := proxmox.IsRunning(ctx, *vm)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s ❌\n", err.Error())
			return
		}

		if isRunning {
			fmt.Fprintln(cmd.OutOrStdout(), "already running ✅")
			return
		}

		if err := proxmox.StartVM(ctx, *vm); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "%s ❌\n", err.Error())
			return
		}

		fmt.Fprintln(cmd.OutOrStdout(), "OK ✅")
	}
}
