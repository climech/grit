package db

import (
	"database/sql"
	"container/list"
	"github.com/climech/grit/graph"
)

// GetGraph builds a graph using Breadth-First Search, and returns the
// requested node.
func (d *Database) GetGraph(nodeId int64) (*graph.Node, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return nil, err
	}
	node, err := txGetGraph(tx, nodeId)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if node == nil {
		tx.Rollback()
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return node, nil
}

// txGetGraph gets the graph as part of a transaction.
func txGetGraph(tx *sql.Tx, nodeId int64) (*graph.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_id = ?", nodeId)
	node, err := rowToNode(row)
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, nil
	}

	queue := list.New()
	queue.PushBack(node)
	visited := make(map[int64]*graph.Node)
	visited[node.Id] = node
	for {
		if elem := queue.Front(); elem == nil {
			break
		} else {
			queue.Remove(elem)
			current := elem.Value.(*graph.Node)

			predecessors, err := txGetPredecessors(tx, current.Id)
			if err != nil {
				return nil, err
			}
			for _, p := range predecessors {
				if _, ok := visited[p.Id]; !ok {
					current.AddPredecessor(p)
					visited[p.Id] = p
					queue.PushBack(p)
				} else {
					current.AddPredecessor(visited[p.Id])
				}
			}

			successors, err := txGetSuccessors(tx, current.Id)
			if err != nil {
				return nil, err
			}
			for _, s := range successors {
				if _, ok := visited[s.Id]; !ok {
					current.AddSuccessor(s)
					visited[s.Id] = s
					queue.PushBack(s)
				} else {
					current.AddSuccessor(visited[s.Id])
				}
			}
		}
	}

	return node, nil
}

// GetAdjacent gets both the predececessors and successors of the node.
func txGetAdjacent(tx *sql.Tx, id int64) ([]*graph.Node, error) {
	rows, err := tx.Query(
		"SELECT node_id, node_name, node_alias, node_checked FROM nodes " +
		"LEFT JOIN edges ON node_id = origin_id " +
		"WHERE dest_id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}
	nodes := rowsToNodes(rows)
	return nodes, nil
}

// txGetPredecessors gets nodes connected to the given node by incoming edges.
func txGetPredecessors(tx *sql.Tx, id int64) ([]*graph.Node, error) {
	rows, err := tx.Query(
		"SELECT node_id, node_name, node_alias, node_checked FROM nodes " +
		"LEFT JOIN edges ON node_id = origin_id " +
		"WHERE dest_id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}
	nodes := rowsToNodes(rows)
	return nodes, nil
}

// txGetSuccessors gets nodes connected to the given node by outgoing edges.
func txGetSuccessors(tx *sql.Tx, id int64) ([]*graph.Node, error) {
	rows, err := tx.Query(
		"SELECT node_id, node_name, node_alias, node_checked FROM nodes " +
		"LEFT JOIN edges ON node_id = dest_id " +
		"WHERE origin_id = ?",
		id,
	)
	if err != nil {
		return nil, err
	}
	nodes := rowsToNodes(rows)
	return nodes, nil
}
