package config

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	cfgservepb "github.com/afnandelfin620-star/cftptest/cftp/cfgserver"
	"github.com/afnandelfin620-star/cftptest/cftp/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// CasdoorConfig Casdoor IAM 连接配置。
type CasdoorConfig struct {
	ClientID     string `json:"ClientID"`     // Casdoor 应用 Client ID
	ClientSecret string `json:"ClientSecret"` // Casdoor 应用 Client Secret
	OrgName      string `json:"OrgName"`      // Casdoor 组织名
	AppName      string `json:"AppName"`      // Casdoor 应用名
	Certificate  string `json:"Certificate"`  // Casdoor 服务端 JWT 公钥证书 (PEM)
}

type SecretConfig struct {
	Casdoor CasdoorConfig `json:"Casdoor"`
}

type Config struct {
	SecretConfig SecretConfig
}

func LoadTransportCredentials() (credentials.TransportCredentials, error) {
	tlsDir := strings.TrimSpace(os.Getenv(EnvTLSDir))
	if tlsDir == "" {
		return insecure.NewCredentials(), nil
	}

	caFile := filepath.Join(tlsDir, "ca.crt")
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read gRPC CA certificate %q: %w", caFile, err)
	}

	pool := x509.NewCertPool()
	if ok := pool.AppendCertsFromPEM(caPEM); !ok {
		return nil, fmt.Errorf("parse gRPC CA certificate %q: no valid certificates found", caFile)
	}

	return credentials.NewTLS(&tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
	}), nil
}

func LoadConfig() (*Config, error) {
	address := util.GetEndpointAddress(EnvCfgServerAddr, "cfgserver", "50051")

	transportCreds, err := LoadTransportCredentials()
	if err != nil {
		return nil, fmt.Errorf("load cfgserver transport credentials: %w", err)
	}

	conn, err := grpc.NewClient(address, grpc.WithTransportCredentials(transportCreds))
	if err != nil {
		return nil, fmt.Errorf("could not connect to cfgserver: %v", err)
	}
	defer conn.Close()

	client := cfgservepb.NewConfigServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := client.GetSystemConfig(ctx, &cfgservepb.GetConfigRequest{SystemName: "canserver"})
	if err != nil {
		return nil, fmt.Errorf("failed to get config from cfgserver: %v", err)
	}

	c := &Config{}
	err = json.Unmarshal([]byte(resp.ConfigJson), &c.SecretConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret config: %v", err)
	}

	return c, nil
}
