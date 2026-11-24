package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Covenantmondei/gopher_wallet/internal/middleware"
	"github.com/Covenantmondei/gopher_wallet/internal/response"
)

func TestHealthCheck(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_ = response.JSON(w, http.StatusOK, response.Envelope{
			"status":  "ok",
			"service": "gopherwallet",
		})
	})

	handler := middleware.Recoverer(middleware.Logger(mux))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, rr.Code)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", contentType)
	}

	var res map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response JSON: %v", err)
	}

	if res["status"] != "ok" {
		t.Errorf("expected status 'ok', got %q", res["status"])
	}
	if res["service"] != "gopherwallet" {
		t.Errorf("expected service 'gopherwallet', got %q", res["service"])
	}
}
