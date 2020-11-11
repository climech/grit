package db

import (
	"fmt"
	"errors"
	"database/sql"

	"github.com/climech/grit/graph"

	_ "github.com/mattn/go-sqlite3"
)

func (d *Database) GetNode(id int64) (*graph.Node, error) {
	row := d.DB.QueryRow("SELECT * FROM nodes WHERE node_id = ?", id)
	return rowToNode(row)
}

func (d *Database) GetNodeByName(name string) (*graph.Node, error) {
	row := d.DB.QueryRow("SELECT * FROM nodes WHERE node_name = ?", name)
	return rowToNode(row)
}

func (d *Database) GetNodeByAlias(alias string) (*graph.Node, error) {
	row := d.DB.QueryRow("SELECT * FROM nodes WHERE node_alias = ?", alias)
	return rowToNode(row)
}

func (d *Database) GetRoots() ([]*graph.Node, error) {
	rows, err := d.DB.Query(
		"SELECT * FROM nodes " +
		"WHERE NOT EXISTS(SELECT * FROM edges WHERE dest_id = node_id)",
	)
	if err != nil {
		return nil, err
	}
	return rowsToNodes(rows), nil
}

func txCreateNode(tx *sql.Tx, name string, predecessorId int64) (int64, error) {
	r, err := tx.Exec(`INSERT INTO nodes (node_name) VALUES (?)`, name)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	id, _ := r.LastInsertId()

	if predecessorId != 0 {
		if _, err := txCreateEdge(tx, predecessorId, id); err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	return id, nil
}

// CreateNode creates a new node and returns its ID.
func (d *Database) CreateNode(name string) (int64, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return 0, err
	}
	id, err := txCreateNode(tx, name, 0)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// CreateSuccessor creates a new node and makes it a successor of an existing
// node. Returns node ID if successful.
func (d *Database) CreateSuccessor(name string, predecessorId int64) (int64, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return 0, err
	}
	nodeId, err := txCreateNode(tx, name, predecessorId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	// Fix status of nodes.
	node, err := txGetGraph(tx, nodeId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if err := txFixBefore(tx, node); err != nil {
		tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return nodeId, nil
}

// txCreateDateNode creates a date node as part of a transaction, if it doesn't
// exist. The date node ID is returned in any case.
func txCreateDateNode(tx *sql.Tx, date string) (int64, error) {
	if graph.ValidateDateNodeName(date) != nil {
		panic("invalid date node name")
	}
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_name = ?", date)
	node, err := rowToNode(row)
	if err != nil {
		return 0, err
	}
	if node != nil {
		return node.Id, nil
	}
	return txCreateNode(tx, date, 0)
}

// CreateSuccessorOfDateNode creates a node and makes it a successor of a d-node.
// If d-node doesn't exist, it's created automatically.
func (d *Database) CreateSuccessorOfDateNode(name, date string) (int64, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return 0, err
	}
	dateNodeId, err := txCreateDateNode(tx, date)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	nodeId, err := txCreateNode(tx, name, dateNodeId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return nodeId, nil
}

// CreateTree creates an entire tree of nodes and returns the root ID.
func txCreateTree(tx *sql.Tx, node *graph.Node, predecessorId int64) (int64, error) {
	tree := node.Tree() // Copy
	treeRootId, err := txCreateNode(tx, tree.Name, predecessorId)
	if err != nil {
		tx.Rollback()
		return 0, err
	} else {
		tree.Id = treeRootId
	}

	// Traverse non-recursively (can be stopped immediately in case of an error).
	stack := []*graph.Node{tree}
	for len(stack) > 0 {
		current := stack[len(stack) - 1]
		if len(current.Successors) > 0 {
			var child *graph.Node
			child, current.Successors = current.Successors[0], current.Successors[1:] // shift
			if id, err := txCreateNode(tx, child.Name, current.Id); err != nil {
				tx.Rollback()
				return 0, err
			} else {
				child.Id = id
			}
			if len(child.Successors) > 0 {
				stack = append(stack, child) // push
			}
		} else {
			stack = stack[:len(stack) - 1] // pop
		}
	}

	// Fix ancestors if necessary.
	if predecessorId != 0 {
		node, err := txGetGraph(tx, treeRootId)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		if node == nil {
			tx.Rollback()
			return 0, fmt.Errorf("something went wrong")
		}
		if err := txFixBefore(tx, node); err != nil {
			tx.Rollback()
			return 0, err
		}
	}

	return treeRootId, nil
}

func (d *Database) CreateTree(node *graph.Node, predecessor interface{}) (int64, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return 0, err
	}

	var predecessorId int64
	switch value := predecessor.(type) {
	case string:
		var err error
		predecessorId, err = txCreateDateNode(tx, value)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
	case int:
		predecessorId = int64(value)
	case int64:
		predecessorId = value
	default:
		panic("unsupported selector type")
	}

	id, err := txCreateTree(tx, node, predecessorId)
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// CheckNode marks the node as completed and updates the graph to reflect the
// change.
func (d *Database) CheckNode(nodeId int64, checked bool) error  {
	tx, err := d.BeginTx()
	if err != nil {
		return err
	}

	node, err := txGetGraph(tx, nodeId)
	if err != nil {
		tx.Rollback()
		return err
	}
	if node == nil {
		return errors.New("node does not exist")
	}

	update := func(node *graph.Node) error {
		_, err := tx.Exec("UPDATE nodes SET node_checked = ? WHERE node_id = ?", node.Checked, node.Id)
		if err != nil {
			tx.Rollback()
			return err
		}
		return nil
	}

	node.Checked = checked
	if err := update(node); err != nil {
		return err
	}
	// Check all direct and indirect successors.
	descendants := node.NodesAfter()
	for _, d := range descendants {
		if d.Checked != checked {
			d.Checked = checked
			if err := update(d); err != nil {
				return err
			}
		}
	}

	// Ancestors may need checking too.
	if err := txFixBefore(tx, node); err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (d *Database) RenameNode(nodeId int64, name string) error  {
	r, err := d.DB.Exec("UPDATE nodes SET node_name = ? WHERE node_id = ?", name, nodeId)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("not found")
	}
	return nil
}

func txUpdateNode(tx *sql.Tx, node *graph.Node) error {
	_, err := tx.Exec(
		"UPDATE nodes SET node_name = ?, node_alias = ?, node_checked = ? WHERE node_id = ?",
		node.Name,
		node.Alias,
		node.Checked,
		node.Id,
	)
	if err != nil {
		return err
	}
	return nil
}

func txDeleteNode(tx *sql.Tx, node *graph.Node) error {
	r, err := tx.Exec(`DELETE FROM nodes WHERE node_id = ?`, node.Id)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("node does not exist")
	}
	return nil
}

// DeleteNode deletes the node and propagates the change to the rest of the
// graph. Returns the node's orphaned successors.
func (d *Database) DeleteNode(id int64) ([]*graph.Node, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return nil, err
	}
	node, err := txGetGraph(tx, id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := txDeleteNode(tx, node); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := txFixBefore(tx, node); err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return node.Successors, nil
}

// DeleteNodeRecursive deletes the entire tree rooted at node and updates the
// graph. Nodes with more than one predecessor are unlinked from the current
// tree. Returns a slice of all deleted nodes.
func (d *Database) DeleteNodeRecursive(id int64) ([]*graph.Node, error) {
	tx, err := d.BeginTx()
	if err != nil {
		return nil, err
	}

	node, err := txGetGraph(tx, id)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	if node == nil {
		tx.Rollback()
		return nil, fmt.Errorf("node does not exist")
	}

	descendants := node.NodesAfter()
	tree := node.Tree()
	var deleted []*graph.Node

	for _, n := range descendants {
		if len(n.Predecessors) == 1 {
			if err := txDeleteNode(tx, n); err != nil {
				tx.Rollback()
				return nil, err
			}
			deleted = append(deleted, n)
		} else {
			for _, p := range n.Predecessors {
				// Only if p belongs to the same tree.
				if tree.Get(p.Id) == nil {
					continue
				}
				if err := txDeleteEdgeByEndpoints(tx, p.Id, n.Id); err != nil {
					tx.Rollback()
					return nil, err
				}
			}
		}
	}

	// Delete the root.
	if err := txDeleteNode(tx, node); err != nil {
		tx.Rollback()
		return nil, err
	}

	if err := txFixBefore(tx, node); err != nil {
		tx.Rollback()
		return nil, err
	}
	
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return deleted, nil
}

// txFixBefore fixes the status of nodes on the paths between the roots and
// the node.
func txFixBefore(tx *sql.Tx, node *graph.Node) error {
	var update []*graph.Node

	found := true
	for found {
		found = false
		node.Each(func(n *graph.Node) {
			if len(n.Successors) == 0 {
				return
			}
			allChecked := true
			for _, succ := range n.Successors {
				if !succ.Checked {
					allChecked = false
					break
				}
			}
			if n.Checked != allChecked {
				n.Checked = allChecked
				found = true
				update = append(update, n)
			}
		})
	}

	for _, node := range update {
		_, err := tx.Exec("UPDATE nodes SET node_checked = ? WHERE node_id = ?", node.Checked, node.Id)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Database) SetAlias(nodeId int64, alias string) error {
	nullable := &alias
	if alias == "" {
		nullable = nil
	}
	r, err := d.DB.Exec("UPDATE nodes SET node_alias = ? WHERE node_id = ?", nullable, nodeId)
	if err != nil {
		return err
	}
	if count, _ := r.RowsAffected(); count == 0 {
		return fmt.Errorf("node does not exist")
	}
	return nil
}
