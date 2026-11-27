package create_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/SuddenGunter/playground/go/tcapp/api/mockdata/create"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

type fakeCreator struct {
	err       error
	gotName   string
	gotValue  string
	callCount int
}

func (f *fakeCreator) Create(_ context.Context, name, value string) (mockdata.Item, error) {
	f.callCount++
	f.gotName, f.gotValue = name, value

	if f.err != nil {
		return mockdata.Item{}, f.err
	}

	return mockdata.Item{ID: 1, Name: name, Value: value}, nil
}

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		body       string
		storeErr   error
		wantStatus int
		wantErrMsg string
		wantStored bool
	}{
		{name: "ok", body: `{"name":"n","value":"v"}`, wantStatus: http.StatusCreated, wantStored: true},
		{name: "value optional", body: `{"name":"n"}`, wantStatus: http.StatusCreated, wantStored: true},
		{name: "invalid json", body: `{`, wantStatus: http.StatusBadRequest, wantErrMsg: "invalid JSON body"},
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest, wantErrMsg: "invalid JSON body"},
		{name: "unknown field", body: `{"name":"n","extra":1}`, wantStatus: http.StatusBadRequest, wantErrMsg: "invalid JSON body"},
		{name: "blank name", body: `{"name":"  ","value":"v"}`, wantStatus: http.StatusBadRequest, wantErrMsg: "name is required"},
		{
			name: "store error", body: `{"name":"n"}`, storeErr: errors.New("db is down"),
			wantStatus: http.StatusInternalServerError, wantErrMsg: "internal error", wantStored: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			store := &fakeCreator{err: tt.storeErr}
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/mock-data", strings.NewReader(tt.body))
			rec := httptest.NewRecorder()

			create.New(slog.New(slog.DiscardHandler), store).ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code, "body: %s", rec.Body)
			assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))

			if !tt.wantStored {
				assert.Zero(t, store.callCount, "store must not be called for invalid input")
			}

			if tt.wantErrMsg != "" {
				assert.JSONEq(t, `{"error":"`+tt.wantErrMsg+`"}`, rec.Body.String())

				return
			}

			var got mockdata.Item
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
			assert.Equal(t, int64(1), got.ID)
			assert.Equal(t, store.gotName, got.Name)
			assert.Equal(t, store.gotValue, got.Value)
		})
	}
}
