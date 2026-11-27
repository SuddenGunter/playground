//go:build integration

package mockdata_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SuddenGunter/playground/go/tcapp/internal/testenv"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

// newStore returns a Store on a fresh, migrated database.
func newStore(t *testing.T) *mockdata.Store {
	t.Helper()

	pool := testenv.Pool(t, pg.NewDatabase(t))

	require.NoError(t, mockdata.Migrate(t.Context(), pool), "migrate")
	// Migrate must be safe to run on every startup.
	require.NoError(t, mockdata.Migrate(t.Context(), pool), "second migrate")

	return mockdata.NewStore(pool)
}

func TestStore_CreateAndGet(t *testing.T) {
	t.Parallel()

	s := newStore(t)

	created, err := s.Create(t.Context(), "alpha", "1")
	require.NoError(t, err)
	require.NotZero(t, created.ID, "id not generated")
	require.False(t, created.CreatedAt.IsZero(), "created_at not set")

	got, err := s.Get(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "alpha", got.Name)
	assert.Equal(t, "1", got.Value)
	assert.WithinDuration(t, created.CreatedAt, got.CreatedAt, 0)
}

func TestStore_GetNotFound(t *testing.T) {
	t.Parallel()

	s := newStore(t)

	_, err := s.Get(t.Context(), 42)
	assert.ErrorIs(t, err, mockdata.ErrNotFound)
}

func TestStore_List(t *testing.T) {
	t.Parallel()

	s := newStore(t)

	empty, err := s.List(t.Context(), 10)
	require.NoError(t, err)
	require.Empty(t, empty)

	for _, name := range []string{"a", "b", "c"} {
		_, err := s.Create(t.Context(), name, name)
		require.NoError(t, err, "create %s", name)
	}

	tests := []struct {
		name  string
		limit int
		want  []string
	}{
		{name: "all", limit: 10, want: []string{"a", "b", "c"}},
		{name: "limited", limit: 2, want: []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			items, err := s.List(t.Context(), tt.limit)
			require.NoError(t, err)

			got := make([]string, 0, len(items))
			for _, it := range items {
				got = append(got, it.Name)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
