// Package get serves GET /mock-data/{id}.
package get

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SuddenGunter/playground/go/tcapp/api/respond"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

// Getter looks up a mock data item by id.
type Getter interface {
	Get(ctx context.Context, id int64) (mockdata.Item, error)
}

// Handler returns one mock data item. The route must define an {id} path wildcard.
type Handler struct {
	log   *slog.Logger
	store Getter
}

// New creates a Handler.
func New(l *slog.Logger, store Getter) *Handler {
	return &Handler{log: l, store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		respond.Error(w, http.StatusBadRequest, "id must be a positive integer")

		return
	}

	item, err := h.store.Get(r.Context(), id)
	if errors.Is(err, mockdata.ErrNotFound) {
		respond.Error(w, http.StatusNotFound, "not found")

		return
	}

	if err != nil {
		respond.InternalError(w, r, h.log, err)

		return
	}

	respond.JSON(w, http.StatusOK, item)
}
