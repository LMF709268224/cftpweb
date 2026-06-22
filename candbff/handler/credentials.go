package handler

import (
	"fmt"
	"net/http"
	"strings"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
)

// ListCredentialDefinitions GET /api/credentials/definitions
func (h *Handler) ListCredentialDefinitions(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	qualIDs := compactStrings(strings.Split(r.URL.Query().Get("qual_ids"), ","))
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
			details = append(details, map[string]interface{}{
				"cred_def_id":      def.GetCredDefUlid(),
				"name":             def.GetName(),
				"description":      def.GetDescription(),
				"file_constraints": def.GetFileConstraints(),
				"category":         def.GetCategory(),
				"respath":          def.GetRespath(),
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
		detailReq := &gcredspb.GetCredentialDefinitionDetailRequest{
			CredDefUlid: def.GetCredDefUlid(),
		}
		detailRes, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), detailReq)
		if err == nil && detailRes != nil {
			details = append(details, map[string]interface{}{
				"cred_def_id":      detailRes.GetCredDefUlid(),
				"name":             detailRes.GetName(),
				"description":      detailRes.GetDescription(),
				"file_constraints": detailRes.GetFileConstraints(),
				"category":         detailRes.GetCategory(),
				"respath":          detailRes.GetRespath(),
			})
			continue
		}
		details = append(details, map[string]interface{}{
			"cred_def_id": def.GetCredDefUlid(),
			"name":        def.GetName(),
			"description": def.GetDescription(),
			"category":    def.GetCategory(),
		})
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"definitions": details,
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
	body.QualUlids = compactStrings(body.QualUlids)
	if !requireRequestField(w, body.PipelineCcUlid, "pipeline_cc_ulid") {
		return
	}
	if len(body.QualUlids) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field \"qual_ids\" is required but was empty")
		return
	}

	res, err := h.Mall.CreateCredentialApplicationOrder(r.Context(), &mallpb.CreateCredentialApplicationOrderRequest{
		CandidateUlid:  candidateID,
		PipelineCcUlid: body.PipelineCcUlid,
		QualUlids:      body.QualUlids,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	res.PaymentKey = formatPaymentKey(res.GetPaymentKey())
	WriteJSON(w, http.StatusCreated, res)
}

// ListCandidateApplications GET /api/credentials/applications
func (h *Handler) ListCandidateApplications(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	credDefID := strings.TrimSpace(r.URL.Query().Get("cred_def_id"))

	req := &gcredspb.ListApplicationsRequest{
		CandidateUlid: candidateID,
		CredDefUlid:   credDefID,
		Page:          1,
		PageSize:      100, // For now, get all applications for candidate
	}

	res, err := h.Creds.ListApplications(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// CheckUploadPermission GET /api/credentials/upload-permission
func (h *Handler) CheckUploadPermission(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	credDefID := strings.TrimSpace(r.URL.Query().Get("cred_def_id"))
	if !requireRequestFields(w, candidateID, "candidate_id", credDefID, "cred_def_id") {
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
	CredDefUlid string `json:"cred_def_id"`
	FileHash    string `json:"file_hash"`
	FileExt     string `json:"file_ext"`
	ContentType string `json:"content_type"`
	FileUsage   string `json:"file_usage"`
}

// RequestUploadUrl POST /api/credentials/upload-url
func (h *Handler) RequestUploadUrl(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	var body RequestUploadUrlReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	if !requireRequestFields(
		w,
		body.CredDefUlid, "cred_def_id",
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
	CredDefUlid string `json:"cred_def_id"`
	Files       []struct {
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
	if !requireRequestField(w, body.CredDefUlid, "cred_def_id") {
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
	AppUlid string `json:"app_id"`
	Files   []struct {
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
	if !requireRequestField(w, body.AppUlid, "app_id") {
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
