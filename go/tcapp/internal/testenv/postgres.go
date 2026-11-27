// Package testenv provides shared helpers for integration tests.
package testenv

import (
	"context"
	"fmt"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// PostgresImage is the image used for integration tests. Fully qualified so podman can resolve it.
const PostgresImage = "docker.io/library/postgres:17-alpine"

// Postgres is a running Postgres container shared by the tests of one package.
type Postgres struct {
	ctr   *postgres.PostgresContainer
	dsn   string
	admin *pgxpool.Pool
	seq   atomic.Int64
}

// StartPostgres starts a Postgres container. Call it from TestMain and Close it when tests finish.
func StartPostgres(ctx context.Context) (*Postgres, error) {
	ctr, err := postgres.Run(ctx, PostgresImage,
		postgres.WithDatabase("tcapp"),
		postgres.WithUsername("tcapp"),
		postgres.WithPassword("tcapp"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		if ctr != nil {
			_ = testcontainers.TerminateContainer(ctr)
		}

		return nil, fmt.Errorf("start postgres container: %w", err)
	}

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)

		return nil, fmt.Errorf("postgres connection string: %w", err)
	}

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		_ = testcontainers.TerminateContainer(ctr)

		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	return &Postgres{ctr: ctr, dsn: dsn, admin: admin}, nil
}

// Close stops the container.
func (p *Postgres) Close() error {
	p.admin.Close()

	if err := testcontainers.TerminateContainer(p.ctr); err != nil {
		return fmt.Errorf("terminate postgres container: %w", err)
	}

	return nil
}

// NewDatabase creates an empty database for one test and returns its connection string.
// Each test gets its own database, so parallel tests never see each other's rows.
func (p *Postgres) NewDatabase(t *testing.T) string {
	t.Helper()

	name := fmt.Sprintf("test_%d_%d", time.Now().UnixNano(), p.seq.Add(1))

	_, err := p.admin.Exec(t.Context(), "CREATE DATABASE "+pgx.Identifier{name}.Sanitize())
	require.NoError(t, err, "create database %s", name)

	u, err := url.Parse(p.dsn)
	require.NoError(t, err, "parse dsn")

	u.Path = "/" + name

	return u.String()
}

// Pool opens a connection pool to dsn that is closed when the test ends.
func Pool(t *testing.T, dsn string) *pgxpool.Pool {
	t.Helper()

	pool, err := pgxpool.New(t.Context(), dsn)
	require.NoError(t, err, "connect to %s", dsn)

	t.Cleanup(pool.Close)

	return pool
}
