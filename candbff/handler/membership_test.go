package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	"google.golang.org/grpc"
)

type membershipRegressionClient struct {
	gmbrpb.GmbrServiceClient

	listRequest     *gmbrpb.ListMembershipsRequest
	activeRequest   *gmbrpb.GetActiveMembershipRequest
	historyRequest  *gmbrpb.ListUserMembershipsRequest
	billingsRequest *gmbrpb.ListMembershipBillingsRequest
	cancelRequest   *gmbrpb.CancelMembershipRequest
}

type membershipUpgradeRegressionClient struct {
	mallpb.MallServiceClient

	previewRequest  *mallpb.PreviewMembershipUpgradeRequest
	upgradeRequest  *mallpb.UpgradeMembershipRequest
	previewResponse *mallpb.PreviewMembershipUpgradeResponse
	upgradeResponse *mallpb.UpgradeMembershipResponse
}

func (c *membershipUpgradeRegressionClient) PreviewMembershipUpgrade(
	_ context.Context,
	request *mallpb.PreviewMembershipUpgradeRequest,
	_ ...grpc.CallOption,
) (*mallpb.PreviewMembershipUpgradeResponse, error) {
	c.previewRequest = request
	if c.previewResponse != nil {
		return c.previewResponse, nil
	}
	return &mallpb.PreviewMembershipUpgradeResponse{
		Eligible:                   true,
		ImmediateChargeAmountMinor: 1560,
		Currency:                   "usd",
	}, nil
}

func (c *membershipUpgradeRegressionClient) UpgradeMembership(
	_ context.Context,
	request *mallpb.UpgradeMembershipRequest,
	_ ...grpc.CallOption,
) (*mallpb.UpgradeMembershipResponse, error) {
	c.upgradeRequest = request
	if c.upgradeResponse != nil {
		return c.upgradeResponse, nil
	}
	return &mallpb.UpgradeMembershipResponse{
		Success:              true,
		MembershipRecordUlid: "record-2",
		PaidAmountMinor:      1560,
		Currency:             "usd",
	}, nil
}

func (c *membershipRegressionClient) ListMemberships(
	_ context.Context,
	request *gmbrpb.ListMembershipsRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.ListMembershipsResponse, error) {
	c.listRequest = request
	return &gmbrpb.ListMembershipsResponse{
		Memberships: []*gmbrpb.Membership{
			nil,
			{MembershipUlid: "active", Name: "Active", Status: "ACTIVE", IsCurrent: true},
			{MembershipUlid: "published", Name: "Published", Status: "published", IsCurrent: true},
			{MembershipUlid: "draft", Name: "Draft", Status: "DRAFT", IsCurrent: true},
			{MembershipUlid: "old", Name: "Old", Status: "ACTIVE", IsCurrent: false},
		},
		NextCursor: "next-memberships",
		HasMore:    true,
	}, nil
}

func (c *membershipRegressionClient) GetActiveMembership(
	_ context.Context,
	request *gmbrpb.GetActiveMembershipRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.GetActiveMembershipResponse, error) {
	c.activeRequest = request
	return &gmbrpb.GetActiveMembershipResponse{}, nil
}

func (c *membershipRegressionClient) ListUserMemberships(
	_ context.Context,
	request *gmbrpb.ListUserMembershipsRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.ListUserMembershipsResponse, error) {
	c.historyRequest = request
	return &gmbrpb.ListUserMembershipsResponse{}, nil
}

func (c *membershipRegressionClient) ListMembershipBillings(
	_ context.Context,
	request *gmbrpb.ListMembershipBillingsRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.ListMembershipBillingsResponse, error) {
	c.billingsRequest = request
	return &gmbrpb.ListMembershipBillingsResponse{}, nil
}

func (c *membershipRegressionClient) CancelMembership(
	_ context.Context,
	request *gmbrpb.CancelMembershipRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.CancelMembershipResponse, error) {
	c.cancelRequest = request
	return &gmbrpb.CancelMembershipResponse{Success: true}, nil
}

func TestListMembershipPlansOnlyReturnsCurrentPublishedPlans(t *testing.T) {
	client := &membershipRegressionClient{}
	handler := &Handler{Gmbr: client}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/membership/plans?page_size=7&cursor=cursor-1",
		"",
		"candidate-1",
		nil,
	)

	handler.ListMembershipPlans(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil {
		t.Fatal("ListMemberships was not called")
	}
	if client.listRequest.GetPageSize() != 7 || client.listRequest.GetCursor() != "cursor-1" {
		t.Fatalf("pagination = (%d, %q), want (7, %q)", client.listRequest.GetPageSize(), client.listRequest.GetCursor(), "cursor-1")
	}

	var response struct {
		Data struct {
			Memberships []*gmbrpb.Membership `json:"memberships"`
			Total       int                  `json:"total"`
			NextCursor  string               `json:"next_cursor"`
			HasMore     bool                 `json:"has_more"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.Total != 2 || len(response.Data.Memberships) != 2 {
		t.Fatalf("filtered memberships = %d/%d, want 2/2", response.Data.Total, len(response.Data.Memberships))
	}
	if response.Data.Memberships[0].GetMembershipUlid() != "active" ||
		response.Data.Memberships[1].GetMembershipUlid() != "published" {
		t.Fatalf("unexpected memberships: %#v", response.Data.Memberships)
	}
	if response.Data.NextCursor != "next-memberships" || !response.Data.HasMore {
		t.Fatalf("pagination response = (%q, %t)", response.Data.NextCursor, response.Data.HasMore)
	}
}

func TestMembershipHandlersKeepCandidateScope(t *testing.T) {
	client := &membershipRegressionClient{}
	handler := &Handler{Gmbr: client}

	activeRecorder := httptest.NewRecorder()
	handler.GetActiveMembership(
		activeRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/membership/active?membership_path=%2Fmembership%2Fprofessional",
			"",
			"candidate-1",
			nil,
		),
	)
	if activeRecorder.Code != http.StatusOK {
		t.Fatalf("active status = %d; body=%q", activeRecorder.Code, activeRecorder.Body.String())
	}
	if client.activeRequest.GetCandidateUlid() != "candidate-1" ||
		client.activeRequest.GetMembershipGpath() != "/membership/professional" {
		t.Fatalf("active request = %#v", client.activeRequest)
	}

	historyRecorder := httptest.NewRecorder()
	handler.ListUserMemberships(
		historyRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/membership/history?page_size=6&cursor=history-cursor",
			"",
			"candidate-1",
			nil,
		),
	)
	if historyRecorder.Code != http.StatusOK {
		t.Fatalf("history status = %d; body=%q", historyRecorder.Code, historyRecorder.Body.String())
	}
	if client.historyRequest.GetFilters().GetCandidateUlid() != "candidate-1" ||
		client.historyRequest.GetPageSize() != 6 ||
		client.historyRequest.GetCursor() != "history-cursor" {
		t.Fatalf("history request = %#v", client.historyRequest)
	}

	billingsRecorder := httptest.NewRecorder()
	handler.ListMembershipBillings(
		billingsRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/membership/billings?membership_record_id=record-1",
			"",
			"candidate-1",
			nil,
		),
	)
	if billingsRecorder.Code != http.StatusOK {
		t.Fatalf("billings status = %d; body=%q", billingsRecorder.Code, billingsRecorder.Body.String())
	}
	if client.billingsRequest.GetFilters().GetCandidateUlid() != "candidate-1" ||
		client.billingsRequest.GetFilters().GetMembershipRecordUlid() != "record-1" {
		t.Fatalf("billings request = %#v", client.billingsRequest)
	}

	cancelRecorder := httptest.NewRecorder()
	handler.CancelMembership(
		cancelRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/membership/cancel",
			`{"membership_record_id":" record-1 "}`,
			"candidate-1",
			nil,
		),
	)
	if cancelRecorder.Code != http.StatusOK {
		t.Fatalf("cancel status = %d; body=%q", cancelRecorder.Code, cancelRecorder.Body.String())
	}
	if client.cancelRequest.GetCandidateUlid() != "candidate-1" ||
		client.cancelRequest.GetMembershipRecordUlid() != "record-1" ||
		client.cancelRequest.GetReason() != "user_requested" {
		t.Fatalf("cancel request = %#v", client.cancelRequest)
	}
}

func TestMembershipUpgradeHandlersKeepCandidateScope(t *testing.T) {
	client := &membershipUpgradeRegressionClient{}
	handler := &Handler{Mall: client}

	previewRecorder := httptest.NewRecorder()
	handler.PreviewMembershipUpgrade(
		previewRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/membership/upgrade/preview",
			`{"candidate_ulid":"forged-candidate","target_membership_ulid":" target-2 ","currency":" usd "}`,
			"candidate-1",
			nil,
		),
	)
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d; body=%q", previewRecorder.Code, previewRecorder.Body.String())
	}
	if client.previewRequest.GetCandidateUlid() != "candidate-1" ||
		client.previewRequest.GetTargetMembershipUlid() != "target-2" ||
		client.previewRequest.GetCurrency() != "usd" {
		t.Fatalf("preview request = %#v", client.previewRequest)
	}

	upgradeRecorder := httptest.NewRecorder()
	handler.UpgradeMembership(
		upgradeRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/membership/upgrade",
			`{"candidate_ulid":"forged-candidate","target_membership_ulid":" target-2 ","currency":" usd ","idempotency_key":" request-1 "}`,
			"candidate-1",
			nil,
		),
	)
	if upgradeRecorder.Code != http.StatusOK {
		t.Fatalf("upgrade status = %d; body=%q", upgradeRecorder.Code, upgradeRecorder.Body.String())
	}
	if client.upgradeRequest.GetCandidateUlid() != "candidate-1" ||
		client.upgradeRequest.GetTargetMembershipUlid() != "target-2" ||
		client.upgradeRequest.GetCurrency() != "usd" ||
		client.upgradeRequest.GetIdempotencyKey() != "request-1" {
		t.Fatalf("upgrade request = %#v", client.upgradeRequest)
	}
}

func TestMembershipUpgradeResponsesPreserveRequiredZeroValues(t *testing.T) {
	client := &membershipUpgradeRegressionClient{
		previewResponse: &mallpb.PreviewMembershipUpgradeResponse{
			Eligible:            false,
			IneligibilityReason: "upgrade unavailable",
		},
		upgradeResponse: &mallpb.UpgradeMembershipResponse{
			Success: false,
			Message: "upgrade failed",
		},
	}
	handler := &Handler{Mall: client}

	previewRecorder := httptest.NewRecorder()
	handler.PreviewMembershipUpgrade(
		previewRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/membership/upgrade/preview",
			`{"target_membership_ulid":"target-2"}`,
			"candidate-1",
			nil,
		),
	)
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("preview status = %d; body=%q", previewRecorder.Code, previewRecorder.Body.String())
	}
	var previewPayload struct {
		Data struct {
			Eligible                    *bool  `json:"eligible"`
			ImmediateChargeAmountMinor  *int64 `json:"immediate_charge_amount_minor"`
			NextCycleRenewalAmountMinor *int64 `json:"next_cycle_renewal_amount_minor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(previewRecorder.Body.Bytes(), &previewPayload); err != nil {
		t.Fatalf("decode preview response: %v", err)
	}
	if previewPayload.Data.Eligible == nil || *previewPayload.Data.Eligible {
		t.Fatalf("preview eligible = %v, want explicit false", previewPayload.Data.Eligible)
	}
	if previewPayload.Data.ImmediateChargeAmountMinor == nil || *previewPayload.Data.ImmediateChargeAmountMinor != 0 {
		t.Fatalf("preview immediate charge = %v, want explicit zero", previewPayload.Data.ImmediateChargeAmountMinor)
	}
	if previewPayload.Data.NextCycleRenewalAmountMinor == nil || *previewPayload.Data.NextCycleRenewalAmountMinor != 0 {
		t.Fatalf("preview renewal amount = %v, want explicit zero", previewPayload.Data.NextCycleRenewalAmountMinor)
	}

	upgradeRecorder := httptest.NewRecorder()
	handler.UpgradeMembership(
		upgradeRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/membership/upgrade",
			`{"target_membership_ulid":"target-2","idempotency_key":"request-1"}`,
			"candidate-1",
			nil,
		),
	)
	if upgradeRecorder.Code != http.StatusOK {
		t.Fatalf("upgrade status = %d; body=%q", upgradeRecorder.Code, upgradeRecorder.Body.String())
	}
	var upgradePayload struct {
		Data struct {
			Success         *bool  `json:"success"`
			Message         string `json:"message"`
			PaidAmountMinor *int64 `json:"paid_amount_minor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(upgradeRecorder.Body.Bytes(), &upgradePayload); err != nil {
		t.Fatalf("decode upgrade response: %v", err)
	}
	if upgradePayload.Data.Success == nil || *upgradePayload.Data.Success {
		t.Fatalf("upgrade success = %v, want explicit false", upgradePayload.Data.Success)
	}
	if upgradePayload.Data.Message != "upgrade failed" {
		t.Fatalf("upgrade message = %q, want %q", upgradePayload.Data.Message, "upgrade failed")
	}
	if upgradePayload.Data.PaidAmountMinor == nil || *upgradePayload.Data.PaidAmountMinor != 0 {
		t.Fatalf("upgrade paid amount = %v, want explicit zero", upgradePayload.Data.PaidAmountMinor)
	}
}

func TestMembershipHandlersRejectMissingRequiredFields(t *testing.T) {
	gmbrClient := &membershipRegressionClient{}
	handler := &Handler{Gmbr: gmbrClient}

	activeRecorder := httptest.NewRecorder()
	handler.GetActiveMembership(
		activeRecorder,
		newCandidateHandlerRequest(http.MethodGet, "/api/membership/active", "", "candidate-1", nil),
	)
	if activeRecorder.Code != http.StatusOK {
		t.Fatalf("active status = %d; body=%q", activeRecorder.Code, activeRecorder.Body.String())
	}
	if gmbrClient.activeRequest.GetMembershipGpath() != "" {
		t.Fatalf("active membership gpath = %q, want empty", gmbrClient.activeRequest.GetMembershipGpath())
	}

	cancelRecorder := httptest.NewRecorder()
	handler.CancelMembership(
		cancelRecorder,
		newCandidateHandlerRequest(http.MethodPost, "/api/membership/cancel", `{}`, "candidate-1", nil),
	)
	assertHandlerAPIError(t, cancelRecorder, http.StatusBadRequest, ErrInvalidRequest)

	upgradeHandler := &Handler{}
	previewRecorder := httptest.NewRecorder()
	upgradeHandler.PreviewMembershipUpgrade(
		previewRecorder,
		newCandidateHandlerRequest(http.MethodPost, "/api/membership/upgrade/preview", `{}`, "candidate-1", nil),
	)
	assertHandlerAPIError(t, previewRecorder, http.StatusBadRequest, ErrInvalidRequest)

	upgradeRecorder := httptest.NewRecorder()
	upgradeHandler.UpgradeMembership(
		upgradeRecorder,
		newCandidateHandlerRequest(http.MethodPost, "/api/membership/upgrade", `{}`, "candidate-1", nil),
	)
	assertHandlerAPIError(t, upgradeRecorder, http.StatusBadRequest, ErrInvalidRequest)
}
