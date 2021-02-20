package graph

import (
	"testing"
	"fmt"
)

// newCyclicGraph creates the following graph:
//
//   (1) ──┐
//   (2) ──┴─> (3) ──┬─> (4)
//    ^              └─> (5)
//    └───────────────────┘
// 
func newCyclicGraph() *Node {
	edges := [][]int64{{1, 3}, {2, 3}, {3, 4}, {4, 5}, {5, 2}}
	nodesById := make(map[int64]*Node)
	for i := int64(1); i <= 5; i++ {
		nodesById[i] = &Node{
			ID: i,
			Name: fmt.Sprintf("node %d", i),
		}
	}
	for _, edge := range edges {
		origin := nodesById[edge[0]]
		target := nodesById[edge[1]]
		origin.AddSuccessor(target)
		target.AddPredecessor(origin)
	}
	return nodesById[1]
}

func TestTree(t *testing.T) {
	rootId := int64(3)
	node := newCyclicGraph().Get(rootId)
	tree := node.Tree()

	want := []int64{2, 3, 4, 5}
	doNotWant := []int64{1}

	for _, id := range want {
		if tree.Get(id) == nil {
			t.Errorf("node %d missing from the tree", id)
		}
	}

	for _, id := range doNotWant {
		if tree.Get(id) != nil {
			t.Errorf("node %d should not exist in the tree", id)
		}
	}

	if len(tree.Get(rootId).Predecessors) != 0 {
		t.Errorf("tree root has predecessors")
	}
}
