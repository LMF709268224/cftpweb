package handler

import (
	gcreds "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// ListCredentials GET /api/credentials
func (h *Handler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	catalogsResp, err := h.Creds.ListCatalogs(r.Context(), &gcreds.ListCatalogsRequest{})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListCredentialsRsp{
		Credentials: make([]CredentialsItem, len(catalogsResp.GetCatalogs())),
	}

	for i, catalog := range catalogsResp.GetCatalogs() {
		item := CredentialsItem{
			CatalogId:   catalog.GetCatalogId(),
			Name:        catalog.GetName(),
			Description: catalog.GetDescription(),
		}

		qualificationResp, err := h.Creds.CheckCandidateQualification(r.Context(), &gcreds.CheckCandidateQualificationRequest{
			CandidateId: candidateID,
			CatalogId:   catalog.GetCatalogId(),
		})
		if err == nil {
			item.CredentialStatus = qualificationResp.GetCredentialStatus()
			item.Message = qualificationResp.GetMessage()
			item.Eligible = qualificationResp.GetEligible()
		}

		credResp, err := h.Creds.GetLatestCredential(r.Context(), &gcreds.GetLatestCredentialRequest{
			CandidateId: candidateID,
			CatalogId:   catalog.CatalogId,
		})
		if err == nil {
			item.CredId = credResp.GetCredId()
			item.CredGuid = credResp.GetCredGuid()
			item.CandidateId = credResp.GetCandidateId()
			item.Version = credResp.GetVersion()
			item.Status = credResp.GetStatus()
			item.Files = toFileInfos(credResp.GetFiles())
			item.AuditorId = credResp.GetAuditorId()
			item.AuditRemark = credResp.GetAuditRemark()
			item.ValidUntil = credResp.GetValidUntil()
			item.CreatedAt = credResp.GetCreatedAt()
		}

		out.Credentials[i] = item
	}

	WriteJSON(w, http.StatusOK, out)
}

func toFileInfos(files []*gcreds.FileInfo) []CertificateFileInfo {
	if files == nil {
		return nil
	}

	out := make([]CertificateFileInfo, 0, len(files))
	for _, file := range files {
		out = append(out, CertificateFileInfo{
			FileHash:  file.GetFileHash(),
			FileName:  file.GetFileName(),
			FileType:  file.GetFileType(),
			FileExt:   file.GetFileExt(),
			FileSize:  file.GetFileSize(),
			FileUsage: file.GetFileUsage(),
		})
	}
	return out
}

// SubmitCredentialApplication POST /api/credentials/{catalogId}/apply
func (h *Handler) SubmitCredentialApplication(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	catalogID := chi.URLParam(r, "catalogId")
	var input struct {
		Files []CertificateFileInfo `json:"files"`
	}
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body")
		return
	}
	if len(input.Files) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "at least one file is required")
		return
	}

	pbFiles := make([]*gcreds.FileInfo, 0, len(input.Files))
	for _, f := range input.Files {
		pbFiles = append(pbFiles, &gcreds.FileInfo{
			FileHash:  f.FileHash,
			FileName:  f.FileName,
			FileType:  f.FileType,
			FileExt:   f.FileExt,
			FileSize:  f.FileSize,
			FileUsage: f.FileUsage,
		})
	}

	//TODO 返回值
	resp, err := h.Creds.SubmitApplication(r.Context(), &gcreds.SubmitApplicationRequest{
		CandidateId: candidateID,
		CatalogId:   catalogID,
		Files:       pbFiles,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
