package multitree

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fatih/color"
)

func (n *Node) checkbox() string {
	switch n.Status() {
	case TaskStatusCompleted:
		return "[x]"
	case TaskStatusInProgress:
		return "[~]"
	case TaskStatusInactive:
		return "[ ]"
	default:
		panic("invalid node status")
	}
}

// String returns a basic string representation of the node. Color is
// automatically disabled when in non-tty output mode.
func (n *Node) String() string {
	var id string
	if n.Alias == "" {
		id = fmt.Sprintf("(%d)", n.ID)
	} else {
		id = fmt.Sprintf("(%d:%s)", n.ID, n.Alias)
	}

	// Change accent color for descendants of the current date node.
	accent := color.New(color.FgCyan).SprintFunc()
	for _, r := range n.Roots() {
		if r.Name == time.Now().Format("2006-01-02") {
			accent = color.New(color.FgYellow).SprintFunc()
			break
		}
	}

	// Highlight root node.
	name := n.Name
	if len(n.parents) == 0 {
		bold := color.New(color.Bold).SprintFunc()
		name = bold(name)
	}

	return fmt.Sprintf("%s %s %s", accent(n.checkbox()), name, accent(id))
}

// StringTree returns a string representation of a tree rooted at node.
//
//     [~] Clean up the house (234)
//      ├──[~] Clean up the bedroom (235)
//      │   ├──[x] Clean up the desk (236)
//      │   ├──[ ] Clean up the floor (237)
//      │   └──[ ] Make the bed (238)
//      ├──[ ] Clean up the kitchen (239)
//      └──[ ] ...
//
func (n *Node) StringTree() string {
	var output string
	var traverse func(*Node, []bool)

	// cont determines if the line should be continued for each of the current
	// indent levels.
	traverse = func(n *Node, cont []bool) {
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

		for i, c := range n.children {
			if i != len(n.children)-1 {
				traverse(c, append(cont, true))
			} else {
				traverse(c, append(cont, false))
			}
		}
	}

	traverse(n, []bool{})
	return output
}

// StringNeighbors returns a string representation of the node's neighborhood,
// e.g.:
//
//   (45) ──┐
//  (150) ──┴── (123) ──┬── (124)
//                      └── (125)
//
func (n *Node) StringNeighbors() string {
	// Stringify the IDs.
	pids := make([]string, 0, len(n.parents))
	for _, p := range n.parents {
		pids = append(pids, fmt.Sprintf("(%d)", p.ID))
	}
	cids := make([]string, 0, len(n.children))
	for _, c := range n.children {
		cids = append(cids, fmt.Sprintf("(%d)", c.ID))
	}

	padleft := func(text string, n int) string {
		return strings.Repeat(" ", n-utf8.RuneCountInString(text)) + text
	}

	var output string
	maxlen := longestStringRuneCount(pids)
	indent := 0
	left := 0

	if length := len(pids); length == 0 {
		output += strings.Repeat(" ", indent)
		left = indent
	} else {
		spaces := strings.Repeat(" ", indent)
		if length == 1 {
			id := padleft(pids[0], maxlen)
			output += spaces + id + " ──── "
			left = indent + maxlen + 6
		} else {
			spaces := strings.Repeat(" ", indent)
			for i, p := range pids {
				id := padleft(p, maxlen)
				if i == 0 {
					output += spaces + id + " ───┐\n"
				} else if i != length-1 {
					output += spaces + id + " ───┤\n"
				} else {
					output += spaces + id + " ───┴─── "
				}
			}
			left = indent + maxlen + 9
		}
	}

	id := fmt.Sprintf("(%d)", n.ID)
	left += len(id)
	accent := color.New(color.FgCyan).SprintFunc()
	output += accent(id)

	if length := len(cids); length == 1 {
		output += " ──── " + cids[0] + "\n"
	} else if length > 1 {
		spaces := strings.Repeat(" ", left)
		for i, c := range cids {
			if i == 0 {
				output += " ───┬─── " + c + "\n"
			} else if i != length-1 {
				output += spaces + "    ├─── " + c + "\n"
			} else {
				output += spaces + "    └─── " + c + "\n"
			}
		}
	} else {
		output += "\n"
	}

	return output
}
