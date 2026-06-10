package handler

import (
	"net/http"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// ListCertificates GET /api/certificates 证书列表
func (h *Handler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	// 1. Get all definitions first
	defsResp, err := h.Creds.ListCredentialDefinitions(r.Context(), &gcredspb.ListCredentialDefinitionsRequest{})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListCertificatesRsp{
		Certificates: make([]CertificateItem, 0),
	}

	// 2. Iterate and get latest credential for each definition
	for _, def := range defsResp.GetDefinitions() {
		item := CertificateItem{
			CatalogId:   def.GetCredDefId(), // Map CredDefId to CatalogId for frontend compatibility
			Name:        def.GetName(),
			Description: def.GetCategory(),
		}

		credResp, err := h.Creds.GetLatestCredential(r.Context(), &gcredspb.GetLatestCredentialRequest{
			CandidateId: candidateID,
			CredDefId:   def.GetCredDefId(), // Use new field
		})

		if err != nil {
			// If not found, skip adding to final certificate list, or just leave empty
			continue
		}

		item.CredId = credResp.GetCredId()
		item.CredGuid = credResp.GetCredGuid()
		item.CandidateId = credResp.GetCandidateId()
		item.Version = credResp.GetVersion()
		item.Status = credResp.GetStatus()
		item.AuditorId = credResp.GetAuditorId()
		item.AuditRemark = credResp.GetAuditRemark()
		item.ValidUntil = credResp.GetValidUntil()
		item.CreatedAt = credResp.GetCreatedAt()
		if detailResp, err := h.Creds.GetCredentialDetail(r.Context(), &gcredspb.GetCredentialDetailRequest{
			CredId: item.CredId,
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
