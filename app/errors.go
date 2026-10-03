package app

import (
	"errors"

	"github.com/climech/grit/db"
)

type ErrCode int

const (
	ErrNotFound ErrCode = iota
	ErrForbidden
	ErrInvalidSelector
	ErrInvalidName
)

type AppError struct {
	Msg  string
	Code ErrCode
}

func (e *AppError) Error() string {
	return e.Msg
}

func NewError(code ErrCode, msg string) *AppError {
	return &AppError{Msg: msg, Code: code}
}

// translateDBError maps db sentinel errors to *AppError and passes others through.
func translateDBError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, db.ErrNodeNotFound), errors.Is(err, db.ErrLinkNotFound):
		return NewError(ErrNotFound, err.Error())
	case errors.Is(err, db.ErrAliasExists):
		return NewError(ErrForbidden, err.Error())
	}
	return err
}
