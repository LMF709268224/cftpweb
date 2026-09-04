package handler

import (
	"net/http"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
)

// ListMembershipPlans GET /api/membership/plans
func (h *Handler) ListMembershipPlans(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)
	locale := requestLocale(r)

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
		plans = append(plans, h.localizedMembership(r.Context(), plan, locale))
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

type PreviewMembershipUpgradeReq struct {
	TargetMembershipULID string `json:"target_membership_ulid"`
}

type PreviewMembershipUpgradeRsp struct {
	Eligible                    bool   `json:"eligible"`
	IneligibilityReason         string `json:"ineligibility_reason,omitempty"`
	ImmediateChargeAmountMinor  int64  `json:"immediate_charge_amount_minor"`
	Currency                    string `json:"currency"`
	CurrentPeriodEndsAt         string `json:"current_period_ends_at"`
	NextCycleRenewalAmountMinor int64  `json:"next_cycle_renewal_amount_minor"`
	TargetMembershipName        string `json:"target_membership_name"`
	CurrentMembershipName       string `json:"current_membership_name"`
	ProrationDate               int64  `json:"proration_date,omitempty"`
}

// PreviewMembershipUpgrade POST /api/membership/upgrade/preview
func (h *Handler) PreviewMembershipUpgrade(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	var req PreviewMembershipUpgradeReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}

	req.TargetMembershipULID = strings.TrimSpace(req.TargetMembershipULID)
	if req.TargetMembershipULID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'target_membership_ulid' is required")
		return
	}

	resp, err := h.Mall.PreviewMembershipUpgrade(r.Context(), &mallpb.PreviewMembershipUpgradeRequest{
		CandidateUlid:        candidateID,
		TargetMembershipUlid: req.TargetMembershipULID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, PreviewMembershipUpgradeRsp{
		Eligible:                    resp.GetEligible(),
		IneligibilityReason:         resp.GetIneligibilityReason(),
		ImmediateChargeAmountMinor:  resp.GetImmediateChargeAmountMinor(),
		Currency:                    resp.GetCurrency(),
		CurrentPeriodEndsAt:         resp.GetCurrentPeriodEndsAt(),
		NextCycleRenewalAmountMinor: resp.GetNextCycleRenewalAmountMinor(),
		TargetMembershipName:        resp.GetTargetMembershipName(),
		CurrentMembershipName:       resp.GetCurrentMembershipName(),
		ProrationDate:               resp.GetProrationDate(),
	})
}

type UpgradeMembershipReq struct {
	TargetMembershipULID string `json:"target_membership_ulid"`
	IdempotencyKey       string `json:"idempotency_key"`
	ProrationDate        int64  `json:"proration_date,omitempty"`
}

type UpgradeMembershipRsp struct {
	Success              bool   `json:"success"`
	Message              string `json:"message"`
	Status               string `json:"status"`
	OrderULID            string `json:"order_ulid,omitempty"`
	ClientSecret         string `json:"client_secret,omitempty"`
	MembershipRecordULID string `json:"membership_record_ulid,omitempty"`
	StripeSubscriptionID string `json:"stripe_subscription_id,omitempty"`
	StripeInvoiceID      string `json:"stripe_invoice_id,omitempty"`
	PaidAmountMinor      int64  `json:"paid_amount_minor"`
	Currency             string `json:"currency,omitempty"`
}

// UpgradeMembership POST /api/membership/upgrade
func (h *Handler) UpgradeMembership(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if candidateID == "" {
		WriteError(w, http.StatusUnauthorized, ErrUnauthorized, "candidate not authenticated")
		return
	}

	var req UpgradeMembershipReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}

	req.TargetMembershipULID = strings.TrimSpace(req.TargetMembershipULID)
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	if req.TargetMembershipULID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'target_membership_ulid' is required")
		return
	}
	if req.ProrationDate < 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'proration_date' must be a non-negative Unix timestamp")
		return
	}

	resp, err := h.Mall.UpgradeMembership(r.Context(), &mallpb.UpgradeMembershipRequest{
		CandidateUlid:        candidateID,
		TargetMembershipUlid: req.TargetMembershipULID,
		IdempotencyKey:       req.IdempotencyKey,
		ProrationDate:        req.ProrationDate,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, UpgradeMembershipRsp{
		Success:              resp.GetSuccess(),
		Message:              resp.GetMessage(),
		Status:               resp.GetStatus(),
		OrderULID:            resp.GetOrderUlid(),
		ClientSecret:         resp.GetClientSecret(),
		MembershipRecordULID: resp.GetMembershipRecordUlid(),
		StripeSubscriptionID: resp.GetStripeSubscriptionId(),
		StripeInvoiceID:      resp.GetStripeInvoiceId(),
		PaidAmountMinor:      resp.GetPaidAmountMinor(),
		Currency:             resp.GetCurrency(),
	})
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
