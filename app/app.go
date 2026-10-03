// Package app implements grit's business logic layer.
package app

import (
	"errors"
	"fmt"
	"path"

	"github.com/climech/grit/db"
	"github.com/climech/grit/multitree"

	"github.com/kirsle/configdir"
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

// AddRoot creates a root and returns it as a member of its multitree.
func (a *App) AddRoot(name string) (*multitree.Node, error) {
	if err := multitree.ValidateNodeName(name); err != nil {
		return nil, NewError(ErrInvalidName, err.Error())
	}
	if err := multitree.ValidateDateNodeName(name); err == nil {
		return nil, NewError(ErrInvalidName,
			fmt.Sprintf("%v is a reserved name", name))
	}
	nodeID, err := a.Database.CreateNode(name, 0)
	if err != nil {
		return nil, translateDBError(err)
	}
	return a.Database.GetGraph(nodeID)
}

// AddChild creates a new node and links an existing node to it. A
// parent d-node is implicitly created, if it doesn't already exist.
func (a *App) AddChild(name string, parent Selector) (*multitree.Node, error) {
	if err := multitree.ValidateNodeName(name); err != nil {
		return nil, NewError(ErrInvalidName, err.Error())
	}
	if err := multitree.ValidateDateNodeName(name); err == nil {
		return nil, NewError(ErrInvalidName,
			fmt.Sprintf("%v is a reserved name", name))
	}
	p, err := a.resolve(parent)
	if err != nil {
		return nil, err
	}

	var nodeID int64
	if p.isMissingDate() {
		nodeID, err = a.Database.CreateChildOfDateNode(p.date, name)
	} else {
		nodeID, err = a.Database.CreateNode(name, p.id)
	}
	if err != nil {
		if errors.Is(err, db.ErrNodeNotFound) {
			return nil, NewError(ErrNotFound, "parent does not exist")
		}
		return nil, err
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
	id, err := a.Database.CreateTree(tree, 0)
	return id, translateDBError(err)
}

// AddChildTree creates a new tree and links parent to its root. It returns the
// root ID.
func (a *App) AddChildTree(tree *multitree.Node, parent Selector) (int64, error) {
	if err := validateTree(tree); err != nil {
		return 0, err
	}

	p, err := a.resolve(parent)
	if err != nil {
		return 0, err
	}

	var rootID int64
	if p.isMissingDate() {
		rootID, err = a.Database.CreateTreeAsChildOfDateNode(p.date, tree)
	} else {
		rootID, err = a.Database.CreateTree(tree, p.id)
	}
	if err != nil {
		if errors.Is(err, db.ErrNodeNotFound) {
			return 0, NewError(ErrNotFound, "parent does not exist")
		}
		return 0, err
	}

	return rootID, nil
}

func (a *App) RenameNode(selector Selector, name string) error {
	if err := multitree.ValidateDateNodeName(name); err == nil {
		return NewError(ErrForbidden, "date nodes cannot be renamed")
	}
	t, err := a.resolve(selector)
	if err != nil {
		return err
	}
	if t.isMissingDate() {
		return NewError(ErrNotFound, "node does not exist")
	}
	if err := multitree.ValidateNodeName(name); err != nil {
		return NewError(ErrInvalidName, err.Error())
	}

	node, err := a.Database.GetNode(t.id)
	if err != nil {
		return err
	}
	if node == nil {
		return NewError(ErrNotFound, "node does not exist")
	}
	if multitree.ValidateDateNodeName(node.Name) == nil {
		return NewError(ErrForbidden, "date nodes cannot be renamed")
	}

	return translateDBError(a.Database.RenameNode(node.ID, name))
}

// GetGraph returns the node as part of its multitree, or nil if the ID doesn't
// exist. A date that has no node yet yields an unsaved mock d-node.
func (a *App) GetGraph(selector Selector) (*multitree.Node, error) {
	t, err := a.resolve(selector)
	if err != nil {
		return nil, err
	}
	if t.isMissingDate() {
		return multitree.NewNode(t.date), nil
	}
	return a.Database.GetGraph(t.id)
}

func (a *App) GetNode(selector Selector) (*multitree.Node, error) {
	t, err := a.resolve(selector)
	if err != nil {
		return nil, err
	}
	if t.isMissingDate() {
		return multitree.NewNode(t.date), nil
	}
	return a.Database.GetNode(t.id)
}

func (a *App) GetNodeByName(name string) (*multitree.Node, error) {
	return a.Database.GetNodeByName(name)
}

func (a *App) GetNodeByAlias(alias string) (*multitree.Node, error) {
	return a.Database.GetNodeByAlias(alias)
}

// LinkNodes creates a new link connecting two nodes. D-nodes are implicitly
// created as needed.
func (a *App) LinkNodes(origin, dest Selector) (*multitree.Link, error) {
	o, err := a.resolve(origin)
	if err != nil {
		return nil, err
	}
	d, err := a.resolve(dest)
	if err != nil {
		return nil, err
	}
	if d.isMissingDate() {
		return nil, NewError(ErrNotFound, "link target does not exist")
	}

	var linkID int64
	if o.isMissingDate() {
		linkID, err = a.Database.CreateLinkFromDateNode(o.date, d.id)
	} else {
		linkID, err = a.Database.CreateLink(o.id, d.id)
	}
	if err != nil {
		return nil, translateDBError(err)
	}

	return a.Database.GetLink(linkID)
}

// UnlinkNodes removes the link connecting the given nodes.
func (a *App) UnlinkNodes(origin, dest Selector) error {
	o, err := a.resolve(origin)
	if err != nil {
		return err
	}
	d, err := a.resolve(dest)
	if err != nil {
		return err
	}
	if o.isMissingDate() || d.isMissingDate() {
		// An empty d-node has no links.
		return NewError(ErrNotFound, "link does not exist")
	}
	return translateDBError(a.Database.DeleteLinkByEndpoints(o.id, d.id))
}

func (a *App) SetAlias(id int64, alias string) error {
	return translateDBError(a.Database.SetAlias(id, alias))
}

// RemoveNode deletes the node and returns its orphaned children.
func (a *App) RemoveNode(selector Selector) ([]*multitree.Node, error) {
	t, err := a.resolve(selector)
	if err != nil {
		return nil, err
	}
	if t.isMissingDate() {
		return nil, NewError(ErrNotFound, "node does not exist")
	}
	orphaned, err := a.Database.DeleteNode(t.id)
	if err != nil {
		return nil, translateDBError(err)
	}
	return orphaned, nil
}

// RemoveNodeRecursive deletes the node and all its tree descendants. Nodes
// that have multiple parents are only unlinked from the current tree.
func (a *App) RemoveNodeRecursive(selector Selector) ([]*multitree.Node, error) {
	t, err := a.resolve(selector)
	if err != nil {
		return nil, err
	}
	if t.isMissingDate() {
		return nil, NewError(ErrNotFound, "node does not exist")
	}
	deleted, err := a.Database.DeleteNodeRecursive(t.id)
	if err != nil {
		return nil, translateDBError(err)
	}
	return deleted, nil
}

func (a *App) checkNode(selector Selector, value bool) error {
	t, err := a.resolve(selector)
	if err != nil {
		return err
	}
	if t.isMissingDate() {
		return NewError(ErrNotFound, "node does not exist")
	}
	if value {
		return translateDBError(a.Database.CheckNode(t.id))
	}
	return translateDBError(a.Database.UncheckNode(t.id))
}

func (a *App) CheckNode(selector Selector) error {
	return a.checkNode(selector, true)
}

func (a *App) UncheckNode(selector Selector) error {
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
