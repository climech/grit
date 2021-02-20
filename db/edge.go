package db

import (
	"database/sql"
	"fmt"

	"github.com/climech/grit/graph"

	_ "github.com/mattn/go-sqlite3"
)

func (d *Database) GetEdge(edgeId int64) (*graph.Edge, error) {
	row := d.DB.QueryRow("SELECT * FROM edges WHERE edge_id = ?", edgeId)
	return rowToEdge(row)
}

func (d *Database) GetEdgeByEndpoints(originID, destID int64) (*graph.Edge, error) {
	row := d.DB.QueryRow(
		"SELECT * FROM edges WHERE origin_id = ? AND dest_id = ?",
		originID,
		destID,
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

func insertEdge(tx *sql.Tx, originID, destID int64) (int64, error) {
	r, err := tx.Exec(
		"INSERT INTO edges (origin_id, dest_id) VALUES (?, ?)",
		originID,
		destID,
	)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func createEdge(tx *sql.Tx, originID, destID int64) (int64, error) {
	if yes, err := isDateNode(tx, destID); err != nil {
		return 0, err
	} else if yes {
		return 0, fmt.Errorf("cannot point edge to a date node")
	}
	edgeId, err := insertEdge(tx, originID, destID)
	if err != nil {
		return 0, err
	}
	node, err := getGraph(tx, destID)
	if err != nil {
		return 0, err
	}
	if node.HasBackEdge() {
		return 0, fmt.Errorf("back edges are not allowed")
	}
	if node.HasForwardEdge() {
		return 0, fmt.Errorf("forward edges are not allowed")
	}
	if err := backpropCompletion(tx, node); err != nil {
		return 0, err
	}
	return edgeId, nil
}

func (d *Database) CreateEdge(originID, destID int64) (int64, error) {
	var edgeId int64
	txf := func(tx *sql.Tx) error {
		id, err := createEdge(tx, originID, destID)
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
func (d *Database) CreateEdgeFromDateNode(date string, destID int64) (int64, error) {
	if err := graph.ValidateDateNodeName(date); err != nil {
		panic(err)
	}

	var edgeId int64
	txf := func(tx *sql.Tx) error {
		originID, err := createDateNodeIfNotExists(tx, date)
		if err != nil {
			return err
		}
		id, err := createEdge(tx, originID, destID)
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

func deleteEdgeByEndpoints(tx *sql.Tx, originID, destID int64) error {
	r, err := tx.Exec("DELETE FROM edges WHERE origin_id = ? AND dest_id = ?",
		originID, destID)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("edge (%d) -> (%d) does not exist", originID, destID)
	}
	return nil
}

func (d *Database) DeleteEdgeByEndpoints(originID, destID int64) error {
	return d.execTxFunc(func(tx *sql.Tx) error {
		if err := deleteEdgeByEndpoints(tx, originID, destID); err != nil {
			return err
		}

		origin, err := getGraph(tx, originID)
		if err != nil {
			return err
		}

		if origin.IsDateNode() && len(origin.Successors) == 0 {
			deleteNode(tx, originID)
		} else {
			if err := backpropCompletion(tx, origin); err != nil {
				return err
			}
		}

		return nil
	})
}
