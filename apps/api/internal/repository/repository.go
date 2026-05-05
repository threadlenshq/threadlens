package repository

import (
	"database/sql"
	"errors"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")
var ErrValidation = errors.New("validation")

type Repository struct {
	DB *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{DB: db}
}
