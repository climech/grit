package db

import (
	"database/sql"

	"github.com/climech/grit/multitree"
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

func scanToNode(s scannable) (*multitree.Node, error) {
	node := &multitree.Node{}
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

func rowToNode(row *sql.Row) (*multitree.Node, error) {
	node, err := scanToNode(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return node, nil
}

func rowsToNodes(rows *sql.Rows) []*multitree.Node {
	defer rows.Close()
	var nodes []*multitree.Node
	for rows.Next() {
		node, _ := scanToNode(rows)
		nodes = append(nodes, node)
	}
	return nodes
}

func rowToLink(row *sql.Row) (*multitree.Link, error) {
	link := &multitree.Link{}
	err := row.Scan(&link.ID, &link.OriginID, &link.DestID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return link, err
}

func rowsToLinks(rows *sql.Rows) []*multitree.Link {
	defer rows.Close()
	var links []*multitree.Link
	for rows.Next() {
		link := &multitree.Link{}
		err := rows.Scan(&link.ID, &link.OriginID, &link.DestID)
		if err != nil {
			panic(err)
		}
		links = append(links, link)
	}
	return links
}

func filterDateNodes(nodes []*multitree.Node) []*multitree.Node {
	var filtered []*multitree.Node
	for _, n := range nodes {
		if n.IsDateNode() {
			filtered = append(filtered, n)
		}
	}
	return filtered
}
