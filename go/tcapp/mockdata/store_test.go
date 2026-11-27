//go:build integration

package mockdata_test

import (
	"errors"
	"testing"

	"github.com/SuddenGunter/playground/go/tcapp/internal/testenv"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

// newStore returns a Store on a fresh, migrated database.
func newStore(t *testing.T) *mockdata.Store {
	t.Helper()

	pool := testenv.Pool(t, pg.NewDatabase(t))

	if err := mockdata.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Migrate must be safe to run on every startup.
	if err := mockdata.Migrate(t.Context(), pool); err != nil {
		t.Fatalf("second migrate: %v", err)
	}

	return mockdata.NewStore(pool)
}

func TestStore_CreateAndGet(t *testing.T) {
	t.Parallel()

	s := newStore(t)

	created, err := s.Create(t.Context(), "alpha", "1")
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	if created.ID == 0 || created.CreatedAt.IsZero() {
		t.Fatalf("generated fields not set: %+v", created)
	}

	got, err := s.Get(t.Context(), created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if !got.CreatedAt.Equal(created.CreatedAt) || got.ID != created.ID || got.Name != "alpha" || got.Value != "1" {
		t.Errorf("got %+v, want %+v", got, created)
	}
}

func TestStore_GetNotFound(t *testing.T) {
	t.Parallel()

	s := newStore(t)

	if _, err := s.Get(t.Context(), 42); !errors.Is(err, mockdata.ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestStore_List(t *testing.T) {
	t.Parallel()

	s := newStore(t)

	empty, err := s.List(t.Context(), 10)
	if err != nil {
		t.Fatalf("list empty: %v", err)
	}

	if len(empty) != 0 {
		t.Fatalf("expected no items, got %d", len(empty))
	}

	for _, name := range []string{"a", "b", "c"} {
		if _, err := s.Create(t.Context(), name, name); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
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
			if err != nil {
				t.Fatalf("list: %v", err)
			}

			if len(items) != len(tt.want) {
				t.Fatalf("got %d items, want %d", len(items), len(tt.want))
			}

			for i, it := range items {
				if it.Name != tt.want[i] {
					t.Errorf("items[%d].Name = %q, want %q", i, it.Name, tt.want[i])
				}
			}
		})
	}
}
