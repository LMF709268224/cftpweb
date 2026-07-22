package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"candbff/config"
	"candbff/handler"
)

// Server 是 candbff 的核心结构
type Server struct {
	config     *config.Config
	grpcPool   *GrpcClientPool
	httpServer *http.Server
	casdoor    *CasdoorClient
}

func NewServer() *Server {
	return &Server{}
}

func (s *Server) Run(ctx context.Context) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	s.config = cfg

	s.casdoor = NewCasdoorClient(cfg.SecretConfig.Casdoor)

	transportCreds, err := config.LoadTransportCredentials()
	if err != nil {
		return fmt.Errorf("load downstream transport credentials: %w", err)
	}

	pool, err := NewGrpcClientPool(transportCreds)
	if err != nil {
		return err
	}
	s.grpcPool = pool

	h := handler.New(
		pool.Lms,
		pool.Mall,
		pool.Gcc,
		pool.Gprog,
		pool.Gmsg,
		pool.Creds,
		pool.Gexam,
		pool.Gmid,
		pool.Gpay,
		pool.Gmbr,
		pool.Gmail,
		os.Getenv(config.EnvCasdoorPublicEndpoint),
		s.config.SecretConfig.Casdoor.ClientID,
		s.config.SecretConfig.Casdoor.ClientSecret,
		s.config.SecretConfig.Casdoor.AppName,
		s.config.SecretConfig.Casdoor.OrgName,
	)
	serverErr := s.serveHTTP(s.buildRouter(h))

	select {
	case err := <-serverErr:
		s.gracefulShutdown()
		return err
	case <-ctx.Done():
		slog.Info("Received shutdown signal; starting graceful shutdown")
		s.gracefulShutdown()
		slog.Info("Service shut down safely")
		return nil
	}
}

func (s *Server) gracefulShutdown() {
	s.shutdownHTTP()

	if s.grpcPool != nil {
		s.grpcPool.Close()
		slog.Info("gRPC client connections closed")
	}
}
