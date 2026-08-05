package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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

func TestMembershipHandlersRejectMissingRequiredFields(t *testing.T) {
	handler := &Handler{}

	activeRecorder := httptest.NewRecorder()
	handler.GetActiveMembership(
		activeRecorder,
		newCandidateHandlerRequest(http.MethodGet, "/api/membership/active", "", "candidate-1", nil),
	)
	assertHandlerAPIError(t, activeRecorder, http.StatusBadRequest, ErrInvalidRequest)

	cancelRecorder := httptest.NewRecorder()
	handler.CancelMembership(
		cancelRecorder,
		newCandidateHandlerRequest(http.MethodPost, "/api/membership/cancel", `{}`, "candidate-1", nil),
	)
	assertHandlerAPIError(t, cancelRecorder, http.StatusBadRequest, ErrInvalidRequest)
}
