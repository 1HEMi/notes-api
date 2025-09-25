package storage

import "errors"

var (
	ErrNoteNotFound = errors.New("note not found")
	ErrTitleExists  = errors.New("title already exists")
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
	ErrForbidden    = errors.New("forbidden access")
)
