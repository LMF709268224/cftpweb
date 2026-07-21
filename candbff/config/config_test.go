package config

import (
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadTransportCredentials(t *testing.T) {
	t.Run("uses insecure credentials when TLS_DIR is empty", func(t *testing.T) {
		t.Setenv(EnvTLSDir, "  ")

		creds, err := LoadTransportCredentials()
		if err != nil {
			t.Fatalf("LoadTransportCredentials() error = %v", err)
		}
		if got := creds.Info().SecurityProtocol; got != "insecure" {
			t.Fatalf("security protocol = %q, want insecure", got)
		}
	})

	t.Run("rejects missing CA certificate", func(t *testing.T) {
		t.Setenv(EnvTLSDir, t.TempDir())

		_, err := LoadTransportCredentials()
		if err == nil || !strings.Contains(err.Error(), "read gRPC CA certificate") {
			t.Fatalf("LoadTransportCredentials() error = %v, want missing CA error", err)
		}
	})

	t.Run("rejects invalid CA certificate", func(t *testing.T) {
		tlsDir := t.TempDir()
		if err := os.WriteFile(filepath.Join(tlsDir, "ca.crt"), []byte("not a certificate"), 0o600); err != nil {
			t.Fatalf("write invalid CA: %v", err)
		}
		t.Setenv(EnvTLSDir, tlsDir)

		_, err := LoadTransportCredentials()
		if err == nil || !strings.Contains(err.Error(), "no valid certificates found") {
			t.Fatalf("LoadTransportCredentials() error = %v, want invalid CA error", err)
		}
	})

	t.Run("loads valid CA certificate", func(t *testing.T) {
		tlsServer := httptest.NewTLSServer(nil)
		t.Cleanup(tlsServer.Close)

		tlsDir := t.TempDir()
		caPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: tlsServer.Certificate().Raw,
		})
		if err := os.WriteFile(filepath.Join(tlsDir, "ca.crt"), caPEM, 0o600); err != nil {
			t.Fatalf("write CA certificate: %v", err)
		}
		t.Setenv(EnvTLSDir, tlsDir)

		creds, err := LoadTransportCredentials()
		if err != nil {
			t.Fatalf("LoadTransportCredentials() error = %v", err)
		}
		if got := creds.Info().SecurityProtocol; got != "tls" {
			t.Fatalf("security protocol = %q, want tls", got)
		}
	})
}
