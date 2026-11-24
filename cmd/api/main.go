package main

import (
	"context"
	"log"
	"net/http"

	"github.com/Covenantmondei/gopher_wallet/internal/config"
	"github.com/Covenantmondei/gopher_wallet/internal/database"
	"github.com/Covenantmondei/gopher_wallet/internal/middleware"
	"github.com/Covenantmondei/gopher_wallet/internal/response"
	"github.com/Covenantmondei/gopher_wallet/internal/server"
	"github.com/Covenantmondei/gopher_wallet/internal/transaction"
	"github.com/Covenantmondei/gopher_wallet/internal/user"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// Initialize PostgreSQL pool
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer pool.Close()

	// Run database migrations
	if err := database.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	// Initialize repositories
	userRepo := user.NewRepository(pool)
	walletRepo := wallet.NewRepository(pool)
	txRepo := transaction.NewRepository(pool)

	// Initialize services
	userService := user.NewService(pool, userRepo, walletRepo, cfg.JWTSecret, cfg.JWTExpirationHours)
	walletService := wallet.NewService(walletRepo)
	txService := transaction.NewService(pool, txRepo, walletRepo, userRepo)

	// Initialize handlers
	userHandler := user.NewHandler(userService)
	walletHandler := wallet.NewHandler(walletService)
	txHandler := transaction.NewHandler(txService)

	mux := http.NewServeMux()

	// Public routes
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		_ = response.JSON(w, http.StatusOK, response.Envelope{
			"status":  "ok",
			"service": "gopherwallet",
			"env":     cfg.Env,
		})
	})
	mux.HandleFunc("POST /api/v1/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/v1/auth/login", userHandler.Login)

	// Authenticated routes
	authMiddleware := middleware.Authenticate(cfg.JWTSecret)
	mux.Handle("GET /api/v1/users/me", authMiddleware(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("GET /api/v1/wallet", authMiddleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("GET /api/v1/wallet/balance", authMiddleware(http.HandlerFunc(walletHandler.GetBalance)))

	// Transaction & Transfer routes
	mux.Handle("POST /api/v1/transactions/credit", authMiddleware(http.HandlerFunc(txHandler.Credit)))
	mux.Handle("POST /api/v1/transactions/debit", authMiddleware(http.HandlerFunc(txHandler.Debit)))
	mux.Handle("POST /api/v1/transfers", authMiddleware(http.HandlerFunc(txHandler.Transfer)))
	mux.Handle("GET /api/v1/transactions", authMiddleware(http.HandlerFunc(txHandler.List)))
	mux.Handle("GET /api/v1/transactions/{id}", authMiddleware(http.HandlerFunc(txHandler.Get)))

	// Wrap mux with global middleware chain
	var handler http.Handler = mux
	handler = middleware.Logger(handler)
	handler = middleware.Recoverer(handler)

	srv := server.New(cfg, handler)

	if err := srv.Start(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
