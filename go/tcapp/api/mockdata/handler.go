// Package mockdata exposes HTTP endpoints for creating and querying mock data.
package mockdata

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/SuddenGunter/playground/go/tcapp/mockdata"
)

const (
	// DefaultLimit is used by the list endpoint when no limit is given.
	DefaultLimit = 50
	// MaxLimit is the largest limit the list endpoint accepts.
	MaxLimit = 500

	maxBodyBytes = 1 << 20
)

// Store is the storage the handlers need. *mockdata.Store implements it.
type Store interface {
	Create(ctx context.Context, name, value string) (mockdata.Item, error)
	Get(ctx context.Context, id int64) (mockdata.Item, error)
	List(ctx context.Context, limit int) ([]mockdata.Item, error)
}

// Handler serves the mock data endpoints.
type Handler struct {
	log   *slog.Logger
	store Store
}

// New creates a Handler.
func New(l *slog.Logger, store Store) *Handler {
	return &Handler{log: l, store: store}
}

// Register adds the mock data routes to mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /mock-data", h.create)
	mux.HandleFunc("GET /mock-data", h.list)
	mux.HandleFunc("GET /mock-data/{id}", h.get)
}

type createRequest struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()

	var req createRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")

		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeError(w, http.StatusBadRequest, "name is required")

		return
	}

	item, err := h.store.Create(r.Context(), req.Name, req.Value)
	if err != nil {
		h.internalError(w, r, err)

		return
	}

	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id must be a positive integer")

		return
	}

	item, err := h.store.Get(r.Context(), id)
	if errors.Is(err, mockdata.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")

		return
	}

	if err != nil {
		h.internalError(w, r, err)

		return
	}

	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	limit := DefaultLimit

	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > MaxLimit {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and "+strconv.Itoa(MaxLimit))

			return
		}

		limit = n
	}

	items, err := h.store.List(r.Context(), limit)
	if err != nil {
		h.internalError(w, r, err)

		return
	}

	if items == nil {
		items = []mockdata.Item{} // render [] instead of null
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) internalError(w http.ResponseWriter, r *http.Request, err error) {
	h.log.ErrorContext(r.Context(), "request failed",
		slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Any("error", err))
	writeError(w, http.StatusInternalServerError, "internal error")
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorResponse{Error: msg})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Headers are already sent; nothing useful to do on a write error.
	_ = json.NewEncoder(w).Encode(v)
}
