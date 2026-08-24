package handler

import (
	"net/http"
	"strings"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"github.com/go-chi/chi/v5"
)

// ListCertificates GET /api/certificates 证书列表
func (h *Handler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	locale := requestLocale(r)

	credsResp, err := h.Creds.ListCandidateCredentials(r.Context(), &gcredspb.ListCandidateCredentialsRequest{
		CandidateUlid: candidateID,
		PageSize:      100,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListCertificatesRsp{
		Certificates: make([]CertificateItem, 0),
	}

	for _, cred := range credsResp.GetCredentials() {
		if cred == nil || strings.EqualFold(strings.TrimSpace(cred.GetSource()), "application") {
			continue
		}

		item := CertificateItem{
			CatalogId: cred.GetCredDefUlid(),
		}

		if defResp, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), &gcredspb.GetCredentialDefinitionDetailRequest{
			CredDefUlid: cred.GetCredDefUlid(),
		}); err == nil && defResp != nil {
			defResp = h.localizedCredentialDefinition(r.Context(), defResp, locale)
			item.Name = defResp.GetName()
			item.Description = defResp.GetDescription()
		}

		item.CredUlid = cred.GetCredUlid()
		item.CredGuid = cred.GetCredGuid()
		item.CandidateUlid = cred.GetCandidateUlid()
		item.Version = cred.GetVersion()
		item.Status = cred.GetStatus()
		item.AuditorUlid = cred.GetAuditorUlid()
		item.AuditRemark = cred.GetAuditRemark()
		item.ValidUntil = cred.GetValidUntil()
		item.CreatedAt = cred.GetCreatedAt()
		item.Source = cred.GetSource()
		if detailResp, err := h.Creds.GetCredentialDetail(r.Context(), &gcredspb.GetCredentialDetailRequest{
			CredUlid: item.CredUlid,
		}); err == nil {
			item.Files = toCertificateFiles(detailResp.GetFiles())
		}

		out.Certificates = append(out.Certificates, item)
	}

	WriteJSON(w, http.StatusOK, out)
}

// DownloadCertificate GET /api/certificates/{id}/download
func (h *Handler) DownloadCertificate(w http.ResponseWriter, r *http.Request) {
	credentialID := strings.TrimSpace(chi.URLParam(r, "id"))
	if !requireRequestField(w, credentialID, "id") {
		return
	}

	credential, err := h.Creds.GetCredentialDetail(r.Context(), &gcredspb.GetCredentialDetailRequest{
		CredUlid: credentialID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if credential.GetCandidateUlid() != CandidateID(r) {
		WriteError(w, http.StatusNotFound, ErrNotFound, "certificate not found or access denied")
		return
	}

	for _, file := range credential.GetFiles() {
		if file == nil || file.GetFileUsage() != "certificate" {
			continue
		}
		viewURL := strings.TrimSpace(file.GetViewUrl())
		if viewURL == "" {
			break
		}

		w.Header().Set("Cache-Control", "no-store")
		http.Redirect(w, r, viewURL, http.StatusTemporaryRedirect)
		return
	}

	WriteError(w, http.StatusNotFound, ErrNotFound, "certificate PDF not found")
}

func toCertificateFiles(files []*gcredspb.FileInfo) []CertificateFileInfo {
	out := make([]CertificateFileInfo, 0, len(files))
	for _, file := range files {
		if file == nil {
			continue
		}
		out = append(out, CertificateFileInfo{
			FileHash:  file.GetFileHash(),
			FileName:  file.GetFileName(),
			FileType:  file.GetFileType(),
			FileExt:   file.GetFileExt(),
			FileSize:  file.GetFileSize(),
			FileUsage: file.GetFileUsage(),
			ViewUrl:   file.GetViewUrl(),
		})
	}
	return out
}
