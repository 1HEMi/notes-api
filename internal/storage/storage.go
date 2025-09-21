package storage

import "errors"

var (
	ErrNoteNotFound = errors.New("note not found")
	ErrTitleExists  = errors.New("title already exists")
	ErrUerNotFound  = errors.New("user not found")
	ErrUserExists   = errors.New("user already exists")
)
