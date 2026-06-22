package handler

import (
	"net/http"
	"strconv"
	"strings"

	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
)

// GetActiveMembership GET /api/membership/active
func (h *Handler) GetActiveMembership(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	resp, err := h.Gmbr.GetActiveMembership(r.Context(), &gmbrpb.GetActiveMembershipRequest{
		CandidateUlid: candidateID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ListUserMemberships GET /api/membership/history
func (h *Handler) ListUserMemberships(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 10
	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	resp, err := h.Gmbr.ListUserMemberships(r.Context(), &gmbrpb.ListUserMembershipsRequest{
		CandidateUlid: candidateID,
		Page:          int32(page),
		PageSize:      int32(pageSize),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ListMembershipBillings GET /api/membership/billings
func (h *Handler) ListMembershipBillings(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	membershipRecordID := strings.TrimSpace(r.URL.Query().Get("membership_record_id"))

	page := 1
	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 10
	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 {
			pageSize = ps
		}
	}

	resp, err := h.Gmbr.ListMembershipBillings(r.Context(), &gmbrpb.ListMembershipBillingsRequest{
		CandidateUlid:        candidateID,
		MembershipRecordUlid: membershipRecordID,
		Page:                 int32(page),
		PageSize:             int32(pageSize),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// CancelMembershipReq membership cancellation request payload
type CancelMembershipReq struct {
	MembershipRecordID string `json:"membership_record_id"`
	Reason             string `json:"reason"`
}

// CancelMembership POST /api/membership/cancel
func (h *Handler) CancelMembership(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	var req CancelMembershipReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}

	req.MembershipRecordID = strings.TrimSpace(req.MembershipRecordID)
	req.Reason = strings.TrimSpace(req.Reason)
	if req.MembershipRecordID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'membership_record_id' is required")
		return
	}
	if req.Reason == "" {
		req.Reason = "user_requested"
	}

	resp, err := h.Gmbr.CancelMembership(r.Context(), &gmbrpb.CancelMembershipRequest{
		MembershipRecordUlid: req.MembershipRecordID,
		CandidateUlid:        candidateID,
		Reason:               req.Reason,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
