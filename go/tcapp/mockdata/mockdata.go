// Package mockdata stores and retrieves rows of the mock_data table.
package mockdata

import (
	"errors"
	"time"
)

// ErrNotFound is returned when the requested item does not exist.
var ErrNotFound = errors.New("mock data not found")

// Item is a single row of the mock_data table.
type Item struct {
	ID        int64     `db:"id"         json:"id"`
	Name      string    `db:"name"       json:"name"`
	Value     string    `db:"value"      json:"value"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
