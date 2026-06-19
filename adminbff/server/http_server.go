package server

import (
	"adminbff/config"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func (s *Server) serveHTTP(router http.Handler) <-chan error {
	addr := os.Getenv(config.EnvHTTPAddress)
	if addr == "" {
		addr = "0.0.0.0:8080"
	}

	s.httpServer = &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("AdminBFF HTTP is listening", "address", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server failed", "error", err)
			serverErr <- fmt.Errorf("HTTP server failed: %w", err)
		}
	}()
	return serverErr
}

func (s *Server) shutdownHTTP() {
	if s.httpServer == nil {
		return
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP server shutdown failed", "error", err)
	} else {
		slog.Info("HTTP server shut down gracefully")
	}
}
