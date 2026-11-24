package server

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Covenantmondei/gopher_wallet/internal/config"
)

type Server struct {
	cfg        *config.Config
	httpServer *http.Server
}

func New(cfg *config.Config, handler http.Handler) *Server {
	addr := fmt.Sprintf(":%s", cfg.Port)

	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:         addr,
			Handler:      handler,
			IdleTimeout:  time.Minute,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
	}
}

func (s *Server) Start() error {
	shutdownError := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		sig := <-quit

		log.Printf("caught signal %s: shutting down gracefully...", sig.String())

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		shutdownError <- s.httpServer.Shutdown(ctx)
	}()

	log.Printf("GopherWallet server listening on %s (env: %s)", s.httpServer.Addr, s.cfg.Env)

	err := s.httpServer.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdownError
	if err != nil {
		return err
	}

	log.Println("server stopped cleanly")
	return nil
}
