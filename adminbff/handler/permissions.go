package handler

import (
	"net/http"
	"strings"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// CheckCandidateQualification GET /api/permissions/check
func (h *Handler) CheckCandidateQualification(w http.ResponseWriter, r *http.Request) {
	candidateId := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("candidate_ulid"), r.URL.Query().Get("candidate_id")))
	credDefId := strings.TrimSpace(firstNonEmpty(r.URL.Query().Get("cred_def_ulid"), r.URL.Query().Get("cred_def_id")))

	if candidateId == "" || credDefId == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Missing required query params")
		return
	}

	req := &gcredspb.CheckCandidateQualificationRequest{
		CandidateUlid: candidateId,
		CredDefUlid:   credDefId,
	}

	res, err := h.Creds.CheckCandidateQualification(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type MarkExpiredReq struct {
	CandidateUlid     string `json:"candidate_ulid"`
	LegacyCandidateID string `json:"candidate_id,omitempty"`
	CredDefUlid       string `json:"cred_def_ulid"`
	LegacyCredDefID   string `json:"cred_def_id,omitempty"`
	Reason            string `json:"reason"`
}

// MarkExpired POST /api/permissions/mark-expired
func (h *Handler) MarkExpired(w http.ResponseWriter, r *http.Request) {
	var body MarkExpiredReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	body.CandidateUlid = strings.TrimSpace(firstNonEmpty(body.CandidateUlid, body.LegacyCandidateID))
	body.CredDefUlid = strings.TrimSpace(firstNonEmpty(body.CredDefUlid, body.LegacyCredDefID))
	if !requireRequestFields(w, body.CandidateUlid, "candidate_ulid", body.CredDefUlid, "cred_def_ulid") {
		return
	}

	auditorID := AdminID(r)

	req := &gcredspb.MarkExpiredRequest{
		CandidateUlid: body.CandidateUlid,
		CredDefUlid:   body.CredDefUlid,
		AuditorUlid:   auditorID,
		Reason:        body.Reason,
	}

	res, err := h.Creds.MarkExpired(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// RevokeCredential POST /api/permissions/revoke-credential
func (h *Handler) RevokeCredential(w http.ResponseWriter, r *http.Request) {
	var body MarkExpiredReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}
	body.CandidateUlid = strings.TrimSpace(firstNonEmpty(body.CandidateUlid, body.LegacyCandidateID))
	body.CredDefUlid = strings.TrimSpace(firstNonEmpty(body.CredDefUlid, body.LegacyCredDefID))
	if !requireRequestFields(w, body.CandidateUlid, "candidate_ulid", body.CredDefUlid, "cred_def_ulid") {
		return
	}

	auditorID := AdminID(r)

	req := &gcredspb.RevokeCredentialRequest{
		CandidateUlid: body.CandidateUlid,
		CredDefUlid:   body.CredDefUlid,
		AuditorUlid:   auditorID,
		Reason:        body.Reason,
	}

	res, err := h.Creds.RevokeCredential(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}
