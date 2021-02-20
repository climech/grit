// Package app implements grit's business logic layer.
package app

import (
	"fmt"
	"os"
	"path"
	"reflect"
	"strconv"

	"github.com/climech/grit/db"
	"github.com/climech/grit/graph"

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
	// TODO: add config
}

func New() (*App, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("couldn't load user's home directory")
	}

	dirpath := path.Join(home, ".config", AppName)
	filepath := path.Join(dirpath, "graph.db")
	if err := os.MkdirAll(dirpath, 0700); err != nil {
		return nil, fmt.Errorf("couldn't create config directory")
	}

	d, err := db.New(filepath)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize db: %v", err)
	}

	return &App{Database: d}, nil
}

func (a *App) Close() {
	a.Database.Close()
}

// AddNode creates a root and returns it as a member of its graph.
func (a *App) AddRoot(name string) (*graph.Node, error) {
	if err := graph.ValidateNodeName(name); err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if err := graph.ValidateDateNodeName(name); err == nil {
		return nil, NewError(ErrInvalidName, fmt.Sprintf("%v is a reserved name", name))
	}
	nodeId, err := a.Database.CreateNode(name, 0)
	if err != nil {
		return nil, err
	}
	return a.Database.GetGraph(nodeId)
}

// AddSuccessor creates a new node and links an existing node to it. A
// predecessor d-node is implicitly created, if it doesn't already exist.
func (a *App) AddSuccessor(name string, predecessor interface{}) (*graph.Node, error) {
	if err := graph.ValidateNodeName(name); err != nil {
		return nil, NewError(ErrInvalidName, err.Error())
	}
	if err := graph.ValidateDateNodeName(name); err == nil {
		return nil, NewError(ErrInvalidName, fmt.Sprintf("%v is a reserved name", name))
	}
	preId, err := a.selectorToId(predecessor)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}

	var nodeId int64
	var nodeErr error
	if preId == 0 {
		nodeId, nodeErr = a.Database.CreateSuccessorOfDateNode(predecessor.(string), name)
	} else {
		nodeId, nodeErr = a.Database.CreateNode(name, preId)
	}

	if nodeErr != nil {
		if e, ok := nodeErr.(sqlite.Error); ok && e.ExtendedCode == sqlite.ErrConstraintForeignKey {
			return nil, NewError(ErrNotFound, "predecessor does not exist")
		} else {
			return nil, nodeErr
		}
	}

	return a.Database.GetGraph(nodeId)
}

func (a *App) AddTree(node *graph.Node, predecessor interface{}) (int64, error) {
	for _, n := range node.GetAll() {
		if err := graph.ValidateNodeName(n.Name); err != nil {
			return 0, NewError(ErrInvalidName, err.Error())
		}
		if err := graph.ValidateDateNodeName(n.Name); err == nil {
			return 0, NewError(ErrInvalidName, fmt.Sprintf("%v is a reserved name", n.Name))
		}
	}
	preId, err := a.selectorToId(predecessor)
	if err != nil {
		return 0, NewError(ErrInvalidSelector, err.Error())
	}
	id, err := a.Database.CreateTree(node, preId)
	if err != nil {
		e, ok := err.(sqlite.Error)
		if ok && e.ExtendedCode == sqlite.ErrConstraintForeignKey {
			return 0, NewError(ErrNotFound, "predecessor does not exist")
		}
		return 0, err
	}
	return id, nil
}

func (a *App) RenameNode(selector interface{}, name string) error {
	if err := graph.ValidateDateNodeName(name); err == nil {
		return NewError(ErrForbidden, "date nodes cannot be renamed")
	}
	id, err := a.selectorToId(selector)
	if err != nil {
		return NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		return NewError(ErrNotFound, "node does not exist")
	}
	if err := graph.ValidateNodeName(name); err != nil {
		return NewError(ErrInvalidName, err.Error())
	}

	node, err := a.Database.GetNode(id)
	if err != nil {
		return err
	}
	if node == nil {
		return NewError(ErrNotFound, "node does not exist")
	}
	if graph.ValidateDateNodeName(node.Name) == nil {
		return NewError(ErrForbidden, "date nodes cannot be renamed")
	}

	if err := a.Database.RenameNode(node.ID, name); err != nil {
		return err
	}
	return nil
}

func (a *App) GetGraph(selector interface{}) (*graph.Node, error) {
	id, err := a.selectorToId(selector)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		if s, ok := selector.(string); ok && graph.ValidateDateNodeName(s) == nil {
			// Return a mock d-node.
			return graph.NewNode(s), nil
		}
		return nil, NewError(ErrNotFound, "node does not exist")
	}
	return a.Database.GetGraph(id)
}

func (a *App) GetNode(selector interface{}) (*graph.Node, error) {
	id, err := a.selectorToId(selector)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	if id == 0 {
		// Return mock d-node.
		return graph.NewNode(selector.(string)), nil
	}
	return a.Database.GetNode(id)
}
func (a *App) GetNodeByName(name string) (*graph.Node, error) {
	return a.Database.GetNodeByName(name)
}

func (a *App) GetNodeByAlias(alias string) (*graph.Node, error) {
	return a.Database.GetNodeByAlias(alias)
}

// LinkNodes creates a new edge connecting two nodes. D-nodes are implicitly
// created as needed.
func (a *App) LinkNodes(origin, dest interface{}) (*graph.Edge, error) {
	originId, err := a.selectorToId(origin)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}
	destId, err := a.selectorToId(dest)
	if err != nil {
		return nil, NewError(ErrInvalidSelector, err.Error())
	}

	var edgeId int64
	var errCreate error
	if originId == 0 {
		edgeId, errCreate = a.Database.CreateEdgeFromDateNode(origin.(string), destId)
	} else {
		edgeId, errCreate = a.Database.CreateEdge(originId, destId)
	}
	if errCreate != nil {
		if e, ok := errCreate.(sqlite.Error); ok {
			switch e.ExtendedCode {
			case sqlite.ErrConstraintUnique:
				return nil, NewError(ErrForbidden, "edge already exists")
			case sqlite.ErrConstraintForeignKey:
				return nil, NewError(ErrNotFound, "origin or dest does not exist")
			case sqlite.ErrConstraintCheck:
				return nil, NewError(ErrForbidden, "loops are not allowed")
			default:
				return nil, e
			}
		} else {
			return nil, errCreate
		}
	}

	return a.Database.GetEdge(edgeId)
}

// UnlinkNodes removes the edge connecting the given nodes.
func (a *App) UnlinkNodes(origin, dest interface{}) error {
	originId, err := a.selectorToId(origin)
	if err != nil {
		return NewError(ErrInvalidSelector, err.Error())
	}
	destId, err := a.selectorToId(dest)
	if err != nil {
		return err
	}
	if originId == 0 || destId == 0 {
		// Assuming there can't be an edge from/to an empty d-node.
		return NewError(ErrNotFound, "edge does not exist")
	}
	if err := a.Database.DeleteEdgeByEndpoints(originId, destId); err != nil {
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

// RemoveNode deletes the node and returns its orphaned successors.
func (a *App) RemoveNode(selector interface{}) ([]*graph.Node, error) {
	id, err := a.selectorToId(selector)
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
// that have multiple predecessors are only unlinked from the current tree.
func (a *App) RemoveNodeRecursive(selector interface{}) ([]*graph.Node, error) {
	id, err := a.selectorToId(selector)
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
	id, err := a.selectorToId(selector)
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

func (a *App) GetRoots() ([]*graph.Node, error) {
	roots, err := a.Database.GetRoots()
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, nil
	}
	var ret []*graph.Node
	for _, r := range roots {
		// Omit d-nodes.
		if graph.ValidateDateNodeName(r.Name) != nil {
			ret = append(ret, r)
		}
	}
	// TODO: sort by name alphabetically(?)
	return ret, nil
}

func (a *App) GetDateNodes() ([]*graph.Node, error) {
	roots, err := a.Database.GetRoots()
	if err != nil {
		return nil, err
	}
	if len(roots) == 0 {
		return nil, nil
	}
	var ret []*graph.Node
	for _, r := range roots {
		// Omit roots that aren't d-nodes.
		if graph.ValidateDateNodeName(r.Name) == nil {
			ret = append(ret, r)
		}
	}
	// TODO: sort by name alphabetically(?)
	return ret, nil
}

func (a *App) stringSelectorToId(selector string) (int64, error) {
	// Check if integer.
	id, err := strconv.ParseInt(selector, 10, 64)
	if err == nil && id > 0 {
		return id, nil
	}
	// Check if date.
	if graph.ValidateDateNodeName(selector) == nil {
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
	if graph.ValidateNodeAlias(selector) == nil {
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

// selectorToId parses the selector and returns a valid node ID, or zero for
// valid d-node name that isn't in the DB.
func (a *App) selectorToId(selector interface{}) (int64, error) {
	switch value := selector.(type) {
	case *graph.Node:
		return value.ID, nil
	case string:
		return a.stringSelectorToId(value)
	case int64:
		if value < 1 {
			return 0, fmt.Errorf("invalid selector")
		}
		return value, nil
	default:
		panic(fmt.Sprintf("unsupported selector type: %v", reflect.TypeOf(selector)))
	}
}
