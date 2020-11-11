package app

import (
)

type ErrCode int

const (
	ErrNotFound ErrCode = iota
	ErrForbidden
	ErrInvalidSelector
	ErrInvalidName
)

type AppError struct {
	Msg string
	Code ErrCode
}

func (e *AppError) Error() string {
	return e.Msg
}

func NewError(code ErrCode, msg string) *AppError {
	return &AppError{Msg: msg, Code: code}
}
