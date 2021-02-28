package multitree

import (
	"fmt"
	"unicode/utf8"
)

func nodesInclude(nodes []*Node, node *Node) bool {
	for _, n := range nodes {
		if n.ID == node.ID {
			return true
		}
	}
	return false
}

func removeNode(nodes []*Node, node *Node) ([]*Node, error) {
	index := -1
	for i, n := range nodes {
		if n.ID == node.ID {
			index = i
			break
		}
	}
	if index == -1 {
		return nil, fmt.Errorf("node was not found")
	}
	return append(nodes[:index], nodes[index+1:]...), nil
}

func copyCompletion(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cp := *value
	return &cp
}

func longestStringRuneCount(slice []string) int {
	var max, count int
	for _, s := range slice {
		count = utf8.RuneCountInString(s)
		if count > max {
			max = count
		}
	}
	return max
}
