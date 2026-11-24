package main

import (
	"context"
	"log"
	"time"

	"github.com/Covenantmondei/gopher_wallet/internal/config"
	"github.com/Covenantmondei/gopher_wallet/internal/database"
)

func main() {
	cfg := config.Load()

	log.Println("Starting database migrations...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := database.Migrate(ctx, pool, "migrations"); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Println("all migrations applied successfully!")
}
