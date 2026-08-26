package handler

import (
	"context"
	"fmt"
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

// GetCredentialDefinitionDetail 获取资格定义详情
func (h *Handler) GetCredentialDefinitionDetail(w http.ResponseWriter, r *http.Request) {
	credDefULID, ok := requiredURLParam(w, r, "cred_def_ulid")
	if !ok {
		return
	}

	res, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), &gcredspb.GetCredentialDefinitionDetailRequest{
		CredDefUlid: credDefULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type CreateCredentialDefinitionReq struct {
	CredDefUlid       string `json:"cred_def_ulid"`
	CredDefId         string `json:"cred_def_id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	Category          string `json:"category"`
	Respath           string `json:"respath"`
	AcquisitionMethod string `json:"acquisition_method"`
	FileConstraints   []struct {
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

	credDefULID := strings.TrimSpace(firstNonEmpty(body.CredDefUlid, body.CredDefId))
	if credDefULID == "" {
		credDefULID = newLmsID()
	}

	req := &gcredspb.CreateCredentialDefinitionRequest{
		CredDefUlid:       credDefULID,
		Name:              body.Name,
		Description:       body.Description,
		Category:          body.Category,
		Respath:           body.Respath,
		AcquisitionMethod: body.AcquisitionMethod,
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

// RequestCredentialDefinitionAttachmentUploadURL requests a direct-upload URL for a definition attachment.
func (h *Handler) RequestCredentialDefinitionAttachmentUploadURL(w http.ResponseWriter, r *http.Request) {
	credDefULID, ok := requiredURLParam(w, r, "cred_def_ulid")
	if !ok {
		return
	}

	var body struct {
		FileName    string `json:"file_name"`
		FileExt     string `json:"file_ext"`
		ContentType string `json:"content_type"`
		FileHash    string `json:"file_hash"`
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body")
		return
	}
	body.FileName = strings.TrimSpace(body.FileName)
	body.FileExt = strings.TrimPrefix(strings.TrimSpace(body.FileExt), ".")
	body.ContentType = strings.TrimSpace(body.ContentType)
	body.FileHash = strings.TrimSpace(body.FileHash)
	if !requireRequestFields(
		w,
		body.FileName, "file_name",
		body.FileExt, "file_ext",
		body.ContentType, "content_type",
		body.FileHash, "file_hash",
	) {
		return
	}

	res, err := h.Creds.AdminRequestDefinitionAttachmentUploadUrl(r.Context(), &gcredspb.AdminRequestDefinitionAttachmentUploadUrlRequest{
		CredDefUlid: credDefULID,
		FileName:    body.FileName,
		FileExt:     body.FileExt,
		ContentType: body.ContentType,
		FileHash:    body.FileHash,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

// UpdateCredentialDefinitionAttachments replaces the complete attachment list for a definition.
func (h *Handler) UpdateCredentialDefinitionAttachments(w http.ResponseWriter, r *http.Request) {
	credDefULID, ok := requiredURLParam(w, r, "cred_def_ulid")
	if !ok {
		return
	}

	var body struct {
		Attachments []*gcredspb.CredentialAttachment `json:"attachments"`
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body")
		return
	}
	for index, attachment := range body.Attachments {
		if attachment == nil {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("attachments[%d] is required", index))
			return
		}
		attachment.Name = strings.TrimSpace(attachment.GetName())
		attachment.Description = strings.TrimSpace(attachment.GetDescription())
		attachment.FileName = strings.TrimSpace(attachment.GetFileName())
		attachment.FileExt = strings.TrimPrefix(strings.TrimSpace(attachment.GetFileExt()), ".")
		attachment.FileHash = strings.TrimSpace(attachment.GetFileHash())
		attachment.FileKey = strings.TrimSpace(attachment.GetFileKey())
		if !requireRequestFields(
			w,
			attachment.Name, fmt.Sprintf("attachments[%d].name", index),
			attachment.FileName, fmt.Sprintf("attachments[%d].file_name", index),
			attachment.FileExt, fmt.Sprintf("attachments[%d].file_ext", index),
			attachment.FileHash, fmt.Sprintf("attachments[%d].file_hash", index),
			attachment.FileKey, fmt.Sprintf("attachments[%d].file_key", index),
		) {
			return
		}
		if attachment.GetFileType() == gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_UNSPECIFIED || attachment.GetFileSize() == 0 {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, fmt.Sprintf("attachments[%d] file_type and file_size are required", index))
			return
		}
	}

	res, err := h.Creds.AdminUpdateCredentialDefinitionAttachments(r.Context(), &gcredspb.AdminUpdateCredentialDefinitionAttachmentsRequest{
		CredDefUlid: credDefULID,
		Attachments: body.Attachments,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

// ListCredentials GET /api/credentials
func (h *Handler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	filters := &gcredspb.CredentialFilters{
		CandidateUlid: strings.TrimSpace(r.URL.Query().Get("candidate_ulid")),
		CredDefUlid:   strings.TrimSpace(r.URL.Query().Get("cred_def_ulid")),
		Status:        strings.TrimSpace(r.URL.Query().Get("status")),
	}
	res, err := h.Creds.ListCredentials(r.Context(), &gcredspb.ListCredentialsRequest{
		Filters:   filters,
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gcredspb.SortOrder(page.Sort),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Creds.GetCredentialCount(ctx, &gcredspb.GetCredentialCountRequest{
			Filters: filters,
			Limit:   limit,
			Cursor:  cursor,
		})
		if err != nil {
			return 0, "", err
		}
		return resp.GetCount(), resp.GetNextCursor(), nil
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	credentials := make([]map[string]interface{}, 0, len(res.GetCredentials()))
	for _, credential := range res.GetCredentials() {
		if credential == nil {
			continue
		}
		item := jsonPayloadObject(credential)
		h.attachCandidateName(item, credential.GetCandidateUlid())
		credentials = append(credentials, item)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"credentials": credentials,
		"total":       total.Total,
		"total_label": total.Label(),
		"total_exact": total.Exact,
		"next_cursor": res.GetNextCursor(),
		"has_more":    res.GetHasMore(),
	})
}

// IgnoreVersionFile POST /api/credentials/version-files/{file_id}/ignore
func (h *Handler) IgnoreVersionFile(w http.ResponseWriter, r *http.Request) {
	fileIDRaw, ok := requiredURLParam(w, r, "file_id")
	if !ok {
		return
	}
	fileID, err := strconv.ParseInt(fileIDRaw, 10, 64)
	if err != nil || fileID <= 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "file_id must be a positive integer")
		return
	}

	var body struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := ReadJSON(r, &body); err != nil {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
			return
		}
	}

	res, err := h.Creds.IgnoreVersionFile(r.Context(), &gcredspb.IgnoreVersionFileRequest{
		FileId:       fileID,
		OperatorUlid: adminActorID(r),
		Reason:       strings.TrimSpace(body.Reason),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// CheckCredentialResourcesExist POST /api/credentials/resources/check
func (h *Handler) CheckCredentialResourcesExist(w http.ResponseWriter, r *http.Request) {
	var req gcredspb.CheckResourcesExistRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if len(req.CredDefIds) == 0 && len(req.PdfTemplateIds) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "cred_def_ids or pdf_template_ids is required")
		return
	}

	res, err := h.Creds.CheckResourcesExist(r.Context(), &req)
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
	page := parseCursorPage(r, 20)
	statusFilter := strings.TrimSpace(r.URL.Query().Get("status"))

	req := &gcredspb.ListApplicationsRequest{
		Filters:   &gcredspb.ApplicationFilters{},
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gcredspb.SortOrder(page.Sort),
	}
	if statusFilter != "" && statusFilter != "0" && strings.ToUpper(statusFilter) != "ALL" {
		req.Filters.Statuses = []string{statusFilter}
	}

	res, err := h.Creds.ListApplications(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Creds.GetApplicationCount(ctx, &gcredspb.GetApplicationCountRequest{
			Filters: req.GetFilters(),
			Limit:   limit,
			Cursor:  cursor,
		})
		if err != nil {
			return 0, "", err
		}
		return resp.GetCount(), resp.GetNextCursor(), nil
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	credentialNames := h.credentialDefinitionNames(r)
	applications := make([]map[string]interface{}, 0, len(res.GetApplications()))
	for _, app := range res.GetApplications() {
		if app == nil {
			continue
		}
		item := jsonPayloadObject(app)
		if name := credentialNames[app.GetCredDefUlid()]; name != "" {
			item["cred_def_name"] = name
			item["credential_name"] = name
		}
		h.attachCandidateName(item, app.GetCandidateUlid())
		applications = append(applications, item)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"applications": applications,
		"total":        total.Total,
		"total_label":  total.Label(),
		"total_exact":  total.Exact,
		"next_cursor":  res.GetNextCursor(),
		"has_more":     res.GetHasMore(),
	})
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

	WriteJSON(w, http.StatusOK, h.applicationDetailPayload(r, res))
}

func (h *Handler) applicationDetailPayload(r *http.Request, app *gcredspb.Application) map[string]interface{} {
	if app == nil {
		return map[string]interface{}{}
	}

	files := make([]map[string]interface{}, 0, len(app.GetFiles()))
	for _, file := range app.GetFiles() {
		files = append(files, credentialFilePayload(file))
	}

	payload := map[string]interface{}{
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
	if name := h.credentialDefinitionNameByID(r, app.GetCredDefUlid()); name != "" {
		payload["cred_def_name"] = name
		payload["credential_name"] = name
	}
	h.attachCandidateName(payload, app.GetCandidateUlid())
	return payload
}

func (h *Handler) credentialDefinitionNames(r *http.Request) map[string]string {
	res, err := h.Creds.ListCredentialDefinitions(r.Context(), &gcredspb.ListCredentialDefinitionsRequest{})
	if err != nil {
		return map[string]string{}
	}
	names := make(map[string]string, len(res.GetDefinitions()))
	for _, def := range res.GetDefinitions() {
		if def == nil {
			continue
		}
		id := strings.TrimSpace(def.GetCredDefUlid())
		name := strings.TrimSpace(def.GetName())
		if id != "" && name != "" {
			names[id] = name
		}
	}
	return names
}

func (h *Handler) credentialDefinitionNameByID(r *http.Request, credDefULID string) string {
	credDefULID = strings.TrimSpace(credDefULID)
	if credDefULID == "" {
		return ""
	}
	res, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), &gcredspb.GetCredentialDefinitionDetailRequest{
		CredDefUlid: credDefULID,
	})
	if err != nil || res == nil {
		return ""
	}
	return strings.TrimSpace(res.GetName())
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

type AuditApplicationReq struct {
	ApplicationId   string `json:"application_id"`
	AppId           string `json:"app_id"`
	AppUlid         string `json:"app_ulid"`
	Approved        bool   `json:"approved"`
	RejectReason    string `json:"reject_reason"`
	RequireResubmit bool   `json:"require_resubmit"`
	ValidUntil      string `json:"valid_until"`
}

// AuditApplication 审核申请
func (h *Handler) AuditApplication(w http.ResponseWriter, r *http.Request) {
	candidateID := AdminID(r)
	if !requireRequestField(w, candidateID, "admin_ulid") {
		return
	}

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
	validUntilRaw := strings.TrimSpace(body.ValidUntil)
	validUntil, err := time.Parse(time.RFC3339, validUntilRaw)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "valid_until must be an RFC3339 timestamp")
		return
	}
	if body.Approved && !validUntil.After(time.Now()) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "valid_until must be in the future when approving an application")
		return
	}

	req := &gcredspb.AuditApplicationRequest{
		AppUlid:       applicationID,
		Approved:      body.Approved,
		AuditRemark:   body.RejectReason,
		AllowReupload: body.RequireResubmit,
		AuditorUlid:   candidateID,
		ValidUntil:    validUntil.UTC().Format(time.RFC3339),
	}

	res, err := h.Creds.AuditApplication(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}
