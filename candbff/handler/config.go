package handler

import (
	"net/http"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

type OrganizationConfigRsp struct {
	CountryCodes []string `json:"country_codes"`
}

// GetOrganizationConfig GET /api/public/config/organization
func (h *Handler) GetOrganizationConfig(w http.ResponseWriter, r *http.Request) {
	org, err := casdoorsdk.GetOrganization(h.CasdoorOrgName)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get organization config")
		return
	}

	WriteJSON(w, http.StatusOK, OrganizationConfigRsp{
		CountryCodes: org.CountryCodes,
	})
}
