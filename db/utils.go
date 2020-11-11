package db

import (
	"database/sql"
	"github.com/climech/grit/graph"
	_ "github.com/mattn/go-sqlite3"
)

func rowToNode(row *sql.Row) (*graph.Node, error) {
	node := &graph.Node{}
	var nullableAlias sql.NullString
	err := row.Scan(&node.Id, &node.Name, &nullableAlias, &node.Checked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	node.Alias = nullableAlias.String
	return node, nil
}

func rowsToNodes(rows *sql.Rows) []*graph.Node {
	var nodes []*graph.Node
	for rows.Next() {
		node := &graph.Node{}
		var nullableAlias sql.NullString
		rows.Scan(&node.Id, &node.Name, &nullableAlias, &node.Checked)
		node.Alias = nullableAlias.String
		nodes = append(nodes, node)
	}
	rows.Close()
	return nodes
}

func rowToEdge(row *sql.Row) (*graph.Edge, error) {
	edge := &graph.Edge{}
	err := row.Scan(&edge.Id, &edge.OriginId, &edge.DestId)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return edge, err
}

func rowsToEdges(rows *sql.Rows) []*graph.Edge {
	var edges []*graph.Edge
	for rows.Next() {
		edge := &graph.Edge{}
		rows.Scan(&edge.Id, &edge.OriginId, &edge.DestId)
		edges = append(edges, edge)
	}
	rows.Close()
	return edges
}
