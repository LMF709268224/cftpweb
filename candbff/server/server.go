package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"candbff/config"
	"candbff/handler"

	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
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

	transportCreds := getCfgServerTransportCreds()

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
		os.Getenv(config.EnvCasdoorPublicEndpoint),
		s.config.SecretConfig.Casdoor.ClientID,
		s.config.SecretConfig.Casdoor.ClientSecret,
		s.config.SecretConfig.Casdoor.AppName,
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

func getCfgServerTransportCreds() credentials.TransportCredentials {
	tlsDir := strings.TrimSpace(os.Getenv(config.EnvTLSDir))
	if tlsDir == "" {
		return insecure.NewCredentials()
	}

	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS12}
	caFile := filepath.Join(tlsDir, "ca.crt")
	if caPEM, err := os.ReadFile(caFile); err == nil {
		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(caPEM); !ok {
			slog.Warn("gRPC: failed to append CA cert", "ca_file", caFile)
			return insecure.NewCredentials()
		}

		tlsConfig.RootCAs = pool
	} else {
		slog.Warn("gRPC: load ca failed", "ca_file", caFile, "error", err)
		return insecure.NewCredentials()
	}

	return credentials.NewTLS(tlsConfig)
}
