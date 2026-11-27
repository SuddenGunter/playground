// Package respond writes JSON HTTP responses.
package respond

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type errorBody struct {
	Error string `json:"error"`
}

// JSON writes v as a JSON response with the given status.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// Headers are already sent; nothing useful to do on a write error.
	_ = json.NewEncoder(w).Encode(v)
}

// Error writes {"error": msg} with the given status.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, errorBody{Error: msg})
}

// InternalError logs err and writes a generic 500 response.
func InternalError(w http.ResponseWriter, r *http.Request, l *slog.Logger, err error) {
	l.ErrorContext(r.Context(), "request failed",
		slog.String("method", r.Method), slog.String("path", r.URL.Path), slog.Any("error", err))
	Error(w, http.StatusInternalServerError, "internal error")
}
