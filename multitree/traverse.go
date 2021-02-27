package multitree

type searchState int

const (
	searchStateWhite searchState = iota // undiscovered
	searchStateGray                     // discovered, but not finished
	searchStateBlack                    // finished
)

// traverseDescendants calls f for each descendant of the node. The function is
// passed a pointer to the current node and a stop function that can be called
// to exit early.
func (n *Node) traverseDescenants(f func(*Node, func())) {
	var stop bool
	stopFunc := func() {
		stop = true
	}
	var traverse func(*Node)
	traverse = func(cur *Node) {
		f(cur, stopFunc)
		for _, c := range cur.children {
			if stop {
				break
			}
			traverse(c)
		}
	}
	traverse(n)
}

// traverseAncestors calls f for each ancestor of the node. The function is
// passed a pointer to the current node and a stop function that can be called
// to exit early.
func (n *Node) traverseAncestors(f func(*Node, func())) {
	var stop bool
	stopFunc := func() {
		stop = true
	}
	var traverse func(*Node)
	traverse = func(cur *Node) {
		f(cur, stopFunc)
		for _, p := range cur.parents {
			if stop {
				break
			}
			traverse(p)
		}
	}
	traverse(n)
}

func dfs(node *Node, f func(*Node, searchState, func()), directed bool) {
	var stop bool
	stopFunc := func() {
		stop = true
	}

	stateByID := make(map[int64]searchState) // not in map => white
	var traverse func(*Node)

	traverse = func(current *Node) {
		stateByID[current.ID] = searchStateGray

		reachable := current.children
		if !directed {
			reachable = append(current.parents, current.children...)
		}

		for _, r := range reachable {
			if stop {
				break
			}
			if ss, ok := stateByID[r.ID]; ok {
				f(r, ss, stopFunc)
			} else {
				f(r, searchStateWhite, stopFunc)
				traverse(r)
			}
		}

		stateByID[current.ID] = searchStateBlack
	}

	f(node, searchStateWhite, stopFunc)
	traverse(node)
}

// depthFirstSearch traverses the graph directionally starting from the node,
// calling f on each step forward. The function is passed a pointer to the
// current node, the node's search state, and a stop function that can be called
// to exit early.
func (n *Node) depthFirstSearch(f func(*Node, searchState, func())) {
	dfs(n, f, true)
}

// depthFirstSearchAll traverses the entire graph, ignoring the direction,
// advancing through parents and children alike. It starts from n and calls f on
// each step forward. The function is passed a pointer to the current node, the
// node's search state, and a stop function that can be called to exit early.
func (n *Node) depthFirstSearchAll(f func(*Node, searchState, func())) {
	dfs(n, f, false)
}
