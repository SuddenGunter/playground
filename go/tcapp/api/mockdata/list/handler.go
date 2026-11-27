// Package list serves GET /mock-data.
package list

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/SuddenGunter/playground/go/tcapp/api/respond"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

const (
	// DefaultLimit is used when the request has no ?limit.
	DefaultLimit = 50
	// MaxLimit is the largest ?limit accepted.
	MaxLimit = 500
)

// Lister returns mock data items ordered by id.
type Lister interface {
	List(ctx context.Context, limit int) ([]mockdata.Item, error)
}

// Handler lists mock data items, up to ?limit=N.
type Handler struct {
	log   *slog.Logger
	store Lister
}

// New creates a Handler.
func New(l *slog.Logger, store Lister) *Handler {
	return &Handler{log: l, store: store}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	limit := DefaultLimit

	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			respond.Error(w, http.StatusBadRequest, "limit must be between 1 and "+strconv.Itoa(MaxLimit))

			return
		}

		limit = n
	}

	items, err := h.store.List(r.Context(), limit)
	if err != nil {
		respond.InternalError(w, r, h.log, err)

		return
	}

	if items == nil {
		items = []mockdata.Item{} // render [] instead of null
	}

	respond.JSON(w, http.StatusOK, items)
}
