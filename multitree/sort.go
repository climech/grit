package multitree

import "sort"

// sortNodesByID sorts the nodes in-place by their ID field (ascending).
func sortNodesByID(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		return nodes[i].ID < nodes[j].ID
	})
}
