package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	gstatus "google.golang.org/grpc/status"
)

// ListCredentialDefinitions GET /api/credentials/definitions
func (h *Handler) ListCredentialDefinitions(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	locale := requestLocale(r)
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
			var translation *gcredspb.CredDefTranslation
			def, translation = h.localizedCredentialDefinitionWithTranslation(r.Context(), def, locale)
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
				"file_constraints":   credentialFileConstraintPayloads(def, translation),
				"category":           def.GetCategory(),
				"respath":            def.GetRespath(),
				"acquisition_method": def.GetAcquisitionMethod(),
				"attachments":        def.GetAttachments(),
				"latest_application": latestApplication,
			})
		}
		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"definitions": details,
		})
		return
	}

	applications, err := h.listActionableCredentialApplications(r.Context(), candidateID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	details := make([]map[string]interface{}, 0, len(applications))
	seenDefinitions := make(map[string]struct{}, len(applications))
	for _, application := range applications {
		credDefID := application.GetCredDefUlid()
		if credDefID == "" {
			continue
		}
		if _, seen := seenDefinitions[credDefID]; seen {
			continue
		}
		seenDefinitions[credDefID] = struct{}{}

		detailReq := &gcredspb.GetCredentialDefinitionDetailRequest{
			CredDefUlid: credDefID,
		}
		detailRes, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), detailReq)
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		var translation *gcredspb.CredDefTranslation
		detailRes, translation = h.localizedCredentialDefinitionWithTranslation(r.Context(), detailRes, locale)
		details = append(details, map[string]interface{}{
			"cred_def_ulid":      detailRes.GetCredDefUlid(),
			"cred_def_id":        detailRes.GetCredDefUlid(),
			"name":               detailRes.GetName(),
			"description":        detailRes.GetDescription(),
			"file_constraints":   credentialFileConstraintPayloads(detailRes, translation),
			"category":           detailRes.GetCategory(),
			"respath":            detailRes.GetRespath(),
			"acquisition_method": detailRes.GetAcquisitionMethod(),
			"attachments":        detailRes.GetAttachments(),
			"latest_application": credentialApplicationPayload(application),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"definitions": details,
	})
}

var actionableCredentialApplicationStatuses = []string{"PendingUpload", "Reupload"}

func (h *Handler) listActionableCredentialApplications(ctx context.Context, candidateID string) ([]*gcredspb.ApplicationSummary, error) {
	applications := make([]*gcredspb.ApplicationSummary, 0)
	cursor := ""
	for {
		res, err := h.Creds.ListCandidateApplications(ctx, &gcredspb.ListApplicationsRequest{
			Filters: &gcredspb.ApplicationFilters{
				CandidateUlid: candidateID,
				Statuses:      actionableCredentialApplicationStatuses,
			},
			PageSize:  100,
			Cursor:    cursor,
			SortOrder: gcredspb.SortOrder_SORT_ORDER_DESC,
		})
		if err != nil {
			return nil, err
		}
		applications = append(applications, res.GetApplications()...)
		if !res.GetHasMore() || res.GetNextCursor() == "" {
			return applications, nil
		}
		cursor = res.GetNextCursor()
	}
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
	if len(body.QualUlids) != 1 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field \"qual_ulids\" must contain exactly one qualification")
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

type credentialApplicationOrderItem struct {
	QualID         string `json:"qual_id"`
	ItemStatus     string `json:"item_status"`
	CredentialULID string `json:"credential_ulid"`
	QualNameHint   string `json:"qual_name_hint"`
}

// ListCredentialApplicationOrdersForCandidate GET /api/credentials/application-orders
func (h *Handler) ListCredentialApplicationOrdersForCandidate(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	summaries := make([]*mallpb.CredentialApplicationOrderSummary, 0)
	cursor := ""
	guard := newCursorScanGuard()
	for {
		list, err := h.Mall.ListCredentialApplicationOrders(r.Context(), &mallpb.ListCredentialApplicationOrdersRequest{
			Filters: &mallpb.CredentialApplicationOrderFilters{
				CandidateUlid: candidateID,
			},
			Cursor:    cursor,
			PageSize:  100,
			SortOrder: mallpb.SortOrder_SORT_ORDER_DESC,
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		summaries = append(summaries, list.GetItems()...)

		nextCursor, done, guardErr := guard.next(cursor, list.GetHasMore(), list.GetNextCursor())
		if guardErr != nil {
			HandleGrpcError(w, gstatus.Error(codes.Internal, guardErr.Error()))
			return
		}
		if done {
			break
		}
		cursor = nextCursor
	}

	orders := make([]map[string]interface{}, 0, len(summaries))
	for _, orderSummary := range summaries {
		if orderSummary == nil || strings.TrimSpace(orderSummary.GetApplicationOrderUlid()) == "" {
			continue
		}

		res, err := h.Mall.GetCredentialApplicationOrderDetail(r.Context(), &mallpb.GetCredentialApplicationOrderDetailRequest{
			ApplicationOrderUlid: orderSummary.GetApplicationOrderUlid(),
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		detail := res.GetDetail()
		if !res.GetFound() || detail == nil || detail.GetSummary() == nil || detail.GetSummary().GetCandidateUlid() != candidateID {
			continue
		}

		items := make([]credentialApplicationOrderItem, 0)
		if raw := strings.TrimSpace(detail.GetApplicationItemsJson()); raw != "" {
			if err := json.Unmarshal([]byte(raw), &items); err != nil {
				WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "Invalid credential application order items returned by GMALL")
				return
			}
		}

		summary := detail.GetSummary()
		orders = append(orders, map[string]interface{}{
			"application_order_ulid": summary.GetApplicationOrderUlid(),
			"order_status":           summary.GetOrderStatus(),
			"pay_order_ulid":         firstNonEmpty(detail.GetPayOrderUlid(), summary.GetPayOrderUlid()),
			"created_at":             summary.GetCreatedAt(),
			"status_at":              detail.GetStatusAt(),
			"updated_at":             detail.GetUpdatedAt(),
			"items":                  items,
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"orders": orders})
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
	locale := requestLocale(r)
	for _, app := range res.GetApplications() {
		if app == nil {
			continue
		}
		payload := credentialApplicationPayload(app)
		if def := h.credentialDefinitionSummary(r.Context(), app.GetCredDefUlid(), locale, definitionNameCache); def != nil {
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

// GetCandidateApplication GET /api/credentials/applications/{appId}
func (h *Handler) GetCandidateApplication(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	applicationID := strings.TrimSpace(chi.URLParam(r, "appId"))
	if applicationID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'app_id' is required")
		return
	}

	application, err := h.Creds.GetApplicationDetail(r.Context(), &gcredspb.GetApplicationDetailRequest{
		AppUlid: applicationID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if application == nil || strings.TrimSpace(application.GetCandidateUlid()) != candidateID {
		WriteError(w, http.StatusNotFound, ErrNotFound, "credential application not found or access denied")
		return
	}

	files := make([]map[string]interface{}, 0, len(application.GetFiles()))
	for _, file := range application.GetFiles() {
		files = append(files, candidateCredentialFilePayload(file))
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"app_ulid":       application.GetAppUlid(),
		"app_id":         application.GetAppUlid(),
		"candidate_ulid": application.GetCandidateUlid(),
		"cred_def_ulid":  application.GetCredDefUlid(),
		"cred_def_id":    application.GetCredDefUlid(),
		"status":         application.GetStatus(),
		"files":          files,
		"auditor_ulid":   application.GetAuditorUlid(),
		"audit_remark":   application.GetAuditRemark(),
		"audit_at":       application.GetAuditAt(),
		"created_at":     application.GetCreatedAt(),
		"update_count":   application.GetUpdateCount(),
	})
}

func candidateCredentialFilePayload(file *gcredspb.FileInfo) map[string]interface{} {
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

func (h *Handler) credentialDefinitionSummary(ctx context.Context, credDefULID string, locale string, cache map[string]map[string]interface{}) map[string]interface{} {
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
	def = h.localizedCredentialDefinition(ctx, def, locale)
	summary := map[string]interface{}{
		"credential_name":        def.GetName(),
		"credential_description": def.GetDescription(),
		"credential_category":    def.GetCategory(),
		"acquisition_method":     def.GetAcquisitionMethod(),
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

	countRes, err := h.Creds.GetApplicationCount(ctx, &gcredspb.GetApplicationCountRequest{
		Filters: &gcredspb.ApplicationFilters{
			CandidateUlid: candidateID,
			Statuses:      actionableCredentialApplicationStatuses,
		},
		Limit: 1000,
	})
	if err != nil {
		HandleGrpcErrorWithContext(w, ctx, err)
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{"actionable_count": countRes.GetCount()})
}
