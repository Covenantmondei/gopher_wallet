package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Covenantmondei/gopher_wallet/internal/config"
	"github.com/Covenantmondei/gopher_wallet/internal/database"
	"github.com/Covenantmondei/gopher_wallet/internal/user"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
)

func setupTestDB(t *testing.T) *user.Handler {
	cfg := config.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		t.Skipf("skipping test: database not reachable: %v", err)
	}

	if err := database.Migrate(ctx, pool, "../migrations"); err != nil {
		t.Fatalf("failed to run migrations: %v", err)
	}

	userRepo := user.NewRepository(pool)
	walletRepo := wallet.NewRepository(pool)
	userService := user.NewService(pool, userRepo, walletRepo, cfg.JWTSecret, cfg.JWTExpirationHours)
	return user.NewHandler(userService)
}

func TestRegisterSuccessAndDuplicate(t *testing.T) {
	handler := setupTestDB(t)

	uniqueEmail := fmt.Sprintf("test-%d@example.com", time.Now().UnixNano())
	payload := map[string]string{
		"email":     uniqueEmail,
		"password":  "supersecret123",
		"full_name": "Test User",
	}
	body, _ := json.Marshal(payload)

	// 1. Initial registration should succeed
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d. body: %s", rr.Code, rr.Body.String())
	}

	var res struct {
		User struct {
			ID       string `json:"id"`
			Email    string `json:"email"`
			FullName string `json:"full_name"`
		} `json:"user"`
		Wallet struct {
			ID       string `json:"id"`
			UserID   string `json:"user_id"`
			Balance  int64  `json:"balance"`
			Currency string `json:"currency"`
		} `json:"wallet"`
	}

	if err := json.Unmarshal(rr.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res.User.Email != uniqueEmail {
		t.Errorf("expected user email %s, got %s", uniqueEmail, res.User.Email)
	}
	if res.Wallet.Balance != 0 {
		t.Errorf("expected initial wallet balance 0, got %d", res.Wallet.Balance)
	}
	if res.Wallet.Currency != "NGN" {
		t.Errorf("expected currency NGN, got %s", res.Wallet.Currency)
	}
	if res.Wallet.UserID != res.User.ID {
		t.Errorf("expected wallet user_id %s to equal user id %s", res.Wallet.UserID, res.User.ID)
	}

	// 2. Duplicate registration with same email should return 409 Conflict
	dupReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	dupRR := httptest.NewRecorder()
	handler.Register(dupRR, dupReq)

	if dupRR.Code != http.StatusConflict {
		t.Errorf("expected status 409 Conflict on duplicate email, got %d", dupRR.Code)
	}
}

func TestRegisterInvalidPayload(t *testing.T) {
	handler := setupTestDB(t)

	// Weak password (< 8 chars)
	payload := map[string]string{
		"email":     "valid@example.com",
		"password":  "short",
		"full_name": "Short Pass User",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()
	handler.Register(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 Bad Request for short password, got %d", rr.Code)
	}
}
