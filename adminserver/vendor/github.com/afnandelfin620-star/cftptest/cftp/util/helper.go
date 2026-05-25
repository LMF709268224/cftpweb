package util

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	// ULID regex: 26 chars, Base32 charset, first char in 0-7
	ulidRegex = regexp.MustCompile(`^[0-7][0-9A-HJKMNP-TV-Z]{25}$`)
	// UUID regex: standard 8-4-4-4-12 hex format
	uuidRegex = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
)

func IsRunningInK8s() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount")
	return err == nil
}

func GetNamespace() (string, error) {
	data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func GetEndpointAddress(envName, svcName, port string) string {
	endpoint := os.Getenv(envName)
	if endpoint == "" {
		endpoint = svcName
		namespace, err := GetNamespace()
		if err != nil {
			namespace = "default"
		}

		endpoint = fmt.Sprintf("%s.%s.svc.cluster.local:%s", svcName, namespace, port)
	}

	return endpoint
}

func GetGrpcClientTransportCreds() credentials.TransportCredentials {
	tlsDir := strings.TrimSpace(os.Getenv("TLS_DIR"))
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
		slog.Warn("gRPC: load ca faild", "ca_file", caFile, "error", err)
		return insecure.NewCredentials()
	}

	return credentials.NewTLS(tlsConfig)
}

func GetGrpcServerTransportCreds() credentials.TransportCredentials {
	tlsDir := os.Getenv("TLS_DIR")
	if tlsDir != "" {
		certFile := filepath.Join(tlsDir, "tls.crt")
		keyFile := filepath.Join(tlsDir, "tls.key")
		tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			slog.Error("failed to load TLS cert/key", "tls_dir", tlsDir, "cert_file", certFile, "key_file", keyFile, "error", err)
			return nil
		}

		tlsConfig := &tls.Config{
			MinVersion:   tls.VersionTLS12,
			Certificates: []tls.Certificate{tlsCert},
		}
		return credentials.NewTLS(tlsConfig)
	}

	return nil
}

// NewULID generates a ULID with the current UTC timestamp.
func NewULID() string {
	ms := ulid.Timestamp(time.Now().UTC())
	id, err := ulid.New(ms, rand.Reader)
	if err != nil {
		panic("NewULID: " + err.Error())
	}
	return id.String()
}

func IsValidULID(s string) bool {
	return ulidRegex.MatchString(s)
}

func IsValidUUID(s string) bool {
	return uuidRegex.MatchString(s)
}
