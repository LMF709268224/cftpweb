package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// ----------------- Credential Definitions -----------------

// ListCredentialDefinitions 获取资格定义列表
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

// CreateCredentialDefinition 创建资格定义
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

// ----------------- Applications (审核中心) -----------------

type ListApplicationsReq struct {
	PageNumber int32  `json:"page_number"`
	PageSize   int32  `json:"page_size"`
	Status     string `json:"status"` // PENDING, APPROVED, REJECTED, RESUBMIT
}

// ListApplications 查询考生资格申请
func (h *Handler) ListApplications(w http.ResponseWriter, r *http.Request) {
	// 默认参数
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
		// TODO: 待 gcreds ListApplicationsRequest 补充 Status 字段后改为下推筛选。
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

// GetApplication 查询考生资格申请详情
func (h *Handler) GetApplication(w http.ResponseWriter, r *http.Request) {
	appID, ok := requiredURLParam(w, r, "app_id")
	if !ok {
		return
	}

	res, err := h.Creds.GetApplicationDetail(r.Context(), &gcredspb.GetApplicationDetailRequest{
		AppUlid: appID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, applicationDetailPayload(res))
}

func applicationDetailPayload(app *gcredspb.Application) map[string]interface{} {
	if app == nil {
		return map[string]interface{}{}
	}

	files := make([]map[string]interface{}, 0, len(app.GetFiles()))
	for _, file := range app.GetFiles() {
		files = append(files, credentialFilePayload(file))
	}

	return map[string]interface{}{
		"app_ulid":       app.GetAppUlid(),
		"app_id":         app.GetAppUlid(),
		"candidate_ulid": app.GetCandidateUlid(),
		"cred_def_ulid":  app.GetCredDefUlid(),
		"cred_def_id":    app.GetCredDefUlid(),
		"status":         app.GetStatus(),
		"files":          files,
		"auditor_ulid":   app.GetAuditorUlid(),
		"audit_remark":   app.GetAuditRemark(),
		"audit_at":       app.GetAuditAt(),
		"created_at":     app.GetCreatedAt(),
		"update_count":   app.GetUpdateCount(),
	}
}

func credentialFilePayload(file *gcredspb.FileInfo) map[string]interface{} {
	if file == nil {
		return map[string]interface{}{}
	}

	return map[string]interface{}{
		"file_hash":  file.GetFileHash(),
		"file_name":  file.GetFileName(),
		"file_type":  file.GetFileType(),
		"file_ext":   file.GetFileExt(),
		"file_size":  file.GetFileSize(),
		"file_usage": file.GetFileUsage(),
		"view_url":   file.GetViewUrl(),
	}
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
	AppId           string `json:"app_id"`
	AppUlid         string `json:"app_ulid"`
	Approved        bool   `json:"approved"`
	RejectReason    string `json:"reject_reason"`
	RequireResubmit bool   `json:"require_resubmit"`
}

// AuditApplication 审核申请
func (h *Handler) AuditApplication(w http.ResponseWriter, r *http.Request) {
	candidateID := AdminID(r)

	var body AuditApplicationReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	applicationID := strings.TrimSpace(body.ApplicationId)
	if applicationID == "" {
		applicationID = strings.TrimSpace(body.AppId)
	}
	if applicationID == "" {
		applicationID = strings.TrimSpace(body.AppUlid)
	}
	if !requireRequestField(w, applicationID, "app_id") {
		return
	}

	req := &gcredspb.AuditApplicationRequest{
		AppUlid:       applicationID,
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
