package proxmox

import (
	"fmt"
	"strings"
)

type SortFunc func(a VirtualMachine, b VirtualMachine) int

func SortByNames(names ...string) SortFunc {
	priorityByName := make(map[string]int, len(names))
	for idx, name := range names {
		priorityByName[name] = idx
	}
	return func(a, b VirtualMachine) int {
		pa := priorityByName[a.Name]
		pb := priorityByName[b.Name]

		if pa < pb {
			return -1
		}
		if pa > pb {
			return 1
		}
		return 0
	}
}

func SortByIDs(ids ...string) SortFunc {
	priorityByID := make(map[string]int, len(ids))
	for idx, id := range ids {
		priorityByID[id] = idx
	}
	return func(a, b VirtualMachine) int {
		pa := priorityByID[fmt.Sprintf("%d", a.ID)]
		pb := priorityByID[fmt.Sprintf("%d", b.ID)]

		if pa < pb {
			return -1
		}
		if pa > pb {
			return 1
		}
		return 0
	}
}

func SortByTags(tags ...string) SortFunc {
	priorityByTags := make(map[string]int, len(tags))
	for idx, tag := range tags {
		parts := strings.Split(tag, ",")
		for _, part := range parts {
			priorityByTags[part] = idx
		}
	}
	return func(a, b VirtualMachine) int {
		pa := priorityByTags[a.Tags]
		pb := priorityByTags[b.Tags]

		if pa < pb {
			return -1
		}
		if pa > pb {
			return 1
		}
		return 0
	}
}
