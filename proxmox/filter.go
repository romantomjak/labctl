package proxmox

import (
	"fmt"
	"strings"
)

type Filter func(vm VirtualMachine) bool

func FilterIsVM() Filter {
	return func(vm VirtualMachine) bool {
		return !vm.IsTemplate
	}
}

func FilterByNames(names ...string) Filter {
	requestedNames := make(map[string]struct{}, len(names))
	for _, name := range names {
		requestedNames[name] = struct{}{}
	}
	return func(vm VirtualMachine) bool {
		_, ok := requestedNames[vm.Name]
		return ok
	}
}

func FilterByTags(tags ...string) Filter {
	requestedTags := make(map[string]struct{}, len(tags))
	for _, tag := range tags {
		requestedTags[tag] = struct{}{}
	}
	return func(vm VirtualMachine) bool {
		vmTags := strings.Split(vm.Tags, ",")
		for _, tag := range vmTags {
			_, ok := requestedTags[tag]
			return ok
		}
		return false
	}
}

func FilterByIDs(ids ...string) Filter {
	requestedIDs := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		requestedIDs[id] = struct{}{}
	}
	return func(vm VirtualMachine) bool {
		_, ok := requestedIDs[fmt.Sprintf("%d", vm.ID)]
		return ok
	}
}
