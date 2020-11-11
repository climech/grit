package graph

import (
	"fmt"
	"time"
	"errors"
	"strings"
	"unicode/utf8"
	"container/list"
	"github.com/fatih/color"
)

type TaskStatus int

const (
	TaskStatusCompleted TaskStatus = iota
	TaskStatusInProgress
	TaskStatusInactive
)

func (s TaskStatus) String() string {
	switch s {
	case TaskStatusCompleted:
		return "completed"
	case TaskStatusInProgress:
		return "in progress"
	case TaskStatusInactive:
		return "inactive"
	default:
		panic("invalid task status")
	}
}

// tricolor holds the search state of a graph node.
type tricolor int

// The three possible search states, see:
// http://www.personal.kent.edu/~rmuhamma/Algorithms/MyAlgorithms/GraphAlgor/depthSearch.htm
const (
	sswhite tricolor = iota // undiscovered
	ssgray                  // discovered, but not finished
	ssblack                 // finished
)

type Node struct {
	Id int64
	Name string
	Alias string
	Comment string
	Checked bool
	Predecessors []*Node
	Successors []*Node
}

func NewNode(name string) *Node {
	return &Node{Name: name}
}

func (n *Node) Status() TaskStatus {
	if n.Checked {
		return TaskStatusCompleted
	} else if len(n.Successors) > 0 {
		checked := false
		n.EachAfter(func(current *Node, _ int) {
			if checked { return }
			if current.Checked {
				checked = true
			}
		})
		if checked {
			return TaskStatusInProgress
		}
	}
	return TaskStatusInactive
}

// Copy returns a deep copy of the graph.
func (n *Node) Copy() *Node {
	nodes := n.GetAll()
	newNodes := make(map[int64]*Node)

	// Copy the nodes, ignore the edges.
	for _, src := range nodes {
		newNodes[src.Id] = &Node{Id: src.Id, Name: src.Name, Checked: src.Checked}
	}

	// Connect the nodes.
	for _, src := range nodes {
		predecessors := make([]*Node, len(src.Predecessors))
		for i, pre := range src.Predecessors {
			predecessors[i] = newNodes[pre.Id]
		}
		successors := make([]*Node, len(src.Successors))
		for i, succ := range src.Successors {
			successors[i] = newNodes[succ.Id]
		}
		newNodes[src.Id].Predecessors = predecessors
		newNodes[src.Id].Successors = successors
	}

	return newNodes[n.Id]
}

// HasForwardEdge returns true if at least one forward edge is found in the
// graph.
func (n *Node) HasForwardEdge() (found bool) {
	var dfs func(*Node)
	color := make(map[*Node]tricolor)

	// To allow for cross edges, record the current source node for each
	// node at the time of discovery.
	var currentSource *Node 
	source := make(map[*Node]*Node)

	// Set the initial state for each node.
	n.Each(func(current *Node){
		color[current] = sswhite
		source[current] = nil
	})

	// Using depth-first search to traverse the graph.
	dfs = func(current *Node) {
		color[current] = ssgray
		source[current] = currentSource
		for _, succ := range current.Successors {
			switch color[succ] {
			case ssblack: // forward (or cross) edge
				if source[current] == source[succ] {
					found = true
				}
			case sswhite: // tree edge
				dfs(succ)
			}
			if found { return }
		}
		color[current] = ssblack
	}

	sources := n.Roots()
	if len(sources) > 0 {
		for _, node := range n.Roots() {
			if c, _ := color[node]; c == sswhite {
				currentSource = node
				dfs(node)
				if found { break }
			}
		}
	} else {
		// Cyclic graph -- take any node, and leave currentSource as nil.
		for _, node := range n.GetAll() {
			if c, _ := color[node]; c == sswhite {
				dfs(node)
				if found { break }
			}
		}
	}

	return found
}

// HasBackEdge returns true if at least one back edge is found in the graph.
func (n *Node) HasBackEdge() (found bool) {
	var dfs func(*Node)
	color := make(map[*Node]tricolor)

	// Set the initial state for each node.
	n.Each(func(cur *Node){
		color[cur] = sswhite
	})

	// Using depth-first search to traverse the graph.
	dfs = func(current *Node) {
		color[current] = ssgray
		for _, succ := range current.Successors {
			switch color[succ] {
			case ssgray: // back edge
				found = true
			case sswhite: // tree edge
				dfs(succ)
			}
			if found { return }
		}
		color[current] = ssblack
	}

	for _, node := range n.GetAll() {
		if c, _ := color[node]; c == sswhite {
			dfs(node)
			if found { break }
		}
	}

	return found
}

// String returns a basic string representation of the node. Color is
// automatically disabled when in non-tty output mode.
func (n *Node) String() string {
	var checkbox string

	switch n.Status() {
	case TaskStatusCompleted:  checkbox = "[x]"
	case TaskStatusInProgress: checkbox = "[~]"
	case TaskStatusInactive:   checkbox = "[ ]"
	}

	var id string
	if n.Alias == "" {
		id = fmt.Sprintf("(%d)", n.Id)
	} else {
		id = fmt.Sprintf("(%d:%s)", n.Id, n.Alias)
	}
	name := n.Name
	accent := color.New(color.FgCyan).SprintFunc()

	// Change accent color for nodes associated with the current d-node.
	if roots := n.RootsBefore(); len(roots) != 0 {
		for _, r := range roots {
			if r.Name == time.Now().Format("2006-01-02") {
				accent = color.New(color.FgYellow).SprintFunc()
				break
			}
		}
	}

	// Highlight root node.
	if len(n.Predecessors) == 0 {
		bold := color.New(color.Bold).SprintFunc()
		name = bold(name)
	}
	checkbox = accent(checkbox)
	id = accent(id)

	return fmt.Sprintf("%s %s %s", checkbox, name, id)
}

// TreeString returns a string representation of a tree rooted at node.
//
//     [~] Clean up the house (234)
//      ├──[~] Clean up the bedroom (235)
//      │   ├──[x] Clean up the desk (236)
//      │   ├──[ ] Clean up the floor (237)
//      │   └──[ ] Make the bed (238)
//      ├──[ ] Clean up the kitchen (239)
//      └──[ ] ...
//
func (n *Node) TreeString() string {
	visited := make(map[*Node]bool)
	var output string
	var traverse func(*Node, []bool)

	// cont determines if the line should be continued for each of the current
	// indent levels.
	traverse = func(n *Node, cont []bool) {
		if _, ok := visited[n]; ok { return }
		visited[n] = true

		var indent string
		if len(cont) > 0 {
			for _, cont := range cont[:len(cont)-1] {
				if cont {
					indent += " │  "
				} else {
					indent += "    "
				}
			}
			if cont[len(cont)-1] {
				indent += " ├──"
			} else {
				indent += " └──"
			}
		}

		output += fmt.Sprintf("%s%s\n", indent, n)

		for i, succ := range n.Successors {
			if i != len(n.Successors) - 1 {
				traverse(succ, append(cont, true))
			} else {
				traverse(succ, append(cont, false))
			}
		}
	}

	traverse(n, []bool{})
	return output
}

// EdgeString returns a string representation of the node's incoming and
// outgoing edges, e.g.:
//
//   (45) ──┐
//  (150) ──┴── (123) ──┬── (124)
//                      ├── (125)
//                      ├── (126)
//                      └── (127)
//
func (n *Node) EdgeString() string {
	// Stringify the IDs.
	predecessors := make([]string, len(n.Predecessors))
	successors := make([]string, len(n.Successors))
	for i, p := range n.Predecessors {
		predecessors[i] = fmt.Sprintf("(%d)", p.Id)
	}
	for i, s := range n.Successors {
		successors[i] = fmt.Sprintf("(%d)", s.Id)
	}

	padleft := func(text string, n int) string {
		return strings.Repeat(" ", n - utf8.RuneCountInString(text)) + text
	}

	var output string
	maxpre := maxRuneCountInStrings(predecessors)
	indent := 0
	left := 0
	
	if length := len(predecessors); length == 0 {
		output += strings.Repeat(" ", indent)
		left = indent
	} else {
		spaces := strings.Repeat(" ", indent)
		if length == 1 {
			id := padleft(predecessors[0], maxpre)
			output += spaces + id + " ──── "
			left = indent + maxpre + 6
		} else {
			spaces := strings.Repeat(" ", indent)
			for i, pre := range predecessors {
				id := padleft(pre, maxpre)
				if i == 0 {
					output += spaces + id + " ───┐\n"
				} else if i != length - 1 {
					output += spaces + id + " ───┤\n"
				} else {
					output += spaces + id + " ───┴─── "
				}
			}
			left = indent + maxpre + 9
		}
	}

	id := fmt.Sprintf("(%d)", n.Id)
	left += len(id)
	accent := color.New(color.FgCyan).SprintFunc()
	output += accent(id)

	if length := len(successors); length == 1 {
		output += " ──── " + successors[0] + "\n"
	} else if length > 1 {
		spaces := strings.Repeat(" ", left)
		for i, succ := range successors {
			if i == 0 {
				output +=          " ───┬─── " + succ + "\n"
			} else if i != length - 1 {
				output += spaces + "    ├─── " + succ + "\n"
			} else {
				output += spaces + "    └─── " + succ + "\n"
			}
		}
	} else {
		output += "\n"
	}

	return output
}

func (n *Node) HasSuccessor(node *Node) bool {
	for _, s := range n.Successors {
		if node.Id != 0 && s.Id == node.Id {
			return true
		}
	}
	return false
}

func (n *Node) HasPredecessor(node *Node) bool {
	for _, p := range n.Predecessors {
		if node.Id != 0 && p.Id == node.Id {
			return true
		}
	}
	return false
}

func (n *Node) AddSuccessor(node *Node) error {
	if n.HasSuccessor(node) {
		return errors.New("successor already exists")
	}
	n.Successors = append(n.Successors, node)
	node.Predecessors = append(node.Predecessors, n)
	return nil
}

func (n *Node) AddPredecessor(node *Node) error {
	if n.HasPredecessor(node) {
		return errors.New("predecessor already exists")
	}
	n.Predecessors = append(n.Predecessors, node)
	node.Successors = append(node.Successors, n)
	return nil
}

func (n *Node) RemoveSuccessor(successor *Node) error {
	index := -1
	for i, succ := range n.Successors {
		if succ.Id == successor.Id {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("node not in successors")
	}
	n.Successors = append(n.Successors[:index], n.Successors[index+1:]...)
	return nil
}

func (n *Node) RemovePredecessor(predecessor *Node) error {
	index := -1
	for i, pre := range n.Predecessors {
		if pre.Id == predecessor.Id {
			index = i
			break
		}
	}
	if index == -1 {
		return errors.New("node not in predecessors")
	}
	n.Predecessors = append(n.Predecessors[:index], n.Predecessors[index+1:]...)
	return nil
}

// Each calls f for each node in the graph.
func (n *Node) Each(f func (*Node)) {
	if len(n.Predecessors) + len(n.Successors) == 0 {
		f(n)
		return
	}

	// Use breadth-first search to traverse the graph; ignore direction.
	visited := make(map[int64]*Node)
	visited[n.Id] = n
	queue := list.New()
	queue.PushBack(n)
	for {
		if elem := queue.Front(); elem == nil {
			break
		} else {
			queue.Remove(elem)
			current := elem.Value.(*Node)
			f(current)
			adjacent := append(current.Predecessors, current.Successors...)
			for _, adj := range adjacent {
				if _, ok := visited[adj.Id]; !ok {
					visited[adj.Id] = adj
					queue.PushBack(adj)
				}
			}
		}
	}
}

// GetAll returns all graph nodes in a slice.
func (n *Node) GetAll() []*Node {
	var nodes []*Node
	n.Each(func(cur *Node) {
		nodes = append(nodes, cur)
	})
	return nodes
}

// Get returns the node reachable from n which matches the given ID, or nil if
// the node cannot be found.
func (n *Node) Get(id int64) *Node {
	if n.Id == id {
		return n
	}

	// Use breadth-first search to traverse the graph; ignore direction.
	visited := make(map[int64]*Node)
	visited[n.Id] = n
	queue := list.New()
	queue.PushBack(n)
	for {
		if elem := queue.Front(); elem == nil {
			break
		} else {
			queue.Remove(elem)
			current := elem.Value.(*Node)
			if current.Id == id {
				return current
			}
			adjacent := append(current.Predecessors, current.Successors...)
			for _, adj := range adjacent {
				if _, ok := visited[adj.Id]; !ok {
					visited[adj.Id] = adj
					queue.PushBack(adj)
				}
			}
		}
	}

	return nil
}

// GetByName returns the first node reachable from n which matches the given
// name, or nil if the node cannot be found.
func (n *Node) GetByName(name string) *Node {
	if n.Name == name {
		return n
	}

	// Use breadth-first search to traverse the graph; ignore direction.
	visited := make(map[int64]*Node)
	visited[n.Id] = n
	queue := list.New()
	queue.PushBack(n)
	for {
		if elem := queue.Front(); elem == nil {
			break
		} else {
			queue.Remove(elem)
			current := elem.Value.(*Node)
			if current.Name == name {
				return current
			}
			adjacent := append(current.Predecessors, current.Successors...)
			for _, adj := range adjacent {
				if _, ok := visited[adj.Id]; !ok {
					visited[adj.Id] = adj
					queue.PushBack(adj)
				}
			}
		}
	}

	return nil
}

// EachAfter calls f for each direct and indirect successor of node.
func (n *Node) EachAfter(f func(*Node, int)) {
	if len(n.Successors) == 0 {
		return
	}
	visited := make(map[int64]bool)
	var traverse func(*Node, int)
	traverse = func(current *Node, distance int) {
		if _, ok := visited[current.Id]; ok {
			return
		}
		if current != n {
			f(current, distance)
		}
		visited[current.Id] = true
		
		for _, succ := range current.Successors {
			traverse(succ, distance+1)
		}
	}
	traverse(n, 0)
}

// NodesAfter returns all direct and indirect successors of the node.
func (n *Node) NodesAfter() []*Node {
	var nodes []*Node
	n.EachAfter(func(n *Node, _ int) {
		nodes = append(nodes, n)
	})
	return nodes
}

// EachBefore calls f for each node in the node's path coming before the node.
func (n *Node) EachBefore(f func(*Node, int)) {
	if len(n.Predecessors) == 0 {
		return
	}

	visited := make(map[int64]bool)
	var traverse func(*Node, int)
	traverse = func(current *Node, distance int) {
		if _, ok := visited[current.Id]; ok {
			return
		}
		if current != n {
			f(current, distance)
		}
		visited[current.Id] = true
		
		for _, pre := range current.Predecessors {
			traverse(pre, distance-1)
		}
	}
	traverse(n, 0)
}

// Tree returns a new tree rooted at node, built by disconnecting the node from
// its predecessors and traversing the graph along the outgoing edges.
// Each node is visited only once to prevent cycles.
func (n *Node) Tree() *Node {
	visited := make(map[*Node]bool)
	var traverse func(*Node) []*Node

	traverse = func(n *Node) []*Node {
		visited[n] = true

		// Create a new node, copy info from n.
		children := make([]*Node, 0, len(n.Successors))
		for _, succ := range n.Successors {
			if _, ok := visited[succ]; ok {
				continue
			}
			child := *succ
			child.Predecessors = []*Node{n}
			child.Successors = traverse(&child)
			children = append(children, &child)
		}
		return children
	}

	root := *n
	root.Predecessors = nil
	root.Successors = traverse(n)

	return &root
}

// Roots returns the roots for the entire graph.
func (n *Node) Roots() (roots []*Node) {
	n.Each(func(cur *Node) {
		if len(cur.Predecessors) == 0 {
			roots = append(roots, cur)
		}
	})
	return
}

// RootsBefore returns roots reachable by backtracking from the node.
func (n *Node) RootsBefore() []*Node {
	if len(n.Predecessors) == 0 {
		return []*Node{n}
	}
	var roots []*Node
	n.EachBefore(func(cur *Node, _ int) {
		if len(cur.Predecessors) == 0 {
			roots = append(roots, cur)
		}
	})
	return roots
}

func ValidateNodeName(name string) error {
	if len(name) == 0 {
		return errors.New("invalid node name (empty name)")
	}
	if len(name) > 100 {
		return errors.New("invalid node name (name too long)")
	}
	return nil
}

func ValidateDateNodeName(name string) error {
	if len(name) == 0 {
		return errors.New("invalid date node name: empty string")
	}
	if _, err := time.Parse("2006-01-02", name); err != nil {
		return errors.New("invalid date node name")
	}
	return nil
}

func ValidateNodeAlias(alias string) error {
	// TODO
	if len(alias) == 0 {
		return errors.New("invalid alias (empty)")
	}
	if len(alias) > 100 {
		return errors.New("invalid alias (too long)")
	}
	return nil
}
