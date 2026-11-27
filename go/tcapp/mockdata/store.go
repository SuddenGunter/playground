package mockdata

import (
	"context"
	_ "embed"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schema string

// Migrate creates the mock_data table if it does not exist yet.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	if _, err := pool.Exec(ctx, schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	return nil
}

// Store is a Postgres-backed repository for mock data.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a Store on top of an existing connection pool.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

const columns = "id, name, value, created_at"

// Create inserts a new item and returns it with its generated fields.
func (s *Store) Create(ctx context.Context, name, value string) (Item, error) {
	rows, err := s.pool.Query(ctx,
		"INSERT INTO mock_data (name, value) VALUES ($1, $2) RETURNING "+columns, name, value)
	if err != nil {
		return Item{}, fmt.Errorf("insert mock data: %w", err)
	}

	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Item])
	if err != nil {
		return Item{}, fmt.Errorf("insert mock data: %w", err)
	}

	return item, nil
}

// Get returns the item with the given id, or ErrNotFound.
func (s *Store) Get(ctx context.Context, id int64) (Item, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+columns+" FROM mock_data WHERE id = $1", id)
	if err != nil {
		return Item{}, fmt.Errorf("get mock data %d: %w", id, err)
	}

	item, err := pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[Item])
	if errors.Is(err, pgx.ErrNoRows) {
		return Item{}, ErrNotFound
	}

	if err != nil {
		return Item{}, fmt.Errorf("get mock data %d: %w", id, err)
	}

	return item, nil
}

// List returns up to limit items ordered by id.
func (s *Store) List(ctx context.Context, limit int) ([]Item, error) {
	rows, err := s.pool.Query(ctx, "SELECT "+columns+" FROM mock_data ORDER BY id LIMIT $1", limit)
	if err != nil {
		return nil, fmt.Errorf("list mock data: %w", err)
	}

	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[Item])
	if err != nil {
		return nil, fmt.Errorf("list mock data: %w", err)
	}

	return items, nil
}
