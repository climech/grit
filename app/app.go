// Package app implements grit's business logic layer.
package app

import (
	"fmt"
	"path"
	"reflect"
	"strconv"

	"github.com/climech/grit/db"
	"github.com/climech/grit/multitree"

	"github.com/kirsle/configdir"
	sqlite "github.com/mattn/go-sqlite3"
)

const (
	AppName = "grit"
)

var (
	Version = "development" // overwritten on build
)

type App struct {
	Database *db.Database
}

func New() (*App, error) {
	configPath := configdir.LocalConfig(AppName)
	if err := configdir.MakePath(configPath); err != nil {
		return nil, err
	}

	dbPath := path.Join(configPath, "graph.db")
	d, err := db.New(dbPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize db: %v", err)
	}

	return &App{Database: d}, nil
}

func (a *App) Close() {
	a.Database.Close()
}

// AddNode creates a root and returns it as a member of its multitree.
func (a *App) AddRoot(name string) (*multitree.Node, error) {
	if err := multitree.ValidateNodeName(name); err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if err := multitree.ValidateDateNodeName(name); err == nil {
		return nil, NewError(ErrInvalidName,
			fmt.Sprintf("%v is a reserved name", name))
	}
	nodeID, err := a.Database.CreateNode(name, 0)
	if err != nil {
		return nil, err
	}
	return a.Database.GetGraph(nodeID)
}

// AddChild creates a new node and links an existing node to it. A
// parent d-node is implicitly created, if it doesn't already exist.
func (a *App) AddChild(name string, parent interface{}) (*multitree.Node, error) {
	if err := multitree.ValidateNodeName(name); err != nil {
		return nil, NewError(ErrInvalidName, err.Error())
	}
	if err := multitree.ValidateDateNodeName(name); err == nil {
		return nil, NewError(ErrInvalidName,
			fmt.Sprintf("%v is a reserved name", name))
	}
	parentID, err := a.selectorToID(parent)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}

	var nodeID int64
	var nodeErr error
	if parentID == 0 {
		nodeID, nodeErr = a.Database.CreateChildOfDateNode(parent.(string), name)
	} else {
		nodeID, nodeErr = a.Database.CreateNode(name, parentID)
	}

	if nodeErr != nil {
		if e, ok := nodeErr.(sqlite.Error); ok && e.ExtendedCode == sqlite.ErrConstraintForeignKey {
			return nil, NewError(ErrNotFound, "parent does not exist")
		} else {
			return nil, nodeErr
		}
	}

	return a.Database.GetGraph(nodeID)
}

func validateTree(root *multitree.Node) error {
	for _, n := range root.All() {
		if err := multitree.ValidateNodeName(n.Name); err != nil {
			return NewError(ErrInvalidName, err.Error())
		}
		if err := multitree.ValidateDateNodeName(n.Name); err == nil {
			return NewError(ErrInvalidName,
				fmt.Sprintf("%v is a reserved name", n.Name))
		}
	}
	return nil
}

// AddRootTree creates a new tree at the root level. It returns the root ID.
func (a *App) AddRootTree(tree *multitree.Node) (int64, error) {
	if err := validateTree(tree); err != nil {
		return 0, err
	}
	return a.Database.CreateTree(tree, 0)
}

// AddChildTree creates a new tree and links parent to its root. It returns the
// root ID.
func (a *App) AddChildTree(tree *multitree.Node, parent interface{}) (int64, error) {
	if err := validateTree(tree); err != nil {
		return 0, err
	}

	parentID, err := a.selectorToID(parent)
	if err != nil {
		return 0, NewError(ErrInvalidSelector, err.Error())
	}

	var rootID int64
	var createErr error
	if parentID == 0 {
		rootID, createErr = a.Database.CreateTreeAsChildOfDateNode(parent.(string), tree)
	} else {
		rootID, createErr = a.Database.CreateTree(tree, parentID)
	}

	if createErr != nil {
		if e, ok := createErr.(sqlite.Error); ok && e.ExtendedCode == sqlite.ErrConstraintForeignKey {
			return 0, NewError(ErrNotFound, "parent does not exist")
		} else {
			return 0, createErr
		}
	}

	return rootID, nil
}

func (a *App) RenameNode(selector interface{}, name string) error {
	if err := multitree.ValidateDateNodeName(name); err == nil {
		return NewError(ErrForbidden, "date nodes cannot be renamed")
	}
	id, err := a.selectorToID(selector)
	if err != nil {
		return NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		return NewError(ErrNotFound, "node does not exist")
	}
	if err := multitree.ValidateNodeName(name); err != nil {
		return NewError(ErrInvalidName, err.Error())
	}

	node, err := a.Database.GetNode(id)
	if err != nil {
		return err
	}
	if node == nil {
		return NewError(ErrNotFound, "node does not exist")
	}
	if multitree.ValidateDateNodeName(node.Name) == nil {
		return NewError(ErrForbidden, "date nodes cannot be renamed")
	}

	if err := a.Database.RenameNode(node.ID, name); err != nil {
		return err
	}
	return nil
}

func (a *App) GetGraph(selector interface{}) (*multitree.Node, error) {
	id, err := a.selectorToID(selector)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		if s, ok := selector.(string); ok && multitree.ValidateDateNodeName(s) == nil {
			// Return a mock d-node.
			return multitree.NewNode(s), nil
		}
		return nil, NewError(ErrNotFound, "node does not exist")
	}
	return a.Database.GetGraph(id)
}

func (a *App) GetNode(selector interface{}) (*multitree.Node, error) {
	id, err := a.selectorToID(selector)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		// Return mock d-node.
		return multitree.NewNode(selector.(string)), nil
	}
	return a.Database.GetNode(id)
}
func (a *App) GetNodeByName(name string) (*multitree.Node, error) {
	return a.Database.GetNodeByName(name)
}

func (a *App) GetNodeByAlias(alias string) (*multitree.Node, error) {
	return a.Database.GetNodeByAlias(alias)
}

// LinkNodes creates a new link connecting two nodes. D-nodes are implicitly
// created as needed.
func (a *App) LinkNodes(origin, dest interface{}) (*multitree.Link, error) {
	originID, err := a.selectorToID(origin)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	destID, err := a.selectorToID(dest)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}

	var linkID int64
	var errCreate error
	if originID == 0 {
		linkID, errCreate = a.Database.CreateLinkFromDateNode(origin.(string), destID)
	} else {
		linkID, errCreate = a.Database.CreateLink(originID, destID)
	}
	if errCreate != nil {
		return nil, errCreate
	}

	return a.Database.GetLink(linkID)
}

// UnlinkNodes removes the link connecting the given nodes.
func (a *App) UnlinkNodes(origin, dest interface{}) error {
	originID, err := a.selectorToID(origin)
	if err != nil {
		return NewError(ErrInvalidSelector, err.Error())
	}
	destID, err := a.selectorToID(dest)
	if err != nil {
		return err
	}
	if originID == 0 || destID == 0 {
		// Assuming there can't be an link from/to an empty d-node.
		return NewError(ErrNotFound, "link does not exist")
	}
	if err := a.Database.DeleteLinkByEndpoints(originID, destID); err != nil {
		return err
	}
	return nil
}

func (a *App) SetAlias(id int64, alias string) error {
	err := a.Database.SetAlias(id, alias)
	if err != nil {
		if e, ok := err.(sqlite.Error); ok && e.ExtendedCode == sqlite.ErrConstraintUnique {
			return NewError(ErrForbidden, "alias already exists")
		}
		return err
	}
	return nil
}

// RemoveNode deletes the node and returns its orphaned children.
func (a *App) RemoveNode(selector interface{}) ([]*multitree.Node, error) {
	id, err := a.selectorToID(selector)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		return nil, NewError(ErrNotFound, "node does not exist")
	}
	orphaned, err := a.Database.DeleteNode(id)
	if err != nil {
		return nil, err
	}
	return orphaned, nil
}

// RemoveNodeRecursive deletes the node and all its tree descendants. Nodes
// that have multiple parents are only unlinked from the current tree.
func (a *App) RemoveNodeRecursive(selector interface{}) ([]*multitree.Node, error) {
	id, err := a.selectorToID(selector)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		return nil, NewError(ErrNotFound, "node does not exist")
	}
	deleted, err := a.Database.DeleteNodeRecursive(id)
	if err != nil {
		return nil, err
	}
	return deleted, nil
}

func (a *App) checkNode(selector interface{}, value bool) error {
	id, err := a.selectorToID(selector)
	if err != nil {
		return NewError(ErrInvalidSelector, err.Error())
	}
	if value {
		return a.Database.CheckNode(id)
	}
	return a.Database.UncheckNode(id)
}

func (a *App) CheckNode(selector interface{}) error {
	return a.checkNode(selector, true)
}

func (a *App) UncheckNode(selector interface{}) error {
	return a.checkNode(selector, false)
}

func (a *App) GetRoots() ([]*multitree.Node, error) {
	roots, err := a.Database.GetRoots()
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, nil
	}
	var ret []*multitree.Node
	for _, r := range roots {
		// Omit d-nodes.
		if multitree.ValidateDateNodeName(r.Name) != nil {
			ret = append(ret, r)
		}
	}
	// TODO: sort by name alphabetically(?)
	return ret, nil
}

func (a *App) GetDateNodes() ([]*multitree.Node, error) {
	roots, err := a.Database.GetRoots()
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, nil
	}
	var ret []*multitree.Node
	for _, r := range roots {
		// Omit roots that aren't d-nodes.
		if multitree.ValidateDateNodeName(r.Name) == nil {
			ret = append(ret, r)
		}
	}
	// TODO: sort by name alphabetically(?)
	return ret, nil
}

func (a *App) stringSelectorToID(selector string) (int64, error) {
	// Check if integer.
	id, err := strconv.ParseInt(selector, 10, 64)
	if err == nil && id > 0 {
		return id, nil
	}
	// Check if date.
	if multitree.ValidateDateNodeName(selector) == nil {
		node, err := a.GetNodeByName(selector)
		if err != nil {
			return 0, err
		}
		if node == nil {
			return 0, nil // not found
		}
		return node.ID, nil
	}
	// Check if alias.
	if multitree.ValidateNodeAlias(selector) == nil {
		node, err := a.GetNodeByAlias(selector)
		if err != nil {
			return 0, err
		}
		if node == nil {
			return 0, nil // not found
		}
		return node.ID, nil
	}
	return 0, fmt.Errorf("invalid selector")
}

// selectorToID parses the selector and returns a valid node ID, or zero for
// valid d-node name that isn't in the DB.
func (a *App) selectorToID(selector interface{}) (int64, error) {
	switch value := selector.(type) {
	case *multitree.Node:
		return value.ID, nil
	case string:
		return a.stringSelectorToID(value)
	case int64:
		if value < 1 {
			return 0, fmt.Errorf("invalid selector")
		}
		return value, nil
	default:
		panic(fmt.Sprintf("unsupported selector type: %v",
			reflect.TypeOf(selector)))
	}
}
