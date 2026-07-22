package handler

import (
	"net/http"
	"strings"

	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
)

// ListMembershipPlans GET /api/membership/plans
func (h *Handler) ListMembershipPlans(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)

	resp, err := h.Gmbr.ListMemberships(r.Context(), &gmbrpb.ListMembershipsRequest{
		PageSize:  page.PageSize,
		SortOrder: gmbrpb.SortOrder(page.Sort),
		Cursor:    page.Cursor,
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
		"page_size":   page.PageSize,
		"next_cursor": resp.GetNextCursor(),
		"has_more":    resp.GetHasMore(),
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
		CandidateUlid:   candidateID,
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

	page := parseCursorPage(r, 10)

	resp, err := h.Gmbr.ListUserMemberships(r.Context(), &gmbrpb.ListUserMembershipsRequest{
		Filters: &gmbrpb.UserMembershipFilters{
			CandidateUlid: candidateID,
		},
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gmbrpb.SortOrder(page.Sort),
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

	page := parseCursorPage(r, 10)

	resp, err := h.Gmbr.ListMembershipBillings(r.Context(), &gmbrpb.ListMembershipBillingsRequest{
		Filters: &gmbrpb.MembershipBillingFilters{
			CandidateUlid:        candidateID,
			MembershipRecordUlid: membershipRecordULID,
		},
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gmbrpb.SortOrder(page.Sort),
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
