package handler

import (
	"net/http"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// ListCertificates GET /api/certificates 证书列表
func (h *Handler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	credsResp, err := h.Creds.ListCandidateCredentials(r.Context(), &gcredspb.ListCandidateCredentialsRequest{
		CandidateUlid: candidateID,
		Page:          1,
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
		item := CertificateItem{
			CatalogId: cred.GetCredDefUlid(),
		}

		if defResp, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), &gcredspb.GetCredentialDefinitionDetailRequest{
			CredDefUlid: cred.GetCredDefUlid(),
		}); err == nil && defResp != nil {
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
		if detailResp, err := h.Creds.GetCredentialDetail(r.Context(), &gcredspb.GetCredentialDetailRequest{
			CredUlid: item.CredUlid,
		}); err == nil {
			item.Files = toCertificateFiles(detailResp.GetFiles())
		}

		out.Certificates = append(out.Certificates, item)
	}

	WriteJSON(w, http.StatusOK, out)
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
