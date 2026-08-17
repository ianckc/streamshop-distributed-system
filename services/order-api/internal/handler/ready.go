package handler

import (
	"log/slog"
	"net/http"

	"github.com/ianckc/distributed-systems/services/order-api/internal/store"
)

type ReadyHandler struct {
	ServiceName string
	Store       store.OrderStore
}

func (h ReadyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := h.Store.Ping(r.Context()); err != nil {
		slog.Error("ready ping failed", "error", err)
		writeJSON(w, http.StatusServiceUnavailable, errorResponse{Error: "not ready"})
		return
	}

	writeJSON(w, http.StatusOK, healthResponse{
		Status:  "ok",
		Service: h.ServiceName,
	})
}
