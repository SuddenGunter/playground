package mockdata_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apimockdata "github.com/SuddenGunter/playground/go/tcapp/api/mockdata"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

var errDB = errors.New("db is down")

// fakeStore is an in-memory Store. Set err to make every call fail.
type fakeStore struct {
	items    []mockdata.Item
	err      error
	gotLimit int
}

func (f *fakeStore) Create(_ context.Context, name, value string) (mockdata.Item, error) {
	if f.err != nil {
		return mockdata.Item{}, f.err
	}

	item := mockdata.Item{ID: int64(len(f.items) + 1), Name: name, Value: value, CreatedAt: time.Unix(0, 0).UTC()}
	f.items = append(f.items, item)

	return item, nil
}

func (f *fakeStore) Get(_ context.Context, id int64) (mockdata.Item, error) {
	if f.err != nil {
		return mockdata.Item{}, f.err
	}

	for _, it := range f.items {
		if it.ID == id {
			return it, nil
		}
	}

	return mockdata.Item{}, mockdata.ErrNotFound
}

func (f *fakeStore) List(_ context.Context, limit int) ([]mockdata.Item, error) {
	f.gotLimit = limit

	if f.err != nil {
		return nil, f.err
	}

	return f.items[:min(limit, len(f.items))], nil
}

func newServer(store *fakeStore) http.Handler {
	mux := http.NewServeMux()
	apimockdata.New(slog.New(slog.DiscardHandler), store).Register(mux)

	return mux
}

func do(t *testing.T, h http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()

	var r io.Reader
	if body != "" {
		r = strings.NewReader(body)
	}

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), method, target, r))

	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

	return rec
}

func seeded() *fakeStore {
	return &fakeStore{items: []mockdata.Item{
		{ID: 1, Name: "alpha", Value: "a"},
		{ID: 2, Name: "beta", Value: "b"},
		{ID: 3, Name: "gamma", Value: "c"},
	}}
}

func TestCreate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantErrMsg string
	}{
		{name: "ok", body: `{"name":"n","value":"v"}`, wantStatus: http.StatusCreated},
		{name: "value optional", body: `{"name":"n"}`, wantStatus: http.StatusCreated},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantErrMsg: "invalid JSON body"},
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest, wantErrMsg: "invalid JSON body"},
		{name: "unknown field", body: `{"name":"n","extra":1}`, wantStatus: http.StatusBadRequest, wantErrMsg: "invalid JSON body"},
		{name: "blank name", body: `{"name":"  ","value":"v"}`, wantStatus: http.StatusBadRequest, wantErrMsg: "name is required"},
		{name: "store error", body: `{"name":"n"}`, storeErr: errDB, wantStatus: http.StatusInternalServerError, wantErrMsg: "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeStore{err: tt.storeErr}
			rec := do(t, newServer(store), http.MethodPost, "/mock-data", tt.body)

			require.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body)

			if tt.wantErrMsg != "" {
				assert.JSONEq(t, `{"error":"`+tt.wantErrMsg+`"}`, rec.Body.String())

				return
			}

			var got mockdata.Item
			decode(t, rec, &got)
			assert.Equal(t, int64(1), got.ID)
			assert.Equal(t, "n", got.Name)
		})
	}
}

func TestGet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		path       string
		storeErr   error
		wantStatus int
		wantName   string
	}{
		{name: "ok", path: "/mock-data/2", wantStatus: http.StatusOK, wantName: "beta"},
		{name: "not found", path: "/mock-data/99", wantStatus: http.StatusNotFound},
		{name: "non-numeric id", path: "/mock-data/abc", wantStatus: http.StatusBadRequest},
		{name: "zero id", path: "/mock-data/0", wantStatus: http.StatusBadRequest},
		{name: "negative id", path: "/mock-data/-1", wantStatus: http.StatusBadRequest},
		{name: "store error", path: "/mock-data/1", storeErr: errDB, wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := seeded()
			store.err = tt.storeErr
			rec := do(t, newServer(store), http.MethodGet, tt.path, "")

			require.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body)

			if tt.wantName == "" {
				return
			}

			var got mockdata.Item
			decode(t, rec, &got)
			assert.Equal(t, tt.wantName, got.Name)
		})
	}
}

func TestList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		query      string
		store      *fakeStore
		wantStatus int
		wantLimit  int
		wantCount  int
	}{
		{name: "default limit", store: seeded(), wantStatus: http.StatusOK, wantLimit: apimockdata.DefaultLimit, wantCount: 3},
		{name: "custom limit", query: "?limit=2", store: seeded(), wantStatus: http.StatusOK, wantLimit: 2, wantCount: 2},
		{name: "max limit", query: "?limit=500", store: seeded(), wantStatus: http.StatusOK, wantLimit: apimockdata.MaxLimit, wantCount: 3},
		{name: "empty table renders []", store: &fakeStore{}, wantStatus: http.StatusOK, wantLimit: apimockdata.DefaultLimit},
		{name: "limit too big", query: "?limit=501", store: seeded(), wantStatus: http.StatusBadRequest},
		{name: "limit zero", query: "?limit=0", store: seeded(), wantStatus: http.StatusBadRequest},
		{name: "limit not a number", query: "?limit=all", store: seeded(), wantStatus: http.StatusBadRequest},
		{name: "store error", store: &fakeStore{err: errDB}, wantStatus: http.StatusInternalServerError, wantLimit: apimockdata.DefaultLimit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := do(t, newServer(tt.store), http.MethodGet, "/mock-data"+tt.query, "")

			require.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body)
			assert.Equal(t, tt.wantLimit, tt.store.gotLimit, "limit passed to store")

			if rec.Code != http.StatusOK {
				return
			}

			if tt.wantCount == 0 {
				assert.JSONEq(t, `[]`, rec.Body.String(), "empty list must render as [], not null")
			}

			var got []mockdata.Item
			decode(t, rec, &got)
			assert.Len(t, got, tt.wantCount)
		})
	}
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), v), "decode body %q", rec.Body.String())
}
