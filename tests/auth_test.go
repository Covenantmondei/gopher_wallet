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
	"github.com/Covenantmondei/gopher_wallet/internal/middleware"
	"github.com/Covenantmondei/gopher_wallet/internal/user"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
)

func TestAuthAndWalletEndpoints(t *testing.T) {
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
	userHandler := user.NewHandler(userService)

	walletService := wallet.NewService(walletRepo)
	walletHandler := wallet.NewHandler(walletService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)

	authMiddleware := middleware.Authenticate(cfg.JWTSecret)
	mux.Handle("GET /api/v1/users/me", authMiddleware(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("GET /api/v1/wallet", authMiddleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("GET /api/v1/wallet/balance", authMiddleware(http.HandlerFunc(walletHandler.GetBalance)))

	email := fmt.Sprintf("auth-test-%d@example.com", time.Now().UnixNano())
	password := "correct-password-123"

	// 1. Register user
	regPayload, _ := json.Marshal(map[string]string{
		"email":     email,
		"password":  password,
		"full_name": "Auth Tester",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regPayload))
	regRR := httptest.NewRecorder()
	mux.ServeHTTP(regRR, regReq)

	if regRR.Code != http.StatusCreated {
		t.Fatalf("registration failed: %d: %s", regRR.Code, regRR.Body.String())
	}

	// 2. Failed login with wrong password
	badLoginPayload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": "wrong-password",
	})
	badLoginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(badLoginPayload))
	badLoginRR := httptest.NewRecorder()
	mux.ServeHTTP(badLoginRR, badLoginReq)

	if badLoginRR.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong password, got %d", badLoginRR.Code)
	}

	// 3. Successful login
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	loginRR := httptest.NewRecorder()
	mux.ServeHTTP(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid login, got %d. body: %s", loginRR.Code, loginRR.Body.String())
	}

	var loginRes struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(loginRR.Body.Bytes(), &loginRes); err != nil {
		t.Fatalf("failed to parse login response: %v", err)
	}
	if loginRes.Token == "" {
		t.Fatal("expected token to be non-empty")
	}

	// 4. Access protected GET /api/v1/users/me without token -> 401 Unauthorized
	unauthReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	unauthRR := httptest.NewRecorder()
	mux.ServeHTTP(unauthRR, unauthReq)

	if unauthRR.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for unauthenticated profile request, got %d", unauthRR.Code)
	}

	// 5. Access protected GET /api/v1/users/me with valid token -> 200 OK
	meReq := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
	meRR := httptest.NewRecorder()
	mux.ServeHTTP(meRR, meReq)

	if meRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for authenticated profile request, got %d: %s", meRR.Code, meRR.Body.String())
	}

	// 6. Access protected GET /api/v1/wallet
	walletReq := httptest.NewRequest(http.MethodGet, "/api/v1/wallet", nil)
	walletReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
	walletRR := httptest.NewRecorder()
	mux.ServeHTTP(walletRR, walletReq)

	if walletRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for wallet request, got %d: %s", walletRR.Code, walletRR.Body.String())
	}

	// 7. Access protected GET /api/v1/wallet/balance
	balReq := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/balance", nil)
	balReq.Header.Set("Authorization", "Bearer "+loginRes.Token)
	balRR := httptest.NewRecorder()
	mux.ServeHTTP(balRR, balReq)

	if balRR.Code != http.StatusOK {
		t.Fatalf("expected 200 for balance request, got %d: %s", balRR.Code, balRR.Body.String())
	}

	var balRes struct {
		Balance          int64  `json:"balance"`
		BalanceFormatted string `json:"balance_formatted"`
		Currency         string `json:"currency"`
	}
	if err := json.Unmarshal(balRR.Body.Bytes(), &balRes); err != nil {
		t.Fatalf("failed to parse balance response: %v", err)
	}
	if balRes.Balance != 0 || balRes.BalanceFormatted != "0.00" || balRes.Currency != "NGN" {
		t.Errorf("unexpected balance response values: %+v", balRes)
	}
}
