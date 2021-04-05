package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/climech/grit/multitree"

	_ "github.com/mattn/go-sqlite3"
)

func getNode(tx *sql.Tx, id int64) (*multitree.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_id = ?", id)
	return rowToNode(row)
}

// GetNode returns the node with the given id, or nil if it doesn't exist.
func (d *Database) GetNode(id int64) (*multitree.Node, error) {
	var node *multitree.Node
	err := d.execTxFunc(func(tx *sql.Tx) error {
		n, err := getNode(tx, id)
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

func getNodeByName(tx *sql.Tx, name string) (*multitree.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_name = ?", name)
	return rowToNode(row)
}

// GetNode returns the node with the given name, or nil if it doesn't exist.
func (d *Database) GetNodeByName(name string) (*multitree.Node, error) {
	var node *multitree.Node
	err := d.execTxFunc(func(tx *sql.Tx) error {
		n, err := getNodeByName(tx, name)
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

func getNodeByAlias(tx *sql.Tx, alias string) (*multitree.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_alias = ?", alias)
	return rowToNode(row)
}

// GetNode returns the node with the given alias, or nil if it doesn't exist.
func (d *Database) GetNodeByAlias(alias string) (*multitree.Node, error) {
	var node *multitree.Node
	err := d.execTxFunc(func(tx *sql.Tx) error {
		n, err := getNodeByAlias(tx, alias)
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

// GetRoots returns a slice of nodes that have no predecessors.
func (d *Database) GetRoots() ([]*multitree.Node, error) {
	rows, err := d.DB.Query(
		"SELECT * FROM nodes " +
			"WHERE NOT EXISTS(SELECT * FROM links WHERE dest_id = node_id)",
	)
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func backpropCompletion(tx *sql.Tx, node *multitree.Node) error {
	var updateQueue []*multitree.Node
	var backprop func(*multitree.Node)

	backprop = func(n *multitree.Node) {
		allChildrenCompleted := true
		for _, c := range n.Children() {
			if !c.IsCompleted() {
				allChildrenCompleted = false
				break
			}
		}
		if n.IsCompleted() != allChildrenCompleted {
			if allChildrenCompleted {
				n.Completed = copyCompletion(n.Children()[0].Completed)
			} else {
				n.Completed = nil
			}
			updateQueue = append(updateQueue, n)
		}
		for _, p := range n.Parents() {
			backprop(p)
		}
	}

	for _, leaf := range node.Leaves() {
		for _, p := range leaf.Parents() {
			backprop(p)
		}
	}

	for _, node := range updateQueue {
		_, err := tx.Exec("UPDATE nodes SET node_completed = ? WHERE node_id = ?",
			node.Completed, node.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

func createNode(tx *sql.Tx, name string, parentID int64) (int64, error) {
	r, err := tx.Exec(`INSERT INTO nodes (node_name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	id, _ := r.LastInsertId()
	if parentID != 0 {
		if _, err := createLink(tx, parentID, id); err != nil {
			return 0, err
		}
		node, err := getGraph(tx, id)
		if err != nil {
			return 0, err
		}
		if err := backpropCompletion(tx, node); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// CreateNode creates a node and returns its ID. It updates the status of
// other nodes in the multitree if needed.
func (d *Database) CreateNode(name string, parentID int64) (int64, error) {
	var childID int64
	txf := func(tx *sql.Tx) error {
		id, err := createNode(tx, name, parentID)
		if err != nil {
			return err
		}
		childID = id
		return nil
	}
	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return childID, nil
}

func createDateNodeIfNotExists(tx *sql.Tx, date string) (int64, error) {
	if err := multitree.ValidateDateNodeName(date); err != nil {
		panic(err)
	}
	node, err := getNodeByName(tx, date)
	if err != nil {
		return 0, err
	}
	if node != nil {
		return node.ID, nil
	}
	return createNode(tx, date, 0)
}

// CreateChildOfDateNode atomically creates a node and links the date node to
// it. Date node is created if it doesn't exist.
func (d *Database) CreateChildOfDateNode(date, name string) (int64, error) {
	var childID int64

	txf := func(tx *sql.Tx) error {
		dateNodeID, err := createDateNodeIfNotExists(tx, date)
		if err != nil {
			return err
		}
		childID, err = createNode(tx, name, dateNodeID)
		if err != nil {
			return err
		}
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return childID, nil
}

func createTree(tx *sql.Tx, node *multitree.Node, parentID int64) (int64, error) {
	tree := node.Tree()
	var retErr error

	tree.TraverseDescendants(func(current *multitree.Node, stop func()) {
		pid := parentID
		parents := current.Parents()
		if len(parents) > 0 {
			pid = parents[0].ID
		}
		id, err := createNode(tx, current.Name, pid)
		if err != nil {
			retErr = err
			stop()
		} else {
			current.ID = id
		}
	})

	if retErr != nil {
		return 0, retErr
	}

	// Update ancestors, if any.
	if parentID != 0 {
		g, err := getGraph(tx, tree.ID)
		if err != nil {
			return 0, err
		}
		if err := backpropCompletion(tx, g); err != nil {
			return 0, err
		}
	}

	return tree.ID, nil
}

// CreateTree saves an entire tree in the database and returns the root ID. It
// updates the status of other nodes in the multitree to reflect the change.
func (d *Database) CreateTree(node *multitree.Node, parentID int64) (int64, error) {
	var rootID int64

	txf := func(tx *sql.Tx) error {
		id, err := createTree(tx, node, parentID)
		if err != nil {
			return err
		}
		rootID = id
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return rootID, nil
}

// CreateTreeAsChildOfDateNode atomically creates a tree and links the date node
// to its root. Date node is created if it doesn't exist.
func (d *Database) CreateTreeAsChildOfDateNode(date string, node *multitree.Node) (int64, error) {
	var rootID int64

	txf := func(tx *sql.Tx) error {
		dateNodeID, err := createDateNodeIfNotExists(tx, date)
		if err != nil {
			return err
		}
		id, err := createTree(tx, node, dateNodeID)
		if err != nil {
			return err
		}
		rootID = id
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return rootID, nil
}

func (d *Database) checkNode(nodeID int64, check bool) error {
	var value *int64
	if check {
		now := time.Now().Unix()
		value = &now
	}

	update := func(tx *sql.Tx, node *multitree.Node) error {
		r, err := tx.Exec("UPDATE nodes SET node_completed = ? WHERE node_id = ?",
			value, node.ID)
		if err != nil {
			return err
		}
		if count, _ := r.RowsAffected(); count == 0 {
			return fmt.Errorf("node does not exist")
		}
		node.Completed = copyCompletion(value)
		return nil
	}

	return d.execTxFunc(func(tx *sql.Tx) error {
		node, err := getGraph(tx, nodeID)
		if err != nil {
			return err
		}
		if node == nil {
			return fmt.Errorf("node does not exist")
		}
		// Update local root.
		if err := update(tx, node); err != nil {
			return err
		}
		// Update direct and indirect successors.
		for _, n := range node.Descendants() {
			if err := update(tx, n); err != nil {
				return err
			}
		}
		if err := backpropCompletion(tx, node); err != nil {
			return err
		}
		return nil
	})
}

// CheckNode marks the node as completed, along with all its direct and indirect
// successors. The rest of the multitree is updated to reflect the change.
func (d *Database) CheckNode(nodeID int64) error {
	return d.checkNode(nodeID, true)
}

// UncheckNode sets the node's status to inactive, along with all its direct
// and indirect successors. The rest of the multitree is updated to reflect the
// change.
func (d *Database) UncheckNode(nodeID int64) error {
	return d.checkNode(nodeID, false)
}

func (d *Database) RenameNode(nodeID int64, name string) error {
	r, err := d.DB.Exec("UPDATE nodes SET node_name = ? WHERE node_id = ?",
		name, nodeID)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func deleteNode(tx *sql.Tx, id int64) error {
	r, err := tx.Exec(`DELETE FROM nodes WHERE node_id = ?`, id)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("node does not exist")
	}
	return nil
}

// DeleteNode deletes a single node and propagates the change to the rest of the
// multitree. It returns the node's orphaned successors.
func (d *Database) DeleteNode(id int64) ([]*multitree.Node, error) {
	var orphans []*multitree.Node

	txf := func(tx *sql.Tx) error {
		node, err := getGraph(tx, id)
		if err != nil {
			return err
		}
		if node == nil {
			return fmt.Errorf("node does not exist")
		}

		if err := deleteNode(tx, id); err != nil {
			return err
		}

		// Auto-delete any empty date nodes.
		for _, dn := range filterDateNodes(node.Parents()) {
			if len(dn.Children()) == 1 {
				if err := deleteNode(tx, dn.ID); err != nil {
					return err
				}
				// Unlink to ignore in backprop.
				if err := multitree.UnlinkNodes(dn, node); err != nil {
					panic(err)
				}
			}
		}

		if err := backpropCompletion(tx, node); err != nil {
			return err
		}
		orphans = node.Children()
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return nil, err
	}
	return orphans, nil
}

// DeleteNodeRecursive deletes the tree rooted at the given node and updates the
// multitree. Nodes that have parents outside of this tree are preserved. It
// returns a slice of all deleted nodes.
func (d *Database) DeleteNodeRecursive(id int64) ([]*multitree.Node, error) {
	var deleted []*multitree.Node

	txf := func(tx *sql.Tx) error {
		node, err := getGraph(tx, id)
		if err != nil {
			return err
		}
		if node == nil {
			return fmt.Errorf("node does not exist")
		}

		if err := deleteNode(tx, id); err != nil {
			return err
		}
		deleted = append(deleted, node)

		for _, d := range node.Descendants() {
			if len(d.Parents()) == 1 {
				if err := deleteNode(tx, d.ID); err != nil {
					return err
				}
				deleted = append(deleted, d)
			}
		}

		if err := backpropCompletion(tx, node); err != nil {
			return err
		}
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return nil, err
	}
	return deleted, nil
}

func (d *Database) SetAlias(nodeID int64, alias string) error {
	nullable := &alias
	if alias == "" {
		nullable = nil
	}
	r, err := d.DB.Exec("UPDATE nodes SET node_alias = ? WHERE node_id = ?",
		nullable, nodeID)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("node does not exist")
	}
	return nil
}
