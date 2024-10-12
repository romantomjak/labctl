package proxmox

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/luthermonson/go-proxmox"
)

type VirtualMachine struct {
	ID         uint64
	CPU        float64
	Disk       uint64
	Mem        uint64
	Name       string
	Node       string
	Status     string
	Storage    string
	Tags       string
	Uptime     uint64
	IsTemplate bool
}

type ListOptions struct {
	Filters  []Filter
	SortFunc func(a VirtualMachine, b VirtualMachine) int
}

func ListVMs(ctx context.Context, opt *ListOptions) ([]VirtualMachine, error) {
	cluster, err := client.Cluster(ctx)
	if err != nil {
		return nil, err
	}

	rs, err := cluster.Resources(ctx, "vm")
	if err != nil {
		return nil, err
	}

	vms := make([]VirtualMachine, 0, len(rs))
	for _, r := range rs {
		vm := VirtualMachine{
			ID:         r.VMID,
			CPU:        r.CPU,
			Disk:       r.Disk,
			Mem:        r.Mem,
			Name:       r.Name,
			Node:       r.Node,
			Status:     r.Status,
			Storage:    r.Status,
			Tags:       r.Tags,
			Uptime:     r.Uptime,
			IsTemplate: r.Template == 1,
		}

		if opt == nil {
			vms = append(vms, vm)
			continue
		}

		matchedAllFilters := true
		for _, f := range opt.Filters {
			if !f(vm) {
				matchedAllFilters = false
				break
			}
		}

		if matchedAllFilters {
			vms = append(vms, vm)
		}
	}

	if opt != nil && opt.SortFunc != nil {
		slices.SortFunc(vms, opt.SortFunc)
	}

	return vms, nil
}

func StartVM(ctx context.Context, vm VirtualMachine) error {
	var upid proxmox.UPID
	if err := client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/start", vm.Node, vm.ID), nil, &upid); err != nil {
		return err
	}

	task := proxmox.NewTask(upid, client)

	status, completed, err := task.WaitForCompleteStatus(ctx, 30, 1)
	if err != nil {
		return err
	}

	if !completed {
		return fmt.Errorf("timed out: %s", task.ExitStatus)
	}

	if !status && !strings.Contains(task.ExitStatus, "already running") {
		return fmt.Errorf("failed: %s", task.ExitStatus)
	}

	return nil
}

func IsRunning(ctx context.Context, vm VirtualMachine) (bool, error) {
	var pvm proxmox.VirtualMachine
	if err := client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/current", vm.Node, vm.ID), &pvm); err != nil {
		return false, err
	}
	return pvm.Status == proxmox.StatusVirtualMachineRunning, nil
}

func IsStopped(ctx context.Context, vm VirtualMachine) (bool, error) {
	var pvm proxmox.VirtualMachine
	if err := client.Get(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/current", vm.Node, vm.ID), &pvm); err != nil {
		return false, err
	}
	return pvm.Status == proxmox.StatusVirtualMachineStopped, nil
}

func StopVM(ctx context.Context, vm VirtualMachine) error {
	var upid proxmox.UPID
	if err := client.Post(ctx, fmt.Sprintf("/nodes/%s/qemu/%d/status/stop", vm.Node, vm.ID), nil, &upid); err != nil {
		return err
	}

	task := proxmox.NewTask(upid, client)

	status, completed, err := task.WaitForCompleteStatus(ctx, 30, 1)
	if err != nil {
		return err
	}

	if !completed {
		return fmt.Errorf("timed out: %s", task.ExitStatus)
	}

	if !status && !strings.Contains(task.ExitStatus, "already stopped") {
		return fmt.Errorf("failed: %s", task.ExitStatus)
	}

	return nil
}
