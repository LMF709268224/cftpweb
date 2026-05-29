package handler

import (
	"fmt"
	"net/http"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// ListCredentialDefinitions GET /api/credentials/definitions
func (h *Handler) ListCredentialDefinitions(w http.ResponseWriter, r *http.Request) {
	req := &gcredspb.ListCredentialDefinitionsRequest{}

	res, err := h.Creds.ListCredentialDefinitions(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// ListCandidateApplications GET /api/credentials/applications
func (h *Handler) ListCandidateApplications(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	req := &gcredspb.ListApplicationsRequest{
		CandidateId: candidateID,
		Page:        1,
		PageSize:    100, // For now, get all applications for candidate
	}

	res, err := h.Creds.ListApplications(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type RequestUploadUrlReq struct {
	CredDefId   string `json:"cred_def_id"`
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
		body.CredDefId, "cred_def_id",
		body.FileHash, "file_hash",
		body.FileExt, "file_ext",
		body.ContentType, "content_type",
		body.FileUsage, "file_usage",
	) {
		return
	}

	req := &gcredspb.RequestUploadUrlRequest{
		CandidateId: candidateID,
		CredDefId:   body.CredDefId,
		FileHash:    body.FileHash,
		FileExt:     body.FileExt,
		ContentType: body.ContentType,
		FileUsage:   body.FileUsage,
	}

	res, err := h.Creds.RequestUploadUrl(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type SubmitApplicationReq struct {
	CredDefId string `json:"cred_def_id"`
	Files     []struct {
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
	if !requireRequestField(w, body.CredDefId, "cred_def_id") {
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
		CandidateId: candidateID,
		CredDefId:   body.CredDefId,
		Files:       pbFiles,
	}

	res, err := h.Creds.SubmitApplication(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type UpdateApplicationReq struct {
	AppId string `json:"app_id"`
	Files []struct {
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
	if !requireRequestField(w, body.AppId, "app_id") {
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
		AppId:       body.AppId,
		CandidateId: candidateID,
		Files:       pbFiles,
	}

	res, err := h.Creds.UpdateApplication(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}
