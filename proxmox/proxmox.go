package proxmox

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/luthermonson/go-proxmox"
)

var client = proxmox.NewClient(fmt.Sprintf("%s/api2/json", os.Getenv("PROXMOX_ADDR")),
	proxmox.WithHTTPClient(&http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}),
	proxmox.WithCredentials(&proxmox.Credentials{
		Username: os.Getenv("PROXMOX_USER"),
		Password: os.Getenv("PROXMOX_PASSWORD"),
		Realm:    "pam",
	}),
)

type VirtualMachine struct {
	ID      uint64
	CPU     float64
	Disk    uint64
	Mem     uint64
	Name    string
	Node    string
	Status  string
	Storage string
	Tags    string
	Uptime  uint64
}

func ListVMs(ctx context.Context) ([]VirtualMachine, error) {
	return ListVMsWithTags(ctx)
}

func ListVMsWithTags(ctx context.Context, tags ...string) ([]VirtualMachine, error) {
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
		if r.Template == 1 {
			continue // ignore VM templates
		}

		vms = append(vms, VirtualMachine{
			ID:      r.VMID,
			CPU:     r.CPU,
			Disk:    r.Disk,
			Mem:     r.Mem,
			Name:    r.Name,
			Node:    r.Node,
			Status:  r.Status,
			Storage: r.Status,
			Tags:    r.Tags,
			Uptime:  r.Uptime,
		})
	}

	// Return everything if no tags were specified.
	if len(tags) == 0 {
		return vms, nil
	}

	// Build a lookup map to avoid looping over tag slice for each VM.
	requestedTags := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		requestedTags[tag] = struct{}{}
	}

	vmsByTags := make(map[string][]VirtualMachine, len(vms))
	for _, vm := range vms {
		_, ok := requestedTags[vm.Tags]
		if !ok {
			continue // VM does not have the requested tags
		}
		vmsByTags[vm.Tags] = append(vmsByTags[vm.Tags], vm)
	}

	sortedVMs := make([]VirtualMachine, 0, len(vms))
	for _, tag := range tags {
		sortedVMs = append(sortedVMs, vmsByTags[tag]...)
	}

	return sortedVMs, nil
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
