package handler

import (
	"net/http"
	"strings"

	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
)

// ListMembershipPlans GET /api/membership/plans
func (h *Handler) ListMembershipPlans(w http.ResponseWriter, r *http.Request) {
	page, pageSize := parsePagination(r, 20)

	resp, err := h.Gmbr.ListMemberships(r.Context(), &gmbrpb.ListMembershipsRequest{
		Page:     int32(page),
		PageSize: int32(pageSize),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	plans := make([]*gmbrpb.Membership, 0, len(resp.GetMemberships()))
	for _, plan := range resp.GetMemberships() {
		if plan == nil {
			continue
		}
		status := strings.ToUpper(strings.TrimSpace(plan.GetStatus()))
		if !plan.GetIsCurrent() {
			continue
		}
		if status != "" && status != "ACTIVE" && status != "PUBLISHED" {
			continue
		}
		plans = append(plans, plan)
	}

	WriteJSON(w, http.StatusOK, map[string]any{
		"memberships": plans,
		"total":       len(plans),
		"page":        page,
		"page_size":   pageSize,
	})
}

// GetActiveMembership GET /api/membership/active
func (h *Handler) GetActiveMembership(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	membershipGpath := strings.TrimSpace(r.URL.Query().Get("membership_gpath"))
	if membershipGpath == "" {
		membershipGpath = strings.TrimSpace(r.URL.Query().Get("membership_path"))
	}
	if membershipGpath == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'membership_gpath' is required")
		return
	}

	resp, err := h.Gmbr.GetActiveMembership(r.Context(), &gmbrpb.GetActiveMembershipRequest{
		CandidateUlid: candidateID,
		MembershipGpath: membershipGpath,
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

	page, pageSize := parsePagination(r, 10)

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

	membershipRecordULID := strings.TrimSpace(r.URL.Query().Get("membership_record_ulid"))
	if membershipRecordULID == "" {
		membershipRecordULID = strings.TrimSpace(r.URL.Query().Get("membership_record_id"))
	}

	page, pageSize := parsePagination(r, 10)

	resp, err := h.Gmbr.ListMembershipBillings(r.Context(), &gmbrpb.ListMembershipBillingsRequest{
		CandidateUlid:        candidateID,
		MembershipRecordUlid: membershipRecordULID,
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
	MembershipRecordID   string `json:"membership_record_id"`
	MembershipRecordULID string `json:"membership_record_ulid"`
	Reason               string `json:"reason"`
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

	req.MembershipRecordULID = strings.TrimSpace(firstNonEmpty(req.MembershipRecordULID, req.MembershipRecordID))
	req.Reason = strings.TrimSpace(req.Reason)
	if req.MembershipRecordULID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'membership_record_ulid' is required")
		return
	}
	if req.Reason == "" {
		req.Reason = "user_requested"
	}

	resp, err := h.Gmbr.CancelMembership(r.Context(), &gmbrpb.CancelMembershipRequest{
		MembershipRecordUlid: req.MembershipRecordULID,
		CandidateUlid:        candidateID,
		Reason:               req.Reason,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
