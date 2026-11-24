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
	"github.com/Covenantmondei/gopher_wallet/internal/transaction"
	"github.com/Covenantmondei/gopher_wallet/internal/user"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
)

type testRig struct {
	mux        *http.ServeMux
	userService *user.Service
	jwtSecret  string
}

func setupTestRig(t *testing.T) *testRig {
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
	txRepo := transaction.NewRepository(pool)

	userService := user.NewService(pool, userRepo, walletRepo, cfg.JWTSecret, cfg.JWTExpirationHours)
	walletService := wallet.NewService(walletRepo)
	txService := transaction.NewService(pool, txRepo, walletRepo, userRepo)

	userHandler := user.NewHandler(userService)
	walletHandler := wallet.NewHandler(walletService)
	txHandler := transaction.NewHandler(txService)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)

	authMiddleware := middleware.Authenticate(cfg.JWTSecret)
	mux.Handle("GET /api/v1/users/me", authMiddleware(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("GET /api/v1/wallet", authMiddleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("GET /api/v1/wallet/balance", authMiddleware(http.HandlerFunc(walletHandler.GetBalance)))
	mux.Handle("POST /api/v1/transactions/credit", authMiddleware(http.HandlerFunc(txHandler.Credit)))
	mux.Handle("POST /api/v1/transactions/debit", authMiddleware(http.HandlerFunc(txHandler.Debit)))
	mux.Handle("POST /api/v1/transfers", authMiddleware(http.HandlerFunc(txHandler.Transfer)))
	mux.Handle("GET /api/v1/transactions", authMiddleware(http.HandlerFunc(txHandler.List)))
	mux.Handle("GET /api/v1/transactions/{id}", authMiddleware(http.HandlerFunc(txHandler.Get)))

	return &testRig{
		mux:        mux,
		userService: userService,
		jwtSecret:  cfg.JWTSecret,
	}
}

func registerAndLogin(t *testing.T, rig *testRig, email, password string) string {
	regPayload, _ := json.Marshal(map[string]string{
		"email":     email,
		"password":  password,
		"full_name": "Test Account",
	})
	regReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(regPayload))
	regRR := httptest.NewRecorder()
	rig.mux.ServeHTTP(regRR, regReq)

	if regRR.Code != http.StatusCreated {
		t.Fatalf("failed to register %s: %s", email, regRR.Body.String())
	}

	loginPayload, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginPayload))
	loginRR := httptest.NewRecorder()
	rig.mux.ServeHTTP(loginRR, loginReq)

	if loginRR.Code != http.StatusOK {
		t.Fatalf("failed to login %s: %s", email, loginRR.Body.String())
	}

	var res struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(loginRR.Body.Bytes(), &res)
	return res.Token
}

func TestCreditDebitAndTransfers(t *testing.T) {
	rig := setupTestRig(t)

	userAEmail := fmt.Sprintf("usera-%d@example.com", time.Now().UnixNano())
	userBEmail := fmt.Sprintf("userb-%d@example.com", time.Now().UnixNano())
	pass := "password12345"

	tokenA := registerAndLogin(t, rig, userAEmail, pass)
	tokenB := registerAndLogin(t, rig, userBEmail, pass)

	// 1. Credit User A with ₦10,000 (1,000,000 kobo)
	creditPayload, _ := json.Marshal(map[string]any{
		"amount":      1000000,
		"description": "Initial funding",
	})
	creditReq := httptest.NewRequest(http.MethodPost, "/api/v1/transactions/credit", bytes.NewReader(creditPayload))
	creditReq.Header.Set("Authorization", "Bearer "+tokenA)
	creditRR := httptest.NewRecorder()
	rig.mux.ServeHTTP(creditRR, creditReq)

	if creditRR.Code != http.StatusOK {
		t.Fatalf("credit failed: %d: %s", creditRR.Code, creditRR.Body.String())
	}

	// 2. Transfer ₦4,000 (400,000 kobo) from User A to User B with Idempotency Key
	transferKey := fmt.Sprintf("idemp-%d", time.Now().UnixNano())
	transferPayload, _ := json.Marshal(map[string]any{
		"recipient_email": userBEmail,
		"amount":          400000,
		"description":     "Dinner share",
	})
	transferReq := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(transferPayload))
	transferReq.Header.Set("Authorization", "Bearer "+tokenA)
	transferReq.Header.Set("Idempotency-Key", transferKey)
	transferRR := httptest.NewRecorder()
	rig.mux.ServeHTTP(transferRR, transferReq)

	if transferRR.Code != http.StatusOK {
		t.Fatalf("transfer failed: %d: %s", transferRR.Code, transferRR.Body.String())
	}

	// 3. Repeat EXACT SAME transfer with same key -> should return cached response (HIT)
	transferReq2 := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(transferPayload))
	transferReq2.Header.Set("Authorization", "Bearer "+tokenA)
	transferReq2.Header.Set("Idempotency-Key", transferKey)
	transferRR2 := httptest.NewRecorder()
	rig.mux.ServeHTTP(transferRR2, transferReq2)

	if transferRR2.Code != http.StatusOK {
		t.Fatalf("idempotent replay failed: %d: %s", transferRR2.Code, transferRR2.Body.String())
	}
	if transferRR2.Header().Get("X-Cache-Lookup") != "HIT" {
		t.Errorf("expected X-Cache-Lookup: HIT, got %s", transferRR2.Header().Get("X-Cache-Lookup"))
	}

	// 4. Reuse same Idempotency-Key with DIFFERENT payload -> 409 Conflict
	conflictPayload, _ := json.Marshal(map[string]any{
		"recipient_email": userBEmail,
		"amount":          999999,
		"description":     "Different amount",
	})
	conflictReq := httptest.NewRequest(http.MethodPost, "/api/v1/transfers", bytes.NewReader(conflictPayload))
	conflictReq.Header.Set("Authorization", "Bearer "+tokenA)
	conflictReq.Header.Set("Idempotency-Key", transferKey)
	conflictRR := httptest.NewRecorder()
	rig.mux.ServeHTTP(conflictRR, conflictReq)

	if conflictRR.Code != http.StatusConflict {
		t.Errorf("expected 409 Conflict on payload tampering, got %d", conflictRR.Code)
	}

	// 5. Verify User A balance is now ₦6,000 (600,000 kobo)
	balReqA := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/balance", nil)
	balReqA.Header.Set("Authorization", "Bearer "+tokenA)
	balRRA := httptest.NewRecorder()
	rig.mux.ServeHTTP(balRRA, balReqA)

	var balResA struct {
		Balance int64 `json:"balance"`
	}
	_ = json.Unmarshal(balRRA.Body.Bytes(), &balResA)
	if balResA.Balance != 600000 {
		t.Errorf("expected User A balance 600000, got %d", balResA.Balance)
	}

	// 6. Verify User B balance is now ₦4,000 (400,000 kobo)
	balReqB := httptest.NewRequest(http.MethodGet, "/api/v1/wallet/balance", nil)
	balReqB.Header.Set("Authorization", "Bearer "+tokenB)
	balRRB := httptest.NewRecorder()
	rig.mux.ServeHTTP(balRRB, balReqB)

	var balResB struct {
		Balance int64 `json:"balance"`
	}
	_ = json.Unmarshal(balRRB.Body.Bytes(), &balResB)
	if balResB.Balance != 400000 {
		t.Errorf("expected User B balance 400000, got %d", balResB.Balance)
	}

	// 7. Verify transaction list on User A
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	listReq.Header.Set("Authorization", "Bearer "+tokenA)
	listRR := httptest.NewRecorder()
	rig.mux.ServeHTTP(listRR, listReq)

	if listRR.Code != http.StatusOK {
		t.Fatalf("transaction list failed: %d: %s", listRR.Code, listRR.Body.String())
	}
	var listRes struct {
		Transactions []transaction.Transaction `json:"transactions"`
	}
	_ = json.Unmarshal(listRR.Body.Bytes(), &listRes)
	if len(listRes.Transactions) != 2 { // 1 credit + 1 debit transfer
		t.Errorf("expected 2 transactions for User A, got %d", len(listRes.Transactions))
	}
}
