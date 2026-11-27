package list_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SuddenGunter/playground/go/tcapp/api/mockdata/list"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

type fakeLister struct {
	items    []mockdata.Item
	err      error
	gotLimit int
}

func (f *fakeLister) List(_ context.Context, limit int) ([]mockdata.Item, error) {
	f.gotLimit = limit

	if f.err != nil {
		return nil, f.err
	}

	return f.items[:min(limit, len(f.items))], nil
}

func seeded() *fakeLister {
	return &fakeLister{items: []mockdata.Item{
		{ID: 1, Name: "alpha"}, {ID: 2, Name: "beta"}, {ID: 3, Name: "gamma"},
	}}
}

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		store      *fakeLister
		wantStatus int
		wantLimit  int
		wantCount  int
	}{
		{name: "default limit", store: seeded(), wantStatus: http.StatusOK, wantLimit: list.DefaultLimit, wantCount: 3},
		{name: "custom limit", query: "?limit=2", store: seeded(), wantStatus: http.StatusOK, wantLimit: 2, wantCount: 2},
		{name: "max limit", query: "?limit=500", store: seeded(), wantStatus: http.StatusOK, wantLimit: list.MaxLimit, wantCount: 3},
		{name: "empty table renders []", store: &fakeLister{}, wantStatus: http.StatusOK, wantLimit: list.DefaultLimit},
		{name: "limit too big", query: "?limit=501", store: seeded(), wantStatus: http.StatusBadRequest},
		{name: "limit zero", query: "?limit=0", store: seeded(), wantStatus: http.StatusBadRequest},
		{name: "limit not a number", query: "?limit=all", store: seeded(), wantStatus: http.StatusBadRequest},
		{
			name: "store error", store: &fakeLister{err: errors.New("db is down")},
			wantStatus: http.StatusInternalServerError, wantLimit: list.DefaultLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/mock-data"+tt.query, nil)
			rec := httptest.NewRecorder()

			list.New(slog.New(slog.DiscardHandler), tt.store).ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
			assert.Equal(t, tt.wantLimit, tt.store.gotLimit, "limit passed to store")

			if rec.Code != http.StatusOK {
				return
			}

			if tt.wantCount == 0 {
				assert.JSONEq(t, `[]`, rec.Body.String(), "empty list must render as [], not null")
			}

			var got []mockdata.Item
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Len(t, got, tt.wantCount)
		})
	}
}
