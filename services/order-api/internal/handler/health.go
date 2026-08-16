package handler

import (
	"encoding/json"
	"net/http"
)

type HealthHandler struct {
	ServiceName string
}

type healthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func (h HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(healthResponse{
		Status:  "ok",
		Service: h.ServiceName,
	})
}
