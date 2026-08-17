package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ianckc/distributed-systems/services/order-api/internal/handler"
)

func TestReadyHandler(t *testing.T) {
	t.Run("GET returns ok when store pings", func(t *testing.T) {
		h := handler.ReadyHandler{
			ServiceName: "order-api",
			Store: fakeOrderStore{
				pingFn: func(context.Context) error { return nil },
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}

		var body struct {
			Status  string `json:"status"`
			Service string `json:"service"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Status != "ok" {
			t.Fatalf("status = %q, want ok", body.Status)
		}
		if body.Service != "order-api" {
			t.Fatalf("service = %q, want order-api", body.Service)
		}
	})

	t.Run("GET returns 503 when store ping fails", func(t *testing.T) {
		h := handler.ReadyHandler{
			ServiceName: "order-api",
			Store: fakeOrderStore{
				pingFn: func(context.Context) error { return errors.New("db down") },
			},
		}

		req := httptest.NewRequest(http.MethodGet, "/ready", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
		}

		var body struct {
			Error string `json:"error"`
		}
		if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
			t.Fatalf("decode response: %v", err)
		}
		if body.Error != "not ready" {
			t.Fatalf("error = %q, want not ready", body.Error)
		}
	})

	t.Run("POST returns method not allowed", func(t *testing.T) {
		h := handler.ReadyHandler{
			ServiceName: "order-api",
			Store: fakeOrderStore{
				pingFn: func(context.Context) error {
					t.Fatal("store should not be called")
					return nil
				},
			},
		}

		req := httptest.NewRequest(http.MethodPost, "/ready", nil)
		rec := httptest.NewRecorder()

		h.ServeHTTP(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}
