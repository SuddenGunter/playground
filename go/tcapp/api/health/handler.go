// Package health provides the liveness endpoint.
package health

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Status string `json:"status"`
}

// Handler returns an http.Handler that always reports the service as healthy.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		// Headers are already sent; nothing useful to do on a write error.
		_ = json.NewEncoder(w).Encode(response{Status: "ok"})
	})
}
