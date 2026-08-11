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

type membershipReadClientStub struct {
	gmbrpb.GmbrServiceClient
	listRequest   *gmbrpb.AdminListMembershipsRequest
	detailRequest *gmbrpb.GetMembershipRequest
}

func (s *membershipReadClientStub) AdminListMemberships(_ context.Context, req *gmbrpb.AdminListMembershipsRequest, _ ...grpc.CallOption) (*gmbrpb.AdminListMembershipsResponse, error) {
	s.listRequest = req
	return &gmbrpb.AdminListMembershipsResponse{
		Memberships: []*gmbrpb.AdminMembership{{
			MembershipUlid:   "membership-1",
			MembershipGpath:  "/memberships/regression",
			Name:             "Regression Membership",
			Description:      "Read-only membership summary",
			DurationInMonths: 12,
			TierLevel:        2,
			Status:           "Active",
			Version:          4,
		}},
		HasMore:    true,
		NextCursor: "next-membership-page",
	}, nil
}

func (s *membershipReadClientStub) GetMembership(_ context.Context, req *gmbrpb.GetMembershipRequest, _ ...grpc.CallOption) (*gmbrpb.Membership, error) {
	s.detailRequest = req
	return &gmbrpb.Membership{
		MembershipUlid:   "membership-1",
		MembershipGpath:  "/memberships/regression",
		Name:             "Regression Membership",
		Description:      "Read-only membership detail",
		FeaturesJson:     `["Priority support","Course discount"]`,
		IdealFor:         "Regression users",
		DurationInMonths: 12,
		CasdoorRoleName:  "member-pro",
		TierLevel:        2,
		Status:           "Active",
		Version:          4,
	}, nil
}

func TestAdminListMembershipConfigsReturnsReadOnlyPage(t *testing.T) {
	client := &membershipReadClientStub{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/memberships/configs?cursor=current-membership&page_size=10", nil)

	(&Handler{Gmbr: client}).AdminListMembershipConfigs(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil || client.listRequest.GetCursor() != "current-membership" || client.listRequest.GetPageSize() != 10 {
		t.Fatalf("membership request = %+v", client.listRequest)
	}
	var payload struct {
		Data struct {
			Memberships []struct {
				MembershipULID string `json:"membership_ulid"`
				Name           string `json:"name"`
			} `json:"memberships"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Memberships) != 1 || payload.Data.Memberships[0].MembershipULID != "membership-1" || payload.Data.Memberships[0].Name != "Regression Membership" || !payload.Data.HasMore || payload.Data.NextCursor != "next-membership-page" {
		t.Fatalf("membership page = %+v", payload.Data)
	}
}

func TestGetMembershipReturnsReadOnlyDetail(t *testing.T) {
	client := &membershipReadClientStub{}
	recorder := httptest.NewRecorder()

	(&Handler{Gmbr: client}).GetMembership(recorder, requestWithURLParam(http.MethodGet, "/api/memberships/membership-1", "membership_ulid", "membership-1"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest == nil || client.detailRequest.GetMembershipUlid() != "membership-1" {
		t.Fatalf("membership detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			MembershipULID string `json:"membership_ulid"`
			Description    string `json:"description"`
			TierLevel      int32  `json:"tier_level"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.MembershipULID != "membership-1" || payload.Data.Description != "Read-only membership detail" || payload.Data.TierLevel != 2 {
		t.Fatalf("membership detail = %+v", payload.Data)
	}
}
