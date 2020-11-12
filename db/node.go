package db

import (
	"fmt"
	"database/sql"

	"github.com/climech/grit/graph"

	_ "github.com/mattn/go-sqlite3"
)

func getNode(tx *sql.Tx, id int64) (*graph.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_id = ?", id)
	return rowToNode(row)
}

func (d *Database) GetNode(id int64) (*graph.Node, error) {
	var node *graph.Node
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

func getNodeByName(tx *sql.Tx, name string) (*graph.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_name = ?", name)
	return rowToNode(row)
}

func (d *Database) GetNodeByName(name string) (*graph.Node, error) {
	var node *graph.Node
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

func getNodeByAlias(tx *sql.Tx, alias string) (*graph.Node, error) {
	row := tx.QueryRow("SELECT * FROM nodes WHERE node_alias = ?", alias)
	return rowToNode(row)
}

func (d *Database) GetNodeByAlias(alias string) (*graph.Node, error) {
	var node *graph.Node
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

func isDateNode(tx *sql.Tx, id int64) (bool, error) {
	node, err := getNode(tx, id)
	if err != nil {
		return false, err
	}
	return (graph.ValidateDateNodeName(node.Name) == nil), nil
}

func fixStatusBefore(tx *sql.Tx, node *graph.Node) error {
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

func createNode(tx *sql.Tx, name string, predecessorId int64) (int64, error) {
	r, err := tx.Exec(`INSERT INTO nodes (node_name) VALUES (?)`, name)
	if err != nil {
		return 0, err
	}
	id, _ := r.LastInsertId()
	if predecessorId != 0 {
		if _, err := createEdge(tx, predecessorId, id); err != nil {
			return 0, err
		}
		node, err := getGraph(tx, id)
		if err != nil {
			return 0, err
		}
		if err := fixStatusBefore(tx, node); err != nil {
			return 0, err
		}
	}
	return id, nil
}

// CreateSuccessor creates a node and returns its ID. It updates the status of
// other nodes in the graph if needed.
func (d *Database) CreateNode(name string, predecessorId int64) (int64, error) {
	var succId int64
	txf := func(tx *sql.Tx) error {
		id, err := createNode(tx, name, predecessorId)
		if err != nil {
			return err
		}
		succId = id
		return nil
	}
	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return succId, nil
}

func createDateNodeIfNotExists(tx *sql.Tx, date string) (int64, error) {
	if err := graph.ValidateDateNodeName(date); err != nil {
		panic(err)
	}
	node, err := getNodeByName(tx, date)
	if err != nil {
		return 0, err
	}
	if node != nil {
		return node.Id, nil
	}
	return createNode(tx, date, 0)
}

// CreateSuccessorOfDateNode atomically creates a node and makes it a successor
// of a date node. Date node is created if it doesn't exist.
func (d *Database) CreateSuccessorOfDateNode(date, name string) (int64, error) {
	var succId int64

	txf := func(tx *sql.Tx) error {
		dateNodeId, err := createDateNodeIfNotExists(tx, date)
		if err != nil {
			return err
		}
		succId, err = createNode(tx, name, dateNodeId)
		if err != nil {
			return err
		}
		return nil
	}
	
	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return succId, nil
}

func createTree(tx *sql.Tx, node *graph.Node, predecessorId int64) (int64, error) {
	tree := node.Tree() // Copy
	rootId, err := createNode(tx, tree.Name, predecessorId)
	if err != nil {
		return 0, err
	} else {
		tree.Id = rootId
	}

	// Traverse non-recursively so we can return immediately in case of error.
	stack := []*graph.Node{tree}
	for len(stack) > 0 {
		current := stack[len(stack) - 1]
		if len(current.Successors) > 0 {
			var child *graph.Node
			child, current.Successors = current.Successors[0], current.Successors[1:] // shift
			if id, err := createNode(tx, child.Name, current.Id); err != nil {
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

	return rootId, nil
}

// CreateTree saves an entire tree in the database and returns the root ID. It
// updates the status of other nodes in the graph to reflect the change.
func (d *Database) CreateTree(node *graph.Node, predecessorId int64) (int64, error) {
	var rootId int64

	txf := func(tx *sql.Tx) error {
		id, err := createTree(tx, node, predecessorId)
		if err != nil {
			return err
		}
		rootId = id
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return rootId, nil
}

// CreateTreeAsSuccessorOfDateNode atomically creates a tree as a successor of
// date node. Date node is created if it doesn't exist.
func (d *Database) CreateTreeAsSuccessorOfDateNode(date string, node *graph.Node) (int64, error) {
	var rootId int64

	txf := func(tx *sql.Tx) error {
		dateNodeId, err := createDateNodeIfNotExists(tx, date)
		if err != nil {
			return err
		}
		id, err := createTree(tx, node, dateNodeId)
		if err != nil {
			return err
		}
		rootId = id
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return 0, err
	}
	return rootId, nil
}

func (d *Database) checkNode(nodeId int64, value bool) error {
	update := func(tx *sql.Tx, node *graph.Node) error {
		r, err := tx.Exec("UPDATE nodes SET node_checked = ? WHERE node_id = ?", value, node.Id)
		if err != nil {
			return err
		}
		if count, _ := r.RowsAffected(); count == 0 {
			return fmt.Errorf("node does not exist")
		}
		node.Checked = value
		return nil
	}

	return d.execTxFunc(func(tx *sql.Tx) error {
		node, err := getGraph(tx, nodeId)
		if err != nil {
			return err
		}
		if node == nil {
			return fmt.Errorf("node does not exist")
		}
		if err := update(tx, node); err != nil {
			return err
		}
		descendants := node.NodesAfter()
		for _, d := range descendants {
			if d.Checked != value {
				if err := update(tx, d); err != nil {
					return err
				}
			}
		}
		if err := fixStatusBefore(tx, node); err != nil {
			return err
		}
		return nil
	})
}

// CheckNode marks the node as completed, along with all its direct and indirect
// successors. The rest of the graph is updated to reflect the change.
func (d *Database) CheckNode(nodeId int64) error {
	return d.checkNode(nodeId, true)
}

// UncheckNode sets the node's status to inactive, along with all its direct
// and indirect successors. The rest of the graph is updated to reflect the
// change.
func (d *Database) UncheckNode(nodeId int64) error {
	return d.checkNode(nodeId, false)
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

// DeleteNode deletes the node and propagates the change to the rest of the
// graph. It returns the node's orphaned successors.
func (d *Database) DeleteNode(id int64) ([]*graph.Node, error) {
	var orphans []*graph.Node

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
		if err := fixStatusBefore(tx, node); err != nil {
			return err
		}
		orphans = node.Successors
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return nil, err
	}
	return orphans, nil
}

// DeleteNodeRecursive deletes the entire tree rooted at node and updates the
// graph. Nodes that have more than one predecessor are unlinked from the current
// tree. Returns a slice of all deleted nodes.
func (d *Database) DeleteNodeRecursive(id int64) ([]*graph.Node, error) {
	var deleted []*graph.Node

	txf := func(tx *sql.Tx) error {
		node, err := getGraph(tx, id)
		if err != nil {
			return err
		}
		if node == nil {
			return fmt.Errorf("node does not exist")
		}

		// Root.
		if err := deleteNode(tx, id); err != nil {
			return err
		}

		// Successors.
		tree := node.Tree()
		for _, n := range node.NodesAfter() {
			if len(n.Predecessors) == 1 {
				if err := deleteNode(tx, n.Id); err != nil {
					return err
				}
				deleted = append(deleted, n)
			} else {
				for _, p := range n.Predecessors {
					if tree.Get(p.Id) == nil {
						continue // only if p belongs to the same tree
					}
					if err := deleteEdgeByEndpoints(tx, p.Id, n.Id); err != nil {
						return err
					}
				}
			}
		}

		if err := fixStatusBefore(tx, node); err != nil {
			return err
		}
		return nil
	}

	if err := d.execTxFunc(txf); err != nil {
		return nil, err
	}
	return deleted, nil
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
