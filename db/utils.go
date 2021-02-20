package db

import (
	"database/sql"

	"github.com/climech/grit/graph"
	_ "github.com/mattn/go-sqlite3"
)

func copyCompletion(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cp := *value
	return &cp
}

type scannable interface {
	Scan(...interface{}) error
}

func scanToNode(s scannable) (*graph.Node, error) {
	node := &graph.Node{}
	var alias sql.NullString
	var completed sql.NullInt64
	err := s.Scan(&node.ID, &node.Name, &alias, &node.Created, &completed)
	if err == nil {
		node.Alias = alias.String
		if completed.Valid {
			node.Completed = &completed.Int64
		}
	}
	return node, err
}

func rowToNode(row *sql.Row) (*graph.Node, error) {
	node, err := scanToNode(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return node, nil
}

func rowsToNodes(rows *sql.Rows) []*graph.Node {
	defer rows.Close()
	var nodes []*graph.Node
	for rows.Next() {
		node, _ := scanToNode(rows)
		nodes = append(nodes, node)
	}
	return nodes
}

func rowToEdge(row *sql.Row) (*graph.Edge, error) {
	edge := &graph.Edge{}
	err := row.Scan(&edge.ID, &edge.OriginID, &edge.DestID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return edge, err
}

func rowsToEdges(rows *sql.Rows) []*graph.Edge {
	defer rows.Close()
	var edges []*graph.Edge
	for rows.Next() {
		edge := &graph.Edge{}
		rows.Scan(&edge.ID, &edge.OriginID, &edge.DestID)
		edges = append(edges, edge)
	}
	return edges
}
