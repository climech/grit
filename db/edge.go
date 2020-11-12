package db

import (
	"fmt"
	"errors"
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

func txCreateEdge(tx *sql.Tx, originId, destId int64) (int64, error) {
	result, err := tx.Exec(
		"INSERT INTO edges (origin_id, dest_id) VALUES (?, ?)",
		originId,
		destId,
	)
	if err != nil {
		tx.Rollback()
		return -1, err
	}
	return result.LastInsertId()
}

func (d *Database) CreateEdge(originId, destId int64) (int64, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return -1, err
	}

	if yes, err := txIsDateNode(tx, destId); err != nil {
		tx.Rollback()
		return -1, err
	} else if yes {
		return -1, errors.New("cannot link to a date node")
	}

	edgeId, err := txCreateEdge(tx, originId, destId)
	if err != nil {
		tx.Rollback()
		return -1, err
	}
	node, err := txGetGraph(tx, destId)
	if err != nil {
		tx.Rollback()
		return -1, err
	}
	if node.HasBackEdge() {
		tx.Rollback()
		return -1, errors.New("back edges not allowed")
	}
	if node.HasForwardEdge() {
		tx.Rollback()
		return -1, errors.New("forward edges not allowed")
	}
	if err := txFixBefore(tx, node); err != nil {
		tx.Rollback()
		return -1, err
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}
	return edgeId, nil
}

// CreateEdgeFromDateNode creates an edge where d-node is the origin. If d-node
// doesn't exist, it's created automatically.
func (d *Database) CreateEdgeFromDateNode(date string, destId int64) (int64, error) {
	if graph.ValidateDateNodeName(date) != nil {
		panic("invalid d-node name")
	}

	tx, err := d.BeginTx()
	if err != nil {
		return -1, err
	}

	// Check if destination is another d-node.
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_id = ?", destId)
	destNode, err := rowToNode(row)
	if err != nil {
		tx.Rollback()
		return -1, err
	}
	if destNode != nil && graph.ValidateDateNodeName(destNode.Name) == nil {
		tx.Rollback()
		return -1, errors.New("Destination can't be a d-node")
	}

	// Create d-node if it doesn't exist.
	_, err = tx.Exec(
		"INSERT INTO nodes (node_name) SELECT ? " +
		"WHERE NOT EXISTS(SELECT 1 FROM nodes WHERE node_name = ?)",
		date,
		date,
	)
	if err != nil {
		tx.Rollback()
		return -1, err
	}

	r, err := tx.Exec(
		`INSERT INTO edges (origin_id, dest_id) VALUES (` +
			`(SELECT node_id FROM nodes WHERE node_name = ?), ` +
			`?` +
		`)`,
		date,
		destId,
	)
	if err != nil {
		tx.Rollback()
		return -1, err
	}
	id, _ := r.LastInsertId()

	// Fix d-node completion, if necessary.
	node, err := txGetGraph(tx, destId)
	if err != nil {
		tx.Rollback()
		return -1, err
	}
	dnode := node.GetByName(date)
	if dnode.Checked && !node.Checked {
		dnode.Checked = false
		if err := txUpdateNode(tx, dnode); err != nil {
			tx.Rollback()
			return -1, err
		}
	}

	if err := tx.Commit(); err != nil {
		return -1, err
	}
	return id, nil
}

func txDeleteEdgeByEndpoints(tx *sql.Tx, originId, destId int64) error {
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
	return nil
}

func (d *Database) DeleteEdgeByEndpoints(originId, destId int64) error {
	tx, err := d.BeginTx()
	if err != nil {
		return err
	}
	if err := txDeleteEdgeByEndpoints(tx, originId, destId); err != nil {
		tx.Rollback()
		return err
	}
	// Fix node status for ancestor nodes.
	node, err := txGetGraph(tx, originId)
	if err != nil {
		tx.Rollback()
		return err
	}
	if err := txFixBefore(tx, node); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}
