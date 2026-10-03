package app

import (
	"strconv"

	"github.com/climech/grit/multitree"
)

type selectorKind int

const (
	selectID selectorKind = iota + 1
	selectDate
	selectAlias
)

// Selector identifies a node by ID, date or alias. The zero value is invalid.
type Selector struct {
	kind selectorKind
	id   int64
	name string
}

// ByID selects a node by its ID.
func ByID(id int64) Selector {
	return Selector{kind: selectID, id: id}
}

// ParseSelector interprets s as a node ID, a date, or an alias, in that order.
func ParseSelector(s string) (Selector, error) {
	if id, err := strconv.ParseInt(s, 10, 64); err == nil && id > 0 {
		return ByID(id), nil
	}
	if multitree.ValidateDateNodeName(s) == nil {
		return Selector{kind: selectDate, name: s}, nil
	}
	if multitree.ValidateNodeAlias(s) == nil {
		return Selector{kind: selectAlias, name: s}, nil
	}
	return Selector{}, NewError(ErrInvalidSelector, "invalid selector")
}

// target is a resolved selector. For a date that has no node in the DB yet,
// id is zero and date is set.
type target struct {
	id   int64
	date string
}

func (t target) isMissingDate() bool {
	return t.date != ""
}

func (a *App) resolve(s Selector) (target, error) {
	switch s.kind {
	case selectID:
		if s.id < 1 {
			return target{}, NewError(ErrInvalidSelector, "invalid selector")
		}
		return target{id: s.id}, nil
	case selectDate:
		node, err := a.GetNodeByName(s.name)
		if err != nil {
			return target{}, err
		}
		if node == nil {
			return target{date: s.name}, nil
		}
		return target{id: node.ID}, nil
	case selectAlias:
		node, err := a.GetNodeByAlias(s.name)
		if err != nil {
			return target{}, err
		}
		if node == nil {
			return target{}, NewError(ErrNotFound, "node does not exist")
		}
		return target{id: node.ID}, nil
	default:
		return target{}, NewError(ErrInvalidSelector, "invalid selector")
	}
}
