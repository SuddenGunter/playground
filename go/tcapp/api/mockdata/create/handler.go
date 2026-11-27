// Package create serves POST /mock-data.
package create

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/SuddenGunter/playground/go/tcapp/api/respond"
	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

const maxBodyBytes = 1 << 20

// Creator stores a new mock data item.
type Creator interface {
	Create(ctx context.Context, name, value string) (mockdata.Item, error)
}

// Handler creates a mock data item from a JSON body {"name": "...", "value": "..."}.
type Handler struct {
	log   *slog.Logger
	store Creator
}

// New creates a Handler.
func New(l *slog.Logger, store Creator) *Handler {
	return &Handler{log: l, store: store}
}

type request struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()

	var req request
	if err := dec.Decode(&req); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid JSON body")

		return
	}

	if strings.TrimSpace(req.Name) == "" {
		respond.Error(w, http.StatusBadRequest, "name is required")

		return
	}

	item, err := h.store.Create(r.Context(), req.Name, req.Value)
	if err != nil {
		respond.InternalError(w, r, h.log, err)

		return
	}

	respond.JSON(w, http.StatusCreated, item)
}
