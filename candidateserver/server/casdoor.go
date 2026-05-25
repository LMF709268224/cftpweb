package server

import (
	"candidateserver/config"
	"github.com/afnandelfin620-star/cftptest/cftp/util"
	"log/slog"
	"strings"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// CasdoorClient 封装 Casdoor SDK 的 IAM 验权操作。
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

// NewCasdoorClient 初始化 Casdoor SDK 全局配置，仅需进程启动时执行一次。
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
