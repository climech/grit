package graph

import (
	"io"
	"bufio"
)

// ImportNodes reads a sequence of tab-indented lines and builds a forest out
// of them.
func ImportNodes(reader io.Reader) (roots []*Node) {
	type stackItem struct {
		indent uint
		node *Node
	}
	var stack []*stackItem
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		indent, name := parseImportLine(scanner.Text())
		newNode := NewNode(name)

		// Backtrack until current indent > top stack indent.
		if len(stack) > 0 {
			top := len(stack) - 1
			for top >= 0 && stack[top].indent >= indent {
				stack = stack[:top] // Pop
				top--
			}
		}
		// Make the new node a successor to the node on top of the stack.
		if len(stack) == 0 {
			roots = append(roots, newNode)
		} else {
			stack[len(stack) - 1].node.AddSuccessor(newNode)
		}
		stack = append(stack, &stackItem{indent: indent, node: newNode}) // Push
	}

	return roots
}

func parseImportLine(line string) (indent uint, nodeName string) {
	for _, r := range line {
		if r == '\t' {
			indent++
		} else {
			break
		}
	}
	nodeName = line[indent:]
	return indent, nodeName
}
