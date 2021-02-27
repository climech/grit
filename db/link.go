package db

import (
	"database/sql"
	"fmt"

	"github.com/climech/grit/multitree"

	_ "github.com/mattn/go-sqlite3"
)

func (d *Database) GetLink(linkID int64) (*multitree.Link, error) {
	row := d.DB.QueryRow("SELECT * FROM links WHERE link_id = ?", linkID)
	return rowToLink(row)
}

func (d *Database) GetLinkByEndpoints(originID, destID int64) (*multitree.Link, error) {
	row := d.DB.QueryRow(
		"SELECT * FROM links WHERE origin_id = ? AND dest_id = ?", originID, destID)
	return rowToLink(row)
}

// GetLinksByNodeID gets the node's incoming and outcoming links.
func (d *Database) GetLinksByNodeID(nodeID int64) ([]*multitree.Link, error) {
	rows, err := d.DB.Query(
		"SELECT * FROM links WHERE origin_id = ? OR dest_id = ?",
		nodeID,
		nodeID,
	)
	if err != nil {
		return nil, err
	}
	return rowsToLinks(rows), nil
}

func insertLink(tx *sql.Tx, originID, destID int64) (int64, error) {
	r, err := tx.Exec("INSERT INTO links (origin_id, dest_id) VALUES (?, ?)",
		originID, destID)
	if err != nil {
		return 0, err
	}
	return r.LastInsertId()
}

func createLink(tx *sql.Tx, originID, destID int64) (int64, error) {
	// Validate link.
	origin, err := getGraph(tx, originID)
	if err != nil {
		return 0, err
	}
	dest, err := getGraph(tx, destID)
	if err != nil {
		return 0, err
	}
	if err := multitree.LinkNodes(origin, dest); err != nil {
		return 0, err
	}

	linkID, err := insertLink(tx, originID, destID)
	if err != nil {
		return 0, err
	}
	if err := backpropCompletion(tx, origin); err != nil {
		return 0, err
	}

	return linkID, nil
}

func (d *Database) CreateLink(originID, destID int64) (int64, error) {
	var linkID int64
	txf := func(tx *sql.Tx) error {
		id, err := createLink(tx, originID, destID)
		if err != nil {
			return err
		}
		linkID = id
		return nil
	}
	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return linkID, nil
}

// CreateLinkFromDateNode atomically creates an link with date node as the
// origin. Date node is automatically created if it doesn't exist.
func (d *Database) CreateLinkFromDateNode(date string, destID int64) (int64, error) {
	if err := multitree.ValidateDateNodeName(date); err != nil {
		panic(err)
	}

	var linkID int64
	txf := func(tx *sql.Tx) error {
		originID, err := createDateNodeIfNotExists(tx, date)
		if err != nil {
			return err
		}
		id, err := createLink(tx, originID, destID)
		if err != nil {
			return err
		}
		linkID = id
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return linkID, nil
}

func deleteLinkByEndpoints(tx *sql.Tx, originID, destID int64) error {
	r, err := tx.Exec("DELETE FROM links WHERE origin_id = ? AND dest_id = ?",
		originID, destID)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("link (%d) -> (%d) does not exist", originID, destID)
	}
	return nil
}

func (d *Database) DeleteLinkByEndpoints(originID, destID int64) error {
	return d.execTxFunc(func(tx *sql.Tx) error {
		if err := deleteLinkByEndpoints(tx, originID, destID); err != nil {
			return err
		}

		origin, err := getGraph(tx, originID)
		if err != nil {
			return err
		}

		if origin.IsDateNode() && len(origin.Children()) == 0 {
			deleteNode(tx, originID)
		} else {
			if err := backpropCompletion(tx, origin); err != nil {
				return err
			}
		}

		return nil
	})
}
