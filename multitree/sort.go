package multitree

import (
	"sort"

	"facette.io/natsort"
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
		return natsort.Compare(nodes[i].Name, nodes[j].Name)
	})
}
