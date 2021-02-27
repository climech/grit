package multitree

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestNode(id int64) *Node {
	created := time.Now().Unix() - 1000000 + id
	return &Node{
		ID:      id,
		Name:    "test",
		Created: created,
	}
}

func sprintfIDs(nodes []*Node) string {
	ids := make([]string, 0, len(nodes))
	for _, n := range nodes {
		ids = append(ids, fmt.Sprintf("%d", n.ID))
	}
	return fmt.Sprintf("[%s]", strings.Join(ids, ", "))
}

func linkOrFail(t *testing.T, origin, dest *Node) {
	if err := LinkNodes(origin, dest); err != nil {
		t.Fatalf("couldn't create link %d->%d: %v", origin.ID, dest.ID, err)
	}
}

func TestLinkNodes(t *testing.T) {
	{
		// Simple link.
		root1 := newTestNode(1)
		root2 := newTestNode(2)

		if err := LinkNodes(root1, root2); err != nil {
			t.Fatalf("couldn't make root a child of another root: %v", err)
		}
		if err := LinkNodes(root1, root2); err == nil {
			t.Error("added duplicate child")
		}

		if len(root1.Parents()) != 0 {
			t.Errorf("root1 shouldn't have any parents")
		}
		childrenWant := []*Node{root2}
		childrenGot := root1.Children()
		if !reflect.DeepEqual(childrenWant, childrenGot) {
			t.Errorf("invalid children after linking nodes; want %s, got %s",
				sprintfIDs(childrenWant), sprintfIDs(childrenGot))
		}

		if len(root2.Children()) != 0 {
			t.Errorf("root2 shouldn't have any children")
		}
		parentsWant := []*Node{root1}
		parentsGot := root2.Parents()
		if !reflect.DeepEqual(parentsWant, parentsGot) {
			t.Errorf("invalid parents after linking nodes; want %s, got %s",
				sprintfIDs(parentsWant), sprintfIDs(parentsGot))
		}
	}

	{
		// Cross link.
		//
		//   (0)   (2)
		//    |
		//   (1)
		//
		var nodes []*Node
		for i := 0; i < 4; i++ {
			nodes = append(nodes, newTestNode(int64(i+1)))
		}
		_ = LinkNodes(nodes[0], nodes[1])

		// (2) -> (0)
		if err := LinkNodes(nodes[2], nodes[1]); err != nil {
			t.Errorf("couldn't create a cross link: %v", err)
		}
	}

	{
		// Diamonds.
		//
		//      (0)
		//     /   \
		//   (1)   (2)
		//           \
		//           (3)
		//
		var nodes []*Node
		for i := 0; i < 4; i++ {
			nodes = append(nodes, newTestNode(int64(i+1)))
		}
		linkOrFail(t, nodes[0], nodes[1])
		linkOrFail(t, nodes[0], nodes[2])
		linkOrFail(t, nodes[2], nodes[3])

		// (0) -> (3)
		if err := LinkNodes(nodes[0], nodes[3]); err == nil {
			t.Errorf("a diamond slipped through LinkNodes: 1->2, 1->3, 3->4, 4->1")
		}
		// (3) -> (1)
		if err := LinkNodes(nodes[3], nodes[1]); err == nil {
			t.Errorf("a diamond slipped through LinkNodes: 1->2, 1->3, 3->4, 2->4")
		}
	}
}

func TestRoots(t *testing.T) {
	//
	//     (0)
	//     / \
	//   (1) (2)  (3)
	//       / \  / \
	//     (4) (5)  (6)
	//
	var nodes []*Node
	for i := 0; i < 7; i++ {
		nodes = append(nodes, newTestNode(int64(i+1)))
	}
	_ = LinkNodes(nodes[0], nodes[1])
	_ = LinkNodes(nodes[0], nodes[2])
	_ = LinkNodes(nodes[2], nodes[4])
	_ = LinkNodes(nodes[2], nodes[5])
	_ = LinkNodes(nodes[3], nodes[5])
	_ = LinkNodes(nodes[3], nodes[6])

	rootsWant := []*Node{nodes[3]}
	rootsGot := nodes[6].Roots()
	if !reflect.DeepEqual(rootsWant, rootsGot) {
		t.Errorf("invalid local roots; want %s, got %s",
			sprintfIDs(rootsWant), sprintfIDs(rootsGot))
	}

	rootsAllWant := []*Node{nodes[0], nodes[3]}
	rootsAllGot := nodes[6].RootsAll()
	if !reflect.DeepEqual(rootsAllWant, rootsAllGot) {
		t.Errorf("invalid global roots; want %s, got %s",
			sprintfIDs(rootsAllWant), sprintfIDs(rootsAllGot))
	}
}

func TestLeaves(t *testing.T) {
	//
	//     (0)
	//     / \
	//   (1) (2)  (3)
	//       / \  / \
	//     (4) (5)  (6)
	//
	var nodes []*Node
	for i := 0; i < 7; i++ {
		nodes = append(nodes, newTestNode(int64(i+1)))
	}
	_ = LinkNodes(nodes[0], nodes[1])
	_ = LinkNodes(nodes[0], nodes[2])
	_ = LinkNodes(nodes[2], nodes[4])
	_ = LinkNodes(nodes[2], nodes[5])
	_ = LinkNodes(nodes[3], nodes[5])
	_ = LinkNodes(nodes[3], nodes[6])

	leavesWant := []*Node{nodes[5], nodes[6]}
	leavesGot := nodes[3].Leaves()
	if !reflect.DeepEqual(leavesWant, leavesGot) {
		t.Errorf("invalid local leaves; want %s, got %s",
			sprintfIDs(leavesWant), sprintfIDs(leavesGot))
	}

	leavesAllWant := []*Node{nodes[1], nodes[4], nodes[5], nodes[6]}
	leavesAllGot := nodes[3].LeavesAll()
	if !reflect.DeepEqual(leavesAllWant, leavesAllGot) {
		t.Errorf("invalid global leaves; want %s, got %s",
			sprintfIDs(leavesAllWant), sprintfIDs(leavesAllGot))
	}
}
