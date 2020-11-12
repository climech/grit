package db

import (
	"fmt"
	"database/sql"

	"github.com/climech/grit/graph"

	_ "github.com/mattn/go-sqlite3"
)

func (d *Database) GetEdge(edgeId int64) (*graph.Edge, error) {
	row := d.DB.QueryRow( "SELECT * FROM edges WHERE edge_id = ?", edgeId)
	return rowToEdge(row)
}

func (d *Database) GetEdgeByEndpoints(originId, destId int64) (*graph.Edge, error) {
	row := d.DB.QueryRow(
		"SELECT * FROM edges WHERE origin_id = ? AND dest_id = ?",
		originId,
		destId,
	)
	return rowToEdge(row)
}

// GetEdgesByNodeId gets the node's incoming and outcoming edges.
func (d *Database) GetEdgesByNodeId(nodeId int64) ([]*graph.Edge, error) {
	rows, err := d.DB.Query(
		"SELECT * FROM edges WHERE origin_id = ? OR dest_id = ?",
		nodeId,
		nodeId,
	)
	if err != nil {
		return nil, err
	}
	return rowsToEdges(rows), nil
}

func insertEdge(tx *sql.Tx, originId, destId int64) (int64, error) {
	r, err := tx.Exec(
		"INSERT INTO edges (origin_id, dest_id) VALUES (?, ?)",
		originId,
		destId,
	)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func createEdge(tx *sql.Tx, originId, destId int64) (int64, error) {
	if yes, err := isDateNode(tx, destId); err != nil {
		return 0, err
	} else if yes {
		return 0, fmt.Errorf("cannot point edge to a date node")
	}
	edgeId, err := insertEdge(tx, originId, destId)
	if err != nil {
		return 0, err
	}
	node, err := getGraph(tx, destId)
	if err != nil {
		return 0, err
	}
	if node.HasBackEdge() {
		return 0, fmt.Errorf("back edges not allowed")
	}
	if node.HasForwardEdge() {
		return 0, fmt.Errorf("forward edges not allowed")
	}
	if err := fixStatusBefore(tx, node); err != nil {
		return 0, err
	}
	return edgeId, nil
}

func (d *Database) CreateEdge(originId, destId int64) (int64, error) {
	var edgeId int64
	txf := func(tx *sql.Tx) error {
		id, err := createEdge(tx, originId, destId)
		if err != nil {
			return err
		}
		edgeId = id
		return nil
	}
	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return edgeId, nil
}

// CreateEdgeFromDateNode atomically creates an edge with date node as the
// origin. Date node is automatically created if it doesn't exist.
func (d *Database) CreateEdgeFromDateNode(date string, destId int64) (int64, error) {
	if err := graph.ValidateDateNodeName(date); err != nil {
		panic(err)
	}

	var edgeId int64
	txf := func(tx *sql.Tx) error {
		originId, err := createDateNodeIfNotExists(tx, date)
		if err != nil {
			return err
		}
		id, err := createEdge(tx, originId, destId)
		if err != nil {
			return err
		}
		edgeId = id
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return edgeId, nil
}

func deleteEdgeByEndpoints(tx *sql.Tx, originId, destId int64) error {
	r, err := tx.Exec(
		"DELETE FROM edges WHERE origin_id = ? AND dest_id = ?",
		originId,
		destId,
	)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("edge (%d) -> (%d) does not exist", originId, destId)
	}
	node, err := getGraph(tx, originId)
	if err != nil {
		return err
	}
	if err := fixStatusBefore(tx, node); err != nil {
		return err
	}
	return nil
}

func (d *Database) DeleteEdgeByEndpoints(originId, destId int64) error {
	return d.execTxFunc(func(tx *sql.Tx) error {
		return deleteEdgeByEndpoints(tx, originId, destId)
	})
}
