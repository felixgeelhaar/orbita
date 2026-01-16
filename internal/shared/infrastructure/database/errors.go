package database

import (
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5"
)

// ErrNoRows is returned when a query expected to return a row returns none.
var ErrNoRows = errors.New("no rows in result set")

// IsNoRows returns true if the error indicates no rows were found.
// This handles both pgx.ErrNoRows and sql.ErrNoRows.
func IsNoRows(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, pgx.ErrNoRows) ||
		errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, ErrNoRows)
}
