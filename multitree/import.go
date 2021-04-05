package multitree

import (
	"bufio"
	"fmt"
	"io"
)

// ImportTrees reads a sequence of tab-indented lines and builds trees out of
// them. It returns pointers to the roots.
func ImportTrees(reader io.Reader) ([]*Node, error) {
	type stackItem struct {
		indent int
		node   *Node
	}

	var roots []*Node
	var stack []*stackItem
	scanner := bufio.NewScanner(reader)
	lineNum := 1

	for scanner.Scan() {
		indent, name := parseImportLine(scanner.Text())

		// Ignore empty lines.
		if name == "" {
			lineNum++
			continue
		}

		if err := ValidateNodeName(name); err != nil {
			return nil, fmt.Errorf("line %d: %v", lineNum, err)
		}

		// Backtrack until current indent > top stack indent.
		if len(stack) > 0 {
			top := len(stack) - 1
			for top >= 0 && stack[top].indent >= indent {
				stack = stack[:top] // pop
				top--
			}
		}

		var newNode *Node
		if len(stack) == 0 {
			newNode = NewNode(name)
			newNode.ID = 1
			roots = append(roots, newNode)
		} else {
			topNode := stack[len(stack)-1].node
			newNode = topNode.New(name)
			_ = LinkNodes(topNode, newNode)
		}

		stack = append(stack, &stackItem{indent: indent, node: newNode})
		lineNum++
	}

	return roots, nil
}

// parseImportLine returns the node's indent level and name.
func parseImportLine(line string) (int, string) {
	if len(line) == 0 {
		return 0, ""
	}
	var indent int
	for i := 0; i < len(line) && (line[i] == '\t' || line[i] == ' '); i++ {
		indent++
	}
	return indent, line[indent:]
}
