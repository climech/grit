package db

import (
	"errors"

	sqlite "github.com/mattn/go-sqlite3"
)

var (
	ErrNodeNotFound = errors.New("node does not exist")
	ErrLinkNotFound = errors.New("link does not exist")
	ErrAliasExists  = errors.New("alias already exists")
)

func isUniqueViolation(err error) bool {
	var e sqlite.Error
	return errors.As(err, &e) && e.ExtendedCode == sqlite.ErrConstraintUnique
}
