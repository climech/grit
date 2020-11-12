package db

import (
	"database/sql"
	"container/list"
	"github.com/climech/grit/graph"
)

// getAdjacent gets both the direct predececessors and successors of the node.
func getAdjacent(tx *sql.Tx, id int64) ([]*graph.Node, error) {
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

// getPredecessors gets nodes connected to the given node by incoming edges.
func getPredecessors(tx *sql.Tx, id int64) ([]*graph.Node, error) {
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

// getSuccessors gets nodes connected to the given node by outgoing edges.
func getSuccessors(tx *sql.Tx, id int64) ([]*graph.Node, error) {
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

func getGraph(tx *sql.Tx, nodeId int64) (*graph.Node, error) {
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

			predecessors, err := getPredecessors(tx, current.Id)
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

			successors, err := getSuccessors(tx, current.Id)
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

// GetGraph builds a graph using Breadth-First Search, and returns the
// requested node as part of the graph.
func (d *Database) GetGraph(nodeId int64) (*graph.Node, error) {
	var node *graph.Node
	err := d.execTxFunc(func(tx *sql.Tx) error {
		n, err := getGraph(tx, nodeId)
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
