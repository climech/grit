package multitree

import (
	"sort"

	"facette.io/natsort"
)

// sortNodesByID sorts a slice of nodes in-place by Node.ID in ascending order.
func sortNodesByID(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
}

// sortNodesByName uses natural sort to sort a slice of nodes in-place by
// Node.Name in ascending order.
func sortNodesByName(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return natsort.Compare(nodes[i].Name, nodes[j].Name)
	})
}
