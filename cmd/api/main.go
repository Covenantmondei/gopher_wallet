package main

import (
	"context"
	"log"
	"net/http"

	_ "github.com/Covenantmondei/gopher_wallet/docs"
	"github.com/Covenantmondei/gopher_wallet/internal/config"
	"github.com/Covenantmondei/gopher_wallet/internal/database"
	"github.com/Covenantmondei/gopher_wallet/internal/middleware"
	"github.com/Covenantmondei/gopher_wallet/internal/response"
	"github.com/Covenantmondei/gopher_wallet/internal/server"
	"github.com/Covenantmondei/gopher_wallet/internal/transaction"
	"github.com/Covenantmondei/gopher_wallet/internal/user"
	"github.com/Covenantmondei/gopher_wallet/internal/wallet"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// @title GopherWallet API
// @version 1.0
// @description A high-performance, secure Wallet & Transaction Management API in Go standard library.
// @contact.name API Support
// @contact.email support@example.com
// @license.name MIT
// @host localhost:8080
// @BasePath /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

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

	// Health check endpoint
	mux.HandleFunc("GET /health", healthHandler(cfg))

	// Swagger documentation UI
	mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/swagger/index.html", http.StatusMovedPermanently)
	})
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	// Public auth routes
	mux.HandleFunc("POST /api/auth/register", userHandler.Register)
	mux.HandleFunc("POST /api/auth/login", userHandler.Login)

	// Authenticated routes
	authMiddleware := middleware.Authenticate(cfg.JWTSecret)
	mux.Handle("GET /api/users/me", authMiddleware(http.HandlerFunc(userHandler.GetProfile)))
	mux.Handle("GET /api/wallet", authMiddleware(http.HandlerFunc(walletHandler.GetWallet)))
	mux.Handle("GET /api/wallet/balance", authMiddleware(http.HandlerFunc(walletHandler.GetBalance)))

	// Transaction & Transfer routes
	mux.Handle("POST /api/transactions/credit", authMiddleware(http.HandlerFunc(txHandler.Credit)))
	mux.Handle("POST /api/transactions/debit", authMiddleware(http.HandlerFunc(txHandler.Debit)))
	mux.Handle("POST /api/transfers", authMiddleware(http.HandlerFunc(txHandler.Transfer)))
	mux.Handle("GET /api/transactions", authMiddleware(http.HandlerFunc(txHandler.List)))
	mux.Handle("GET /api/transactions/{id}", authMiddleware(http.HandlerFunc(txHandler.Get)))

	// Wrap mux with global middleware chain
	var handler http.Handler = mux
	handler = middleware.Logger(handler)
	handler = middleware.Recoverer(handler)

	srv := server.New(cfg, handler)

	if err := srv.Start(); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

// HealthCheck godoc
// @Summary Check service health
// @Description Returns the health status and environment of the service
// @Tags System
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health [get]
func healthHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = response.JSON(w, http.StatusOK, response.Envelope{
			"status":  "ok",
			"service": "gopherwallet",
			"env":     cfg.Env,
		})
	}
}
