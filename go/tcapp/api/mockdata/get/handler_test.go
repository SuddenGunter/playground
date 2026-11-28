package get_test

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

	"github.com/SuddenGunter/playground/go/tcapp/api/mockdata/get"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

type fakeGetter struct {
	items map[int64]mockdata.Item
	err   error
}

func (f *fakeGetter) Get(_ context.Context, id int64) (mockdata.Item, error) {
	if f.err != nil {
		return mockdata.Item{}, f.err
	}

	item, ok := f.items[id]
	if !ok {
		return mockdata.Item{}, mockdata.ErrNotFound
	}

	return item, nil
}

func TestHandler(t *testing.T) {
	t.Parallel()

	items := map[int64]mockdata.Item{
		1: {ID: 1, Name: "alpha", Value: "a"},
		2: {ID: 2, Name: "beta", Value: "b"},
	}

	tests := []struct {
		name       string
		id         string
		storeErr   error
		wantStatus int
		wantName   string
	}{
		{name: "ok", id: "2", wantStatus: http.StatusOK, wantName: "beta"},
		{name: "not found", id: "99", wantStatus: http.StatusNotFound},
		{name: "non-numeric id", id: "abc", wantStatus: http.StatusBadRequest},
		{name: "zero id", id: "0", wantStatus: http.StatusBadRequest},
		{name: "negative id", id: "-1", wantStatus: http.StatusBadRequest},
		{name: "store error", id: "1", storeErr: errors.New("db is down"), wantStatus: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/mock-data/"+tt.id, nil)
			req.SetPathValue("id", tt.id)

			rec := httptest.NewRecorder()

			get.New(slog.New(slog.DiscardHandler), &fakeGetter{items: items, err: tt.storeErr}).ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			if tt.wantName == "" {
				return
			}

			var got mockdata.Item
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Equal(t, tt.wantName, got.Name)
		})
	}
}
