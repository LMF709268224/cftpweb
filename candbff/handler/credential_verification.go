package handler

import (
	"net/http"
	"strings"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// GetCredentialVerificationTrustAnchor exposes the public verification trust anchor.
// The PDF itself is intentionally not part of this request.
func (h *Handler) GetCredentialVerificationTrustAnchor(w http.ResponseWriter, r *http.Request) {
	if h.Creds == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "credential service unavailable")
		return
	}

	response, err := h.Creds.GetPrimaryPublicKey(r.Context(), &gcredspb.GetPrimaryPublicKeyRequest{})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if response == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "empty credential verification trust anchor response")
		return
	}

	WriteJSON(w, http.StatusOK, credentialVerificationTrustAnchorResponse{
		PublicKeyPEM:            response.GetPublicKeyPem(),
		Algorithm:               response.GetAlgorithm(),
		KeyID:                   response.GetKeyId(),
		RevokedLeafFingerprints: response.GetRevokedLeafFingerprints(),
	})
}

// CheckCredentialVerificationValidity checks the ULID and PDF hash pair online.
// Only the derived hash and certificate ID cross this boundary; no PDF bytes are uploaded.
func (h *Handler) CheckCredentialVerificationValidity(w http.ResponseWriter, r *http.Request) {
	credULID := strings.TrimSpace(r.URL.Query().Get("cred_ulid"))
	pdfHash := strings.TrimSpace(r.URL.Query().Get("pdf_hash"))
	if !requireRequestFields(w, credULID, "cred_ulid", pdfHash, "pdf_hash") {
		return
	}

	if h.Creds == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "credential service unavailable")
		return
	}

	response, err := h.Creds.CheckCredentialValidity(r.Context(), &gcredspb.CheckCredentialValidityRequest{
		CredUlid: credULID,
		PdfHash:  pdfHash,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if response == nil {
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "empty credential validity response")
		return
	}

	WriteJSON(w, http.StatusOK, credentialVerificationValidityResponse{
		Exists:       response.GetExists(),
		IsValid:      response.GetIsValid(),
		Status:       response.GetStatus(),
		CredDefName:  response.GetCredDefName(),
		IssueDate:    response.GetIssueDate(),
		ValidUntil:   response.GetValidUntil(),
		RevokedAt:    response.GetRevokedAt(),
		RevokeReason: response.GetRevokeReason(),
	})
}

type credentialVerificationTrustAnchorResponse struct {
	PublicKeyPEM            string   `json:"public_key_pem"`
	Algorithm               string   `json:"algorithm"`
	KeyID                   string   `json:"key_id"`
	RevokedLeafFingerprints []string `json:"revoked_leaf_fingerprints,omitempty"`
}

type credentialVerificationValidityResponse struct {
	Exists       bool                      `json:"exists"`
	IsValid      bool                      `json:"is_valid"`
	Status       gcredspb.CredentialStatus `json:"status"`
	CredDefName  string                    `json:"cred_def_name,omitempty"`
	IssueDate    string                    `json:"issue_date,omitempty"`
	ValidUntil   string                    `json:"valid_until,omitempty"`
	RevokedAt    string                    `json:"revoked_at,omitempty"`
	RevokeReason string                    `json:"revoke_reason,omitempty"`
}
