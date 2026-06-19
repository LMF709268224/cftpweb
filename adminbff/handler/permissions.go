package handler

import (
	"net/http"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

type PermissionReq struct {
	CandidateId string `json:"candidate_id"`
	CredDefId   string `json:"cred_def_id"`
	Reason      string `json:"reason"`
}

// GrantUploadPermission POST /api/permissions/grant
func (h *Handler) GrantUploadPermission(w http.ResponseWriter, r *http.Request) {
	var body PermissionReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	// Get actual OperatorId from session
	operatorID := AdminID(r)

	req := &gcredspb.GrantUploadPermissionRequest{
		CandidateId:  body.CandidateId,
		CredDefId:    body.CredDefId,
		OperatorId:   operatorID,
		Reason:       body.Reason,
		SourceSystem: "admin_ui",
	}

	res, err := h.Creds.GrantUploadPermission(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// RevokeUploadPermission POST /api/permissions/revoke
func (h *Handler) RevokeUploadPermission(w http.ResponseWriter, r *http.Request) {
	var body PermissionReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	operatorID := AdminID(r)

	req := &gcredspb.RevokeUploadPermissionRequest{
		CandidateId:  body.CandidateId,
		CredDefId:    body.CredDefId,
		OperatorId:   operatorID,
		Reason:       body.Reason,
		SourceSystem: "admin_ui",
	}

	res, err := h.Creds.RevokeUploadPermission(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// CheckCandidateQualification GET /api/permissions/check
func (h *Handler) CheckCandidateQualification(w http.ResponseWriter, r *http.Request) {
	candidateId := r.URL.Query().Get("candidate_id")
	credDefId := r.URL.Query().Get("cred_def_id")

	if candidateId == "" || credDefId == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Missing required query params")
		return
	}

	req := &gcredspb.CheckCandidateQualificationRequest{
		CandidateId: candidateId,
		CredDefId:   credDefId,
	}

	res, err := h.Creds.CheckCandidateQualification(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type MarkExpiredReq struct {
	CandidateId string `json:"candidate_id"`
	CredDefId   string `json:"cred_def_id"`
	Reason      string `json:"reason"`
}

// MarkExpired POST /api/permissions/mark-expired
func (h *Handler) MarkExpired(w http.ResponseWriter, r *http.Request) {
	var body MarkExpiredReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	auditorID := AdminID(r)

	req := &gcredspb.MarkExpiredRequest{
		CandidateId: body.CandidateId,
		CredDefId:   body.CredDefId,
		AuditorId:   auditorID,
		Reason:      body.Reason,
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

	auditorID := AdminID(r)

	req := &gcredspb.RevokeCredentialRequest{
		CandidateId: body.CandidateId,
		CredDefId:   body.CredDefId,
		AuditorId:   auditorID,
		Reason:      body.Reason,
	}

	res, err := h.Creds.RevokeCredential(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}
