package handler

import (
	"net/http"
	"os"

	"candidateserver/config"
)

type PublicConfigRsp struct {
	StripePublishableKey string `json:"stripe_publishable_key,omitempty"`
}

// GetPublicConfig returns browser-safe runtime configuration.
func (h *Handler) GetPublicConfig(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, PublicConfigRsp{
		StripePublishableKey: os.Getenv(config.EnvStripePublishableKey),
	})
}
