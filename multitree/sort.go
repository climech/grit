package multitree

import (
	"sort"

	"github.com/climech/naturalsort"
)

// SortNodesByID sorts a slice of nodes in-place by Node.ID in ascending order.
func SortNodesByID(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
}

// SortNodesByName uses natural sort to sort a slice of nodes in-place by
// Node.Name in ascending order.
func SortNodesByName(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return naturalsort.Compare(nodes[i].Name, nodes[j].Name)
	})
}

// SortNodesByStatus sorts a the nodes by their status in the order of
// Inactive -> InProgress -> Completed
func SortNodesByStatus(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		if nodes[i].Status() == nodes[j].Status() {
			return naturalsort.Compare(nodes[i].Name, nodes[j].Name)
		}
		return nodes[i].Status() > nodes[j].Status()
	})
}
