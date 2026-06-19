package server

import (
	"adminbff/config"
	"github.com/afnandelfin620-star/cftptest/cftp/util"
	"log/slog"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// CasdoorClient 灏佽 Casdoor SDK 鐨?IAM 楠屾潈鎿嶄綔銆?
type CasdoorClient struct {
	cfg config.CasdoorConfig
}

func getCasdoorEndpoint() string {
	addr := util.GetEndpointAddress(config.EnvCasdoorEndpoint, "casdoor", "8000")
	if !strings.HasPrefix(addr, "http://") && !strings.HasPrefix(addr, "https://") {
		return "http://" + addr
	}
	return addr
}

// NewCasdoorClient 鍒濆鍖?Casdoor SDK 鍏ㄥ眬閰嶇疆锛屼粎闇€杩涚▼鍚姩鏃舵墽琛屼竴娆°€?
func NewCasdoorClient(cfg config.CasdoorConfig) *CasdoorClient {
	endpoint := getCasdoorEndpoint()
	casdoorsdk.InitConfig(
		endpoint,
		cfg.ClientID,
		cfg.ClientSecret,
		cfg.Certificate,
		cfg.OrgName,
		cfg.AppName,
	)

	slog.Info("Casdoor client initialized",
		"endpoint", endpoint,
		"org", cfg.OrgName,
		"app", cfg.AppName,
	)

	return &CasdoorClient{cfg: cfg}
}
