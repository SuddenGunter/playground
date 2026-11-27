package respond_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/SuddenGunter/playground/go/tcapp/api/respond"
)

func TestJSON(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	respond.JSON(rec, http.StatusCreated, map[string]int{"n": 1})

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"n":1}`, rec.Body.String())
}

func TestError(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	respond.Error(rec, http.StatusBadRequest, "bad")

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.JSONEq(t, `{"error":"bad"}`, rec.Body.String())
}

func TestInternalError(t *testing.T) {
	t.Parallel()

	var logs bytes.Buffer

	logger := slog.New(slog.NewTextHandler(&logs, nil))
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()

	respond.InternalError(rec, req, logger, errors.New("db exploded"))

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"error":"internal error"}`, rec.Body.String(), "internal details must not leak")
	assert.Contains(t, logs.String(), "db exploded")
}
