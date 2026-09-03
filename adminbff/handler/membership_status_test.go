package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	"google.golang.org/grpc"
)

type membershipStatusClientStub struct {
	gmbrpb.GmbrServiceClient
	updateRequest    *gmbrpb.AdminUpdateMembershipConfigRequest
	deprecateRequest *gmbrpb.AdminDeprecateMembershipConfigRequest
}

func (s *membershipStatusClientStub) AdminUpdateMembershipConfig(_ context.Context, req *gmbrpb.AdminUpdateMembershipConfigRequest, _ ...grpc.CallOption) (*gmbrpb.AdminUpdateMembershipConfigResponse, error) {
	s.updateRequest = req
	return &gmbrpb.AdminUpdateMembershipConfigResponse{Success: true, Status: req.GetStatus()}, nil
}

func (s *membershipStatusClientStub) AdminDeprecateMembershipConfig(_ context.Context, req *gmbrpb.AdminDeprecateMembershipConfigRequest, _ ...grpc.CallOption) (*gmbrpb.AdminDeprecateMembershipConfigResponse, error) {
	s.deprecateRequest = req
	return &gmbrpb.AdminDeprecateMembershipConfigResponse{Success: true}, nil
}

func TestAdminDeprecateMembershipConfigUsesPhysicalMembershipID(t *testing.T) {
	client := &membershipStatusClientStub{}
	recorder := httptest.NewRecorder()
	request := requestWithURLParam(
		http.MethodPost,
		"/api/memberships/01KZZKAW2FZDZD9S68NFAYPAN5/deprecate",
		"membership_ulid",
		"01KZZKAW2FZDZD9S68NFAYPAN5",
	)

	(&Handler{Gmbr: client}).AdminDeprecateMembershipConfig(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.deprecateRequest == nil {
		t.Fatal("AdminDeprecateMembershipConfig was not called")
	}
	if client.deprecateRequest.GetMembershipUlid() != "01KZZKAW2FZDZD9S68NFAYPAN5" {
		t.Fatalf("membership_ulid = %q", client.deprecateRequest.GetMembershipUlid())
	}
	if client.deprecateRequest.MembershipUlid == nil {
		t.Fatal("membership_ulid optional field was not set")
	}
	if client.deprecateRequest.TierLevel != nil || client.deprecateRequest.GetMembershipGpath() != "" {
		t.Fatalf("unexpected category fallback fields: %+v", client.deprecateRequest)
	}
}

func TestAdminUpdateMembershipConfigForwardsReactivationStatus(t *testing.T) {
	client := &membershipStatusClientStub{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodPut,
		"/api/memberships",
		strings.NewReader(`{"membership_gpath":"/memberships/regression","tier_level":2,"status":"Active"}`),
	)

	(&Handler{Gmbr: client}).AdminUpdateMembershipConfig(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.updateRequest == nil {
		t.Fatal("AdminUpdateMembershipConfig was not called")
	}
	if client.updateRequest.GetMembershipGpath() != "/memberships/regression" || client.updateRequest.GetTierLevel() != 2 {
		t.Fatalf("membership locator = (%q, %d)", client.updateRequest.GetMembershipGpath(), client.updateRequest.GetTierLevel())
	}
	if client.updateRequest.Status == nil || client.updateRequest.GetStatus() != "Active" {
		t.Fatalf("status = %q, want Active", client.updateRequest.GetStatus())
	}
}
