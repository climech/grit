package db

import (
	"container/list"
	"database/sql"
	"fmt"

	"github.com/climech/grit/multitree"
)

// getAdjacent gets parents and children of the node.
func getAdjacent(tx *sql.Tx, id int64) ([]*multitree.Node, error) {
	rows, err := tx.Query(
		"SELECT node_id, node_name, node_alias, node_created, node_completed "+
			"FROM nodes LEFT JOIN links ON node_id = origin_id WHERE dest_id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}
	nodes := rowsToNodes(rows)
	return nodes, nil
}

// getParents gets nodes connected to the given node by incoming links.
func getParents(tx *sql.Tx, id int64) ([]*multitree.Node, error) {
	rows, err := tx.Query(
		"SELECT node_id, node_name, node_alias, node_created, node_completed "+
			"FROM nodes LEFT JOIN links ON node_id = origin_id WHERE dest_id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}
	nodes := rowsToNodes(rows)
	return nodes, nil
}

// getChildren gets nodes connected to the given node by outgoing links.
func getChildren(tx *sql.Tx, id int64) ([]*multitree.Node, error) {
	rows, err := tx.Query(
		"SELECT node_id, node_name, node_alias, node_created, node_completed "+
			"FROM nodes LEFT JOIN links ON node_id = dest_id WHERE origin_id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}
	nodes := rowsToNodes(rows)
	return nodes, nil
}

func getGraph(tx *sql.Tx, nodeID int64) (*multitree.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_id = ?", nodeID)
	node, err := rowToNode(row)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, nil
	}

	queue := list.New()
	queue.PushBack(node)
	visited := make(map[int64]*multitree.Node)
	visited[node.ID] = node

	linkNodes := func(origin, dest *multitree.Node) {
		if origin.HasChild(dest) {
			return
		}
		if err := multitree.LinkNodes(origin, dest); err != nil {
			panic(fmt.Sprintf("invalid multitree link in DB (%d->%d): %v",
				origin.ID, dest.ID, err))
		}
	}

	for {
		if elem := queue.Front(); elem == nil {
			break
		} else {
			queue.Remove(elem)
			current := elem.Value.(*multitree.Node)

			parents, err := getParents(tx, current.ID)
			if err != nil {
				return nil, err
			}
			for _, p := range parents {
				if _, ok := visited[p.ID]; !ok {
					linkNodes(p, current)
					visited[p.ID] = p
					queue.PushBack(p)
				} else {
					linkNodes(visited[p.ID], current)
				}
			}

			children, err := getChildren(tx, current.ID)
			if err != nil {
				return nil, err
			}
			for _, c := range children {
				if _, ok := visited[c.ID]; !ok {
					linkNodes(current, c)
					visited[c.ID] = c
					queue.PushBack(c)
				} else {
					linkNodes(current, visited[c.ID])
				}
			}
		}
	}

	return node, nil
}

// GetGraph builds a multitree using Breadth-First Search, and returns the
// requested node as part of the multitree.
func (d *Database) GetGraph(nodeID int64) (*multitree.Node, error) {
	var node *multitree.Node
	err := d.execTxFunc(func(tx *sql.Tx) error {
		n, err := getGraph(tx, nodeID)
		if err != nil {
			return err
		}
		node = n
		return nil
	})
	if err != nil {
		return nil, err
	}
	return node, nil
}
