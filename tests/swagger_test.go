package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/Covenantmondei/gopher_wallet/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func TestSwaggerEndpoints(t *testing.T) {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	// 1. Test redirect from /swagger to /swagger/index.html
	reqRedirect := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	rrRedirect := httptest.NewRecorder()
	mux.ServeHTTP(rrRedirect, reqRedirect)

	if rrRedirect.Code != http.StatusMovedPermanently {
		t.Errorf("expected 301 Moved Permanently for /swagger, got %d", rrRedirect.Code)
	}
	if loc := rrRedirect.Header().Get("Location"); loc != "/swagger/index.html" {
		t.Errorf("expected redirect Location /swagger/index.html, got %s", loc)
	}

	// 2. Test /swagger/index.html returns 200 OK
	reqUI := httptest.NewRequest(http.MethodGet, "/swagger/index.html", nil)
	rrUI := httptest.NewRecorder()
	mux.ServeHTTP(rrUI, reqUI)

	if rrUI.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /swagger/index.html, got %d", rrUI.Code)
	}
	body := rrUI.Body.String()
	if !strings.Contains(body, "swagger-ui") {
		t.Errorf("expected HTML body to contain 'swagger-ui', got: %s", body[:min(len(body), 200)])
	}

	// 3. Test /swagger/doc.json returns 200 OK with valid Swagger schema
	reqDoc := httptest.NewRequest(http.MethodGet, "/swagger/doc.json", nil)
	rrDoc := httptest.NewRecorder()
	mux.ServeHTTP(rrDoc, reqDoc)

	if rrDoc.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for /swagger/doc.json, got %d", rrDoc.Code)
	}

	var docMap map[string]any
	if err := json.Unmarshal(rrDoc.Body.Bytes(), &docMap); err != nil {
		t.Fatalf("failed to decode doc.json: %v", err)
	}

	paths, ok := docMap["paths"].(map[string]any)
	if !ok {
		t.Fatal("expected 'paths' object in doc.json")
	}

	// Verify key endpoints are registered in Swagger
	expectedEndpoints := []string{
		"/health",
		"/api/auth/register",
		"/api/auth/login",
		"/api/users/me",
		"/api/wallet",
		"/api/wallet/balance",
		"/api/transactions/credit",
		"/api/transactions/debit",
		"/api/transfers",
		"/api/transactions",
		"/api/transactions/{id}",
	}

	for _, ep := range expectedEndpoints {
		if _, exists := paths[ep]; !exists {
			t.Errorf("expected endpoint %q in Swagger docs paths", ep)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
