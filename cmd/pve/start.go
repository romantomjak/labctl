package pve

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

var start = &cobra.Command{
	Use:   "start [flags] [args]",
	Short: "Start VMs",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		opts := &proxmox.ListOptions{
			Filters: []proxmox.Filter{
				proxmox.FilterIsVM(),
			},
		}

		switch {
		case flagVMIDs:
			opts.Filters = append(opts.Filters, proxmox.FilterByIDs(args...))
			opts.SortFunc = proxmox.SortByIDs(args...)
		case flagTags:
			opts.Filters = append(opts.Filters, proxmox.FilterByTags(args...))
			opts.SortFunc = proxmox.SortByTags(args...)
		default:
			opts.Filters = append(opts.Filters, proxmox.FilterByNames(args...))
			opts.SortFunc = proxmox.SortByNames(args...)
		}

		vms, err := proxmox.ListVMs(ctx, opts)
		if err != nil {
			return err
		}

		if len(vms) == 0 {
			fmt.Println("No VMs matched the specified arguments 💔")
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
