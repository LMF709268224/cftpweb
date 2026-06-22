package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// ----------------- Credential Definitions -----------------

// ListCredentialDefinitions 鑾峰彇璧勬牸瀹氫箟鍒楄〃
func (h *Handler) ListCredentialDefinitions(w http.ResponseWriter, r *http.Request) {
	req := &gcredspb.ListCredentialDefinitionsRequest{}

	res, err := h.Creds.ListCredentialDefinitions(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type CreateCredentialDefinitionReq struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Category        string `json:"category"`
	FileConstraints []struct {
		Name       string `json:"name"`
		Type       int32  `json:"type"`
		IsRequired bool   `json:"is_required"`
	} `json:"file_constraints"`
}

// CreateCredentialDefinition 鍒涘缓璧勬牸瀹氫箟
func (h *Handler) CreateCredentialDefinition(w http.ResponseWriter, r *http.Request) {
	var body CreateCredentialDefinitionReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	req := &gcredspb.CreateCredentialDefinitionRequest{
		Name:        body.Name,
		Description: body.Description,
		Category:    body.Category,
	}

	for _, fc := range body.FileConstraints {
		req.FileConstraints = append(req.FileConstraints, &gcredspb.CredentialFileConstraint{
			Name:       fc.Name,
			Type:       gcredspb.CredentialFileType(fc.Type),
			IsRequired: fc.IsRequired,
		})
	}

	res, err := h.Creds.CreateCredentialDefinition(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// ----------------- Applications (瀹℃牳涓績) -----------------

type ListApplicationsReq struct {
	PageNumber int32  `json:"page_number"`
	PageSize   int32  `json:"page_size"`
	Status     string `json:"status"` // PENDING, APPROVED, REJECTED, RESUBMIT
}

// ListApplications 鏌ヨ鑰冪敓璧勬牸鐢宠
func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	// 榛樿鍙傛暟
	page := uint32(1)
	pageSize := uint32(20)

	// query params
	qPageNumber := r.URL.Query().Get("page_number")
	qPageSize := r.URL.Query().Get("page_size")
	statusFilter := normalizeApplicationStatus(r.URL.Query().Get("status"))

	if qPageNumber != "" {
		if val, err := strconv.Atoi(qPageNumber); err == nil {
			page = uint32(val)
		}
	}
	if qPageSize != "" {
		if val, err := strconv.Atoi(qPageSize); err == nil {
			pageSize = uint32(val)
		}
	}

	req := &gcredspb.ListApplicationsRequest{Page: page, PageSize: pageSize}
	if statusFilter != "" {
		// TODO: 寰?gcreds ListApplicationsRequest 琛ュ厖 Status 瀛楁鍚庢敼涓轰笅鎺ㄧ瓫閫夈€?
		req.Page = 1
		req.PageSize = 500
	}

	res, err := h.Creds.ListApplications(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	if statusFilter != "" {
		filtered := make([]*gcredspb.ApplicationSummary, 0, len(res.GetApplications()))
		for _, app := range res.GetApplications() {
			if normalizeApplicationStatus(app.GetStatus()) == statusFilter {
				filtered = append(filtered, app)
			}
		}

		start := int((page - 1) * pageSize)
		end := start + int(pageSize)
		if start > len(filtered) {
			start = len(filtered)
		}
		if end > len(filtered) {
			end = len(filtered)
		}

		res.Applications = filtered[start:end]
		res.Total = uint32(len(filtered))
	}

	WriteJSON(w, http.StatusOK, res)
}

func normalizeApplicationStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "", "0", "ALL", "APPLICATION_STATUS_UNSPECIFIED":
		return ""
	case "1", "PENDING", "APPLICATION_STATUS_PENDING":
		return "PENDING"
	case "2", "APPROVED", "APPLICATION_STATUS_APPROVED":
		return "APPROVED"
	case "3", "REJECTED", "APPLICATION_STATUS_REJECTED":
		return "REJECTED"
	case "4", "RESUBMIT", "REUPLOAD", "NEEDS_RESUBMIT", "APPLICATION_STATUS_RESUBMIT", "APPLICATION_STATUS_REUPLOAD":
		return "RESUBMIT"
	default:
		return strings.ToUpper(strings.TrimSpace(status))
	}
}

type AuditApplicationReq struct {
	ApplicationId   string `json:"application_id"`
	Approved        bool   `json:"approved"`
	RejectReason    string `json:"reject_reason"`
	RequireResubmit bool   `json:"require_resubmit"`
}

// AuditApplication 瀹℃牳鐢宠
func (h *Handler) AuditApplication(w http.ResponseWriter, r *http.Request) {
	candidateID := AdminID(r)

	var body AuditApplicationReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	req := &gcredspb.AuditApplicationRequest{
		AppUlid:       body.ApplicationId,
		Approved:      body.Approved,
		AuditRemark:   body.RejectReason,
		AllowReupload: body.RequireResubmit,
		AuditorUlid:   candidateID,
		ValidUntil:    time.Now().AddDate(2, 0, 0).Format(time.RFC3339),
	}

	res, err := h.Creds.AuditApplication(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}
