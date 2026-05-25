package handler

import (
	gcreds "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"net/http"
)

// ListCertificates  GET /api/certificates  证书列表
func (h *Handler) ListCertificates(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)

	catalogsResp, err := h.Creds.ListCatalogs(r.Context(), &gcreds.ListCatalogsRequest{})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListCertificatesRsp{
		Certificates: make([]CertificateItem, 0),
	}

	for _, catalog := range catalogsResp.GetCatalogs() {
		item := CertificateItem{
			CatalogId:   catalog.GetCatalogId(),
			Name:        catalog.GetName(),
			Description: catalog.GetDescription(),
		}

		credentialResp, err := h.Creds.GetLatestCredential(r.Context(), &gcreds.GetLatestCredentialRequest{
			CandidateId: candidateID,
			CatalogId:   catalog.GetCatalogId(),
		})
		if err != nil {
			// 如果没找到该类别的证书记录，则跳过
			continue
		}

		if err == nil {
			item.CredId = credentialResp.GetCredId()
			item.CredGuid = credentialResp.GetCredGuid()
			item.CandidateId = credentialResp.GetCandidateId()
			item.Version = credentialResp.GetVersion()
			item.Status = credentialResp.GetStatus()
			item.Files = toFileInfos(credentialResp.GetFiles())
			item.AuditorId = credentialResp.GetAuditorId()
			item.AuditRemark = credentialResp.GetAuditRemark()
			item.ValidUntil = credentialResp.GetValidUntil()
			item.CreatedAt = credentialResp.GetCreatedAt()
		}

		out.Certificates = append(out.Certificates, item)
	}

	WriteJSON(w, http.StatusOK, out)
}
