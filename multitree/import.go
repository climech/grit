package multitree

import (
	"bufio"
	"fmt"
	"io"
)

// ImportNodes reads a sequence of tab-indented lines and builds trees out of
// them. It returns pointers to the roots.
func ImportNodes(reader io.Reader) ([]*Node, error) {
	type stackItem struct {
		indent int
		node   *Node
	}

	var roots []*Node
	var stack []*stackItem
	scanner := bufio.NewScanner(reader)
	lineNum := 1

	for scanner.Scan() {
		indent, name, err := parseImportLine(scanner.Text())
		if err != nil {
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
			newNode = New(name)
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

func parseImportLine(line string) (int, string, error) {
	var indent, i int
	for i < len(line) && line[i] == '\t' {
		indent++
		i++
	}
	name := line[indent:]
	if err := ValidateNodeName(name); err != nil {
		return 0, "", err
	}
	return indent, name, nil
}
