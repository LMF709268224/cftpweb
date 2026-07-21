package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
)

// ListCredentialDefinitions GET /api/credentials/definitions
func (h *Handler) ListCredentialDefinitions(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	qualIDs := compactStrings(strings.Split(firstNonEmpty(r.URL.Query().Get("qual_ulids"), r.URL.Query().Get("qual_ids")), ","))
	if len(qualIDs) > 0 {
		details := make([]map[string]interface{}, 0, len(qualIDs))
		for _, qualID := range qualIDs {
			def, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), &gcredspb.GetCredentialDefinitionDetailRequest{
				CredDefUlid: qualID,
			})
			if err != nil {
				HandleGrpcError(w, err)
				return
			}
			latestApplication, err := h.latestCredentialApplication(r.Context(), candidateID, qualID)
			if err != nil {
				HandleGrpcError(w, err)
				return
			}
			details = append(details, map[string]interface{}{
				"cred_def_ulid":      def.GetCredDefUlid(),
				"cred_def_id":        def.GetCredDefUlid(),
				"name":               def.GetName(),
				"description":        def.GetDescription(),
				"file_constraints":   def.GetFileConstraints(),
				"category":           def.GetCategory(),
				"respath":            def.GetRespath(),
				"latest_application": latestApplication,
			})
		}
		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"definitions": details,
		})
		return
	}

	req := &gcredspb.ListCandidateEligibleDefinitionsRequest{
		CandidateUlid: candidateID,
	}

	res, err := h.Creds.ListCandidateEligibleDefinitions(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	details := make([]map[string]interface{}, 0, len(res.GetDefinitions()))
	for _, def := range res.GetDefinitions() {
		latestApplication, err := h.latestCredentialApplication(r.Context(), candidateID, def.GetCredDefUlid())
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		detailReq := &gcredspb.GetCredentialDefinitionDetailRequest{
			CredDefUlid: def.GetCredDefUlid(),
		}
		detailRes, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), detailReq)
		if err == nil && detailRes != nil {
			details = append(details, map[string]interface{}{
				"cred_def_ulid":      detailRes.GetCredDefUlid(),
				"cred_def_id":        detailRes.GetCredDefUlid(),
				"name":               detailRes.GetName(),
				"description":        detailRes.GetDescription(),
				"file_constraints":   detailRes.GetFileConstraints(),
				"category":           detailRes.GetCategory(),
				"respath":            detailRes.GetRespath(),
				"latest_application": latestApplication,
			})
			continue
		}
		details = append(details, map[string]interface{}{
			"cred_def_ulid":      def.GetCredDefUlid(),
			"cred_def_id":        def.GetCredDefUlid(),
			"name":               def.GetName(),
			"description":        def.GetDescription(),
			"category":           def.GetCategory(),
			"latest_application": latestApplication,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"definitions": details,
	})
}

// CheckCandidateQualifications GET /api/credentials/qualifications
func (h *Handler) CheckCandidateQualifications(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	qualIDs := compactStrings(strings.Split(firstNonEmpty(r.URL.Query().Get("qual_ulids"), r.URL.Query().Get("qual_ids")), ","))
	if len(qualIDs) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field \"qual_ulids\" is required but was empty")
		return
	}

	items := make([]map[string]interface{}, 0, len(qualIDs))
	for _, qualID := range qualIDs {
		check, err := h.Creds.CheckCandidateQualification(r.Context(), &gcredspb.CheckCandidateQualificationRequest{
			CandidateUlid: candidateID,
			CredDefUlid:   qualID,
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		items = append(items, map[string]interface{}{
			"qual_id":           qualID,
			"cred_def_ulid":     qualID,
			"eligible":          check.GetEligible(),
			"credential_status": check.GetCredentialStatus().String(),
			"message":           check.GetMessage(),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"qualifications": items,
	})
}

// CreateCredentialApplicationOrder POST /api/credentials/application-orders
func (h *Handler) CreateCredentialApplicationOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var body CreateCredentialApplicationOrderReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	body.PipelineCcUlid = strings.TrimSpace(body.PipelineCcUlid)
	body.BundleUlid = strings.TrimSpace(body.BundleUlid)
	body.QualUlids = compactStrings(body.QualUlids)
	if len(body.QualUlids) == 0 {
		body.QualUlids = compactStrings(body.LegacyQualIDs)
	}
	if !requireRequestFields(w, body.PipelineCcUlid, "pipeline_cc_ulid", body.BundleUlid, "bundle_ulid") {
		return
	}
	if len(body.QualUlids) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field \"qual_ulids\" is required but was empty")
		return
	}

	res, err := h.Mall.CreateCredentialApplicationOrder(r.Context(), &mallpb.CreateCredentialApplicationOrderRequest{
		CandidateUlid:  candidateID,
		PipelineCcUlid: body.PipelineCcUlid,
		BundleUlid:     body.BundleUlid,
		QualUlids:      body.QualUlids,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, credentialApplicationOrderPayload(res))
}

// ListCandidateApplications GET /api/credentials/applications
func (h *Handler) ListCandidateApplications(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	credDefID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("cred_def_ulid"), r.URL.Query().Get("cred_def_id")))

	page := parseCursorPage(r, 10)

	req := &gcredspb.ListApplicationsRequest{
		Filters: &gcredspb.ApplicationFilters{
			CandidateUlid: candidateID,
			CredDefUlid:   credDefID,
		},
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gcredspb.SortOrder(page.Sort),
	}

	res, err := h.Creds.ListCandidateApplications(r.Context(), req)
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

	applications := make([]map[string]interface{}, 0, len(res.GetApplications()))
	definitionNameCache := map[string]map[string]interface{}{}
	for _, app := range res.GetApplications() {
		if app == nil {
			continue
		}
		payload := credentialApplicationPayload(app)
		if def := h.credentialDefinitionSummary(r.Context(), app.GetCredDefUlid(), definitionNameCache); def != nil {
			for key, value := range def {
				payload[key] = value
			}
		}
		applications = append(applications, payload)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"applications": applications,
		"total":        total.Total,
		"total_label":  total.Label(),
		"total_exact":  total.Exact,
		"page_size":    page.PageSize,
		"next_cursor":  res.GetNextCursor(),
		"prev_cursor":  res.GetPrevCursor(),
		"has_more":     res.GetHasMore(),
	})
}

func (h *Handler) latestCredentialApplication(ctx context.Context, candidateID, credDefID string) (map[string]interface{}, error) {
	res, err := h.Creds.ListCandidateApplications(ctx, &gcredspb.ListApplicationsRequest{
		Filters: &gcredspb.ApplicationFilters{
			CandidateUlid: candidateID,
			CredDefUlid:   credDefID,
		},
		PageSize:  1,
		SortOrder: gcredspb.SortOrder_SORT_ORDER_DESC,
	})
	if err != nil {
		return nil, err
	}
	applications := res.GetApplications()
	if len(applications) == 0 || applications[0] == nil {
		return nil, nil
	}
	return credentialApplicationPayload(applications[0]), nil
}

func (h *Handler) credentialDefinitionSummary(ctx context.Context, credDefULID string, cache map[string]map[string]interface{}) map[string]interface{} {
	credDefULID = strings.TrimSpace(credDefULID)
	if credDefULID == "" {
		return nil
	}
	if cached, ok := cache[credDefULID]; ok {
		return cached
	}
	def, err := h.Creds.GetCredentialDefinitionDetail(ctx, &gcredspb.GetCredentialDefinitionDetailRequest{
		CredDefUlid: credDefULID,
	})
	if err != nil || def == nil {
		cache[credDefULID] = nil
		return nil
	}
	summary := map[string]interface{}{
		"credential_name":        def.GetName(),
		"credential_description": def.GetDescription(),
		"credential_category":    def.GetCategory(),
	}
	cache[credDefULID] = summary
	return summary
}

func credentialApplicationOrderPayload(res *mallpb.CreateCredentialApplicationOrderResponse) map[string]interface{} {
	if res == nil {
		return map[string]interface{}{}
	}
	return map[string]interface{}{
		"application_order_ulid": res.GetApplicationOrderUlid(),
		"order_status":           res.GetOrderStatus(),
		"pay_order_ulid":         res.GetPayOrderUlid(),
		"payment_key":            formatPaymentKey(res.GetPaymentKey()),
		"reused_existing":        res.GetReusedExisting(),
		"message":                res.GetMessage(),
	}
}

func credentialApplicationPayload(app *gcredspb.ApplicationSummary) map[string]interface{} {
	return map[string]interface{}{
		"app_ulid":       app.GetAppUlid(),
		"app_id":         app.GetAppUlid(),
		"candidate_ulid": app.GetCandidateUlid(),
		"cred_def_ulid":  app.GetCredDefUlid(),
		"cred_def_id":    app.GetCredDefUlid(),
		"status":         app.GetStatus(),
		"auditor_ulid":   app.GetAuditorUlid(),
		"audit_remark":   app.GetAuditRemark(),
		"audit_at":       app.GetAuditAt(),
		"created_at":     app.GetCreatedAt(),
		"update_count":   app.GetUpdateCount(),
	}
}

// CheckUploadPermission GET /api/credentials/upload-permission
func (h *Handler) CheckUploadPermission(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	credDefID := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("cred_def_ulid"), r.URL.Query().Get("cred_def_id")))
	if !requireRequestFields(w, candidateID, "candidate_ulid", credDefID, "cred_def_ulid") {
		return
	}

	res, err := h.Creds.CheckUploadPermission(r.Context(), &gcredspb.CheckUploadPermissionRequest{
		CandidateUlid: candidateID,
		CredDefUlid:   credDefID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type RequestUploadUrlReq struct {
	CredDefUlid     string `json:"cred_def_ulid"`
	LegacyCredDefID string `json:"cred_def_id,omitempty"`
	FileHash        string `json:"file_hash"`
	FileExt         string `json:"file_ext"`
	ContentType     string `json:"content_type"`
	FileUsage       string `json:"file_usage"`
}

// RequestUploadUrl POST /api/credentials/upload-url
func (h *Handler) RequestUploadUrl(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var body RequestUploadUrlReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	body.CredDefUlid = strings.TrimSpace(firstNonEmpty(body.CredDefUlid, body.LegacyCredDefID))
	if !requireRequestFields(
		w,
		body.CredDefUlid, "cred_def_ulid",
		body.FileHash, "file_hash",
		body.FileExt, "file_ext",
		body.ContentType, "content_type",
		body.FileUsage, "file_usage",
	) {
		return
	}

	req := &gcredspb.RequestUploadUrlRequest{
		CandidateUlid: candidateID,
		CredDefUlid:   body.CredDefUlid,
		FileHash:      body.FileHash,
		FileExt:       body.FileExt,
		ContentType:   body.ContentType,
		FileUsage:     body.FileUsage,
	}

	res, err := h.Creds.RequestUploadUrl(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type SubmitApplicationReq struct {
	CredDefUlid     string `json:"cred_def_ulid"`
	LegacyCredDefID string `json:"cred_def_id,omitempty"`
	Files           []struct {
		FileHash  string `json:"file_hash"`
		FileName  string `json:"file_name"`
		FileType  int32  `json:"file_type"`
		FileExt   string `json:"file_ext"`
		FileSize  uint64 `json:"file_size"`
		FileUsage string `json:"file_usage"`
	} `json:"files"`
}

// SubmitApplication POST /api/credentials/apply
func (h *Handler) SubmitApplication(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var body SubmitApplicationReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	body.CredDefUlid = strings.TrimSpace(firstNonEmpty(body.CredDefUlid, body.LegacyCredDefID))
	if !requireRequestField(w, body.CredDefUlid, "cred_def_ulid") {
		return
	}

	pbFiles := make([]*gcredspb.FileInfo, 0, len(body.Files))
	for i, f := range body.Files {
		if !requireRequestFields(
			w,
			f.FileHash, fmt.Sprintf("files[%d].file_hash", i),
			f.FileName, fmt.Sprintf("files[%d].file_name", i),
			f.FileExt, fmt.Sprintf("files[%d].file_ext", i),
			f.FileUsage, fmt.Sprintf("files[%d].file_usage", i),
		) {
			return
		}
		pbFiles = append(pbFiles, &gcredspb.FileInfo{
			FileHash:  f.FileHash,
			FileName:  f.FileName,
			FileType:  gcredspb.CredentialFileType(f.FileType),
			FileExt:   f.FileExt,
			FileSize:  f.FileSize,
			FileUsage: f.FileUsage,
		})
	}

	req := &gcredspb.SubmitApplicationRequest{
		CandidateUlid: candidateID,
		CredDefUlid:   body.CredDefUlid,
		Files:         pbFiles,
	}

	res, err := h.Creds.SubmitApplication(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type UpdateApplicationReq struct {
	AppUlid     string `json:"app_ulid"`
	LegacyAppID string `json:"app_id,omitempty"`
	Files       []struct {
		FileHash  string `json:"file_hash"`
		FileName  string `json:"file_name"`
		FileType  int32  `json:"file_type"`
		FileExt   string `json:"file_ext"`
		FileSize  uint64 `json:"file_size"`
		FileUsage string `json:"file_usage"`
	} `json:"files"`
}

// UpdateApplication PUT /api/credentials/apply
func (h *Handler) UpdateApplication(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var body UpdateApplicationReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	body.AppUlid = strings.TrimSpace(firstNonEmpty(body.AppUlid, body.LegacyAppID))
	if !requireRequestField(w, body.AppUlid, "app_ulid") {
		return
	}

	pbFiles := make([]*gcredspb.FileInfo, 0, len(body.Files))
	for i, f := range body.Files {
		if !requireRequestFields(
			w,
			f.FileHash, fmt.Sprintf("files[%d].file_hash", i),
			f.FileName, fmt.Sprintf("files[%d].file_name", i),
			f.FileExt, fmt.Sprintf("files[%d].file_ext", i),
			f.FileUsage, fmt.Sprintf("files[%d].file_usage", i),
		) {
			return
		}
		pbFiles = append(pbFiles, &gcredspb.FileInfo{
			FileHash:  f.FileHash,
			FileName:  f.FileName,
			FileType:  gcredspb.CredentialFileType(f.FileType),
			FileExt:   f.FileExt,
			FileSize:  f.FileSize,
			FileUsage: f.FileUsage,
		})
	}

	req := &gcredspb.UpdateApplicationRequest{
		AppUlid:       body.AppUlid,
		CandidateUlid: candidateID,
		Files:         pbFiles,
	}

	res, err := h.Creds.UpdateApplication(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// GetActionableCredentialCount GET /api/credentials/actionable-count
func (h *Handler) GetActionableCredentialCount(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	ctx := r.Context()

	defsRes, err := h.Creds.ListCandidateEligibleDefinitions(ctx, &gcredspb.ListCandidateEligibleDefinitionsRequest{
		CandidateUlid: candidateID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	defs := defsRes.GetDefinitions()
	if len(defs) == 0 {
		WriteJSON(w, http.StatusOK, map[string]interface{}{"actionable_count": 0})
		return
	}

	actionableCount := 0
	for _, def := range defs {
		credDefUlid := def.GetCredDefUlid()

		// 1. Check if they have ANY application for this def
		countRes, err := h.Creds.GetApplicationCount(ctx, &gcredspb.GetApplicationCountRequest{
			Filters: &gcredspb.ApplicationFilters{
				CandidateUlid: candidateID,
				CredDefUlid:   credDefUlid,
			},
			Limit: 1,
		})
		if err == nil && countRes.GetCount() == 0 {
			// No application at all -> actionable
			actionableCount++
			continue
		}

		// 2. If they have an application, check if there are any that need resubmit/reupload
		reuploadCountRes, err := h.Creds.GetApplicationCount(ctx, &gcredspb.GetApplicationCountRequest{
			Filters: &gcredspb.ApplicationFilters{
				CandidateUlid: candidateID,
				CredDefUlid:   credDefUlid,
				Statuses:      []string{"REUPLOAD", "RESUBMIT", "NEEDS_RESUBMIT", "APPLICATION_STATUS_REUPLOAD", "APPLICATION_STATUS_RESUBMIT"},
			},
			Limit: 1,
		})
		if err == nil && reuploadCountRes.GetCount() > 0 {
			actionableCount++
		}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"actionable_count": actionableCount})
}
