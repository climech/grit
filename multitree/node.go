package multitree

import (
	"fmt"
	"time"
)

type Node struct {
	ID   int64
	Name string

	// Alias is an optional secondary identifier of the node.
	Alias string

	// Created holds the Unix timestamp for the node's creation time.
	Created int64

	// Completed points to the Unix timestamp of when the node was marked as
	// completed, or nil, if the node hasn't been completed yet.
	Completed *int64

	parents  []*Node
	children []*Node
}

func NewNode(name string) *Node {
	return &Node{Name: name}
}

// nextID returns one more than the highest ID in the
// multitree.
func (n *Node) nextID() int64 {
	var max int64
	for _, node := range n.All() {
		if node.ID > max {
			max = node.ID
		}
	}
	return max + 1
}

// New creates a new node with the ID set to 1 more than the highest ID in the
// multitree.
func (n *Node) New(name string) *Node {
	return &Node{ID: n.nextID(), Name: name}
}

func (n *Node) IsCompleted() bool {
	return n.Completed != nil
}

// IsCompletedOnDate returns true if n was completed on date given as a string
// in the format "YYYY-MM-DD". The start of day is determined by offset, e.g. if
// offset is 4, the day starts at 4 A.M.
func (n *Node) IsCompletedOnDate(date string, offset int) bool {
	t := n.TimeCompleted()

	if !t.IsZero() {
		start, err := time.Parse("2006-01-02", date)
		if err != nil {
			panic(err)
		}
		start = start.Local().Add(time.Duration(offset) * time.Hour)
		end := start.Add(24 * time.Hour)

		if t.Equal(start) || (t.After(start) && t.Before(end)) {
			return true
		}
	}

	return false
}

func (n *Node) IsInProgress() bool {
	if n.IsCompleted() {
		return false
	}
	for _, d := range n.Descendants() {
		if d.IsCompleted() {
			return true
		}
	}
	return false
}

func (n *Node) IsInactive() bool {
	return !n.IsCompleted() && !n.IsInProgress()
}

func (n *Node) IsRoot() bool {
	return len(n.parents) == 0
}

func (n *Node) IsDateNode() bool {
	if n.IsRoot() && ValidateDateNodeName(n.Name) == nil {
		return true
	}
	return false
}

// TimeCompleted returns the task completion time as local time.Time.
func (n *Node) TimeCompleted() time.Time {
	var t time.Time
	if n.Completed != nil {
		t = time.Unix(*n.Completed, 0)
	}
	return t
}

func (n *Node) Children() []*Node {
	return n.children
}

func (n *Node) Parents() []*Node {
	return n.parents
}

// Ancestors returns a flat list of the node's ancestors.
func (n *Node) Ancestors() []*Node {
	var nodes []*Node
	n.TraverseAncestors(func(current *Node, _ func()) {
		nodes = append(nodes, current)
	})
	if len(nodes) > 0 {
		return nodes[1:]
	}
	return nodes
}

// Descendants returns a flat list of the node's descendants.
func (n *Node) Descendants() []*Node {
	var nodes []*Node
	n.TraverseDescendants(func(current *Node, _ func()) {
		nodes = append(nodes, current)
	})
	if len(nodes) > 0 {
		return nodes[1:]
	}
	return nodes
}

// All returns a flat list of all nodes in the multitree. The nodes are sorted
// by ID in ascending order.
func (n *Node) All() []*Node {
	var nodes []*Node
	n.DepthFirstSearchUndirected(func(cur *Node, ss SearchState, _ func()) {
		if ss == SearchStateWhite {
			nodes = append(nodes, cur)
		}
	})
	SortNodesByID(nodes)
	return nodes
}

// Roots returns the local roots found by following the node's ancestors all
// the way up. The nodes are sorted by ID in ascending order.
func (n *Node) Roots() []*Node {
	var roots []*Node
	n.TraverseAncestors(func(current *Node, _ func()) {
		if current.IsRoot() {
			roots = append(roots, current)
		}
	})
	SortNodesByID(roots)
	return roots
}

// RootsAll returns a list of all roots in the multitree, not just the roots
// local to the node. The nodes are sorted by ID in ascending order.
func (n *Node) RootsAll() []*Node {
	var roots []*Node
	for _, node := range n.All() {
		if node.IsRoot() {
			roots = append(roots, node)
		}
	}
	SortNodesByID(roots)
	return roots
}

// Roots returns the local roots found by following the node's descendants all
// the way down. The nodes are sorted by ID in ascending order.
func (n *Node) IsLeaf() bool {
	return len(n.children) == 0
}

func (n *Node) HasChildren() bool {
	return len(n.children) != 0
}

func (n *Node) HasParents() bool {
	return len(n.parents) != 0
}

func (n *Node) Leaves() []*Node {
	var leaves []*Node
	n.TraverseDescendants(func(current *Node, _ func()) {
		if current.IsLeaf() {
			leaves = append(leaves, current)
		}
	})
	SortNodesByID(leaves)
	return leaves
}

// LeavesAll returns a list of all leaves in the multitree, not just the leaves
// local to the node. The nodes are sorted by ID in ascending order.
func (n *Node) LeavesAll() []*Node {
	var leaves []*Node
	for _, node := range n.All() {
		if node.IsLeaf() {
			leaves = append(leaves, node)
		}
	}
	SortNodesByID(leaves)
	return leaves
}

// Tree returns a tree rooted at n, induced by following the children all the
// way down to the leaves. The tree nodes are guaranteed to only have one
// parent.
func (n *Node) Tree() *Node {
	root := n.DeepCopy()
	root.parents = nil
	root.TraverseDescendants(func(current *Node, _ func()) {
		for _, child := range current.children {
			child.parents = []*Node{current}
		}
	})
	return root
}

// Copy returns a shallow, unlinked copy of the node.
func (n *Node) Copy() *Node {
	return &Node{
		ID:        n.ID,
		Name:      n.Name,
		Alias:     n.Alias,
		Created:   n.Created,
		Completed: copyCompletion(n.Completed),
	}
}

// Copy returns a deep copy of the entire multitree that the node belongs to.
func (n *Node) DeepCopy() *Node {
	nodes := n.All()
	nodesByID := make(map[int64]*Node)

	// Copy the nodes into the map, ignoring the links for now.
	for _, src := range nodes {
		nodesByID[src.ID] = src.Copy()
	}

	// Create the links between the new nodes.
	for _, src := range nodes {
		nodesByID[src.ID].parents = make([]*Node, len(src.parents))
		for i, p := range src.parents {
			nodesByID[src.ID].parents[i] = nodesByID[p.ID]
		}
		nodesByID[src.ID].children = make([]*Node, len(src.children))
		for i, c := range src.children {
			nodesByID[src.ID].children[i] = nodesByID[c.ID]
		}
	}

	return nodesByID[n.ID]
}

func (n *Node) HasChild(node *Node) bool {
	return nodesInclude(n.children, node)
}

func (n *Node) HasParent(node *Node) bool {
	return nodesInclude(n.parents, node)
}

// hasBackEdge returns true if at least one back edge is found in the directed
// graph.
func (n *Node) hasBackEdge() (found bool) {
	roots := n.RootsAll()
	if len(roots) == 0 {
		// Cyclic graph -- choose any node as our "root".
		roots = append(roots, n)
	}
	for _, r := range roots {
		r.DepthFirstSearch(func(cur *Node, ss SearchState, stop func()) {
			if ss == SearchStateGray {
				found = true
				stop()
			}
		})
		if found {
			break
		}
	}
	return found
}

// hasDiamond returns true if at least one diamond is found in the graph, which
// is here assumed to be a DAG. (A diamond occurs when two directed paths
// diverge from a node and meet again at some other node.)
func (n *Node) hasDiamond() (found bool) {
	roots := n.RootsAll()
	if len(roots) == 0 {
		panic("cyclic graph passed to Node.hasDiamond")
	}
	for _, r := range roots {
		r.DepthFirstSearch(func(cur *Node, ss SearchState, stop func()) {
			if ss == SearchStateBlack {
				found = true
				stop()
			}
		})
		if found {
			break
		}
	}
	return found
}

// Get returns the first node matching the ID, or nil, if no match is found.
func (n *Node) Get(id int64) *Node {
	for _, node := range n.All() {
		if node.ID == id {
			return node
		}
	}
	return nil
}

// Get returns the first node matching the name, or nil, if no match is found.
func (n *Node) GetByName(name string) *Node {
	for _, node := range n.All() {
		if node.Name == name {
			return node
		}
	}
	return nil
}

// Get returns the first node matching the alias, or nil, if no match is found.
func (n *Node) GetByAlias(alias string) *Node {
	for _, node := range n.All() {
		if node.Alias == alias {
			return node
		}
	}
	return nil
}

// validateNewLink creates deep copies of the nodes' graphs, connects the copied
// nodes, and checks if the resulting graph is a valid multitree.
func validateNewLink(origin, dest *Node) error {
	if origin.ID == 0 || dest.ID == 0 {
		panic("link endpoints must have IDs")
	}
	if origin.HasChild(dest) != dest.HasParent(origin) {
		panic("parent/child out of sync")
	}
	if origin.HasChild(dest) {
		return fmt.Errorf("link already exists")
	}
	if ValidateDateNodeName(dest.Name) == nil {
		return fmt.Errorf("cannot unroot date node")
	}

	parent := origin.DeepCopy()
	child := dest.DeepCopy()
	parent.children = append(parent.children, child)
	child.parents = append(child.parents, parent)

	if parent.hasBackEdge() {
		return fmt.Errorf("cycles are not allowed")
	}
	if parent.hasDiamond() {
		return fmt.Errorf("diamonds are not allowed")
	}

	return nil
}

// LinkNodes creates a directed link from origin to dest, provided that the
// resulting graph will be a valid multitree. It returns an error otherwise.
// The given nodes are modified only if no error occurs.
func LinkNodes(origin, dest *Node) error {
	if err := validateNewLink(origin, dest); err != nil {
		return err
	}
	origin.children = append(origin.children, dest)
	dest.parents = append(dest.parents, origin)
	return nil
}

// LinkNodes removes an existing directed link between origin and dest. It
// returns an error if the link doesn't exist.
func UnlinkNodes(origin, dest *Node) error {
	if origin.HasChild(dest) != dest.HasParent(origin) {
		panic("parent/child out of sync")
	}
	if !origin.HasChild(dest) {
		return fmt.Errorf("link does not exist")
	}
	origin.children, _ = removeNode(origin.children, dest)
	dest.parents, _ = removeNode(dest.parents, origin)
	return nil
}
