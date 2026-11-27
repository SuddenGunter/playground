// Package mockdata stores and retrieves rows of the mock_data table.
package mockdata

import (
	"errors"
	"time"
)

var (
	// ErrNotFound is returned when the requested item does not exist.
	ErrNotFound = errors.New("mock data not found")
	// ErrInvalid is returned when an item violates a database constraint, e.g. a field is too long.
	// The wrapping error's message says which rule was broken and is safe to show to clients.
	ErrInvalid = errors.New("invalid mock data")
)

// Length limits enforced by the database schema, in characters.
const (
	MaxNameLength  = 256
	MaxValueLength = 4096
)

// Item is a single row of the mock_data table.
type Item struct {
	ID        int64     `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	Value     string    `db:"value"      json:"value"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
