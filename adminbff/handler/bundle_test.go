package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type bundleMallClientStub struct {
	mallpb.MallServiceClient
	syncCalled      bool
	metadataRequest *mallpb.UpdateBundleMetadataRequest
}

func (s *bundleMallClientStub) UpdateBundleMetadata(_ context.Context, request *mallpb.UpdateBundleMetadataRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	s.metadataRequest = request
	return &emptypb.Empty{}, nil
}

func TestUpdateBundleDetailURLPresence(t *testing.T) {
	for _, tc := range []struct {
		name, body, want string
		present          bool
	}{
		{name: "old client", body: `{"target_ulid":"bundle-1","name":"Course"}`},
		{name: "clear", body: `{"target_ulid":"bundle-1","detail_url":""}`, present: true},
		{name: "set", body: `{"target_ulid":"bundle-1","detail_url":"https://example.com/course"}`, present: true, want: "https://example.com/course"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &bundleMallClientStub{}
			recorder := httptest.NewRecorder()
			(&Handler{Mall: client}).UpdateBundleMeta(recorder, httptest.NewRequest(http.MethodPut, "/api/mall/bundles/bundle-1/meta", strings.NewReader(tc.body)))
			if recorder.Code != http.StatusOK || client.metadataRequest == nil {
				t.Fatalf("unexpected response: %d %s", recorder.Code, recorder.Body.String())
			}
			if (client.metadataRequest.DetailUrl != nil) != tc.present || client.metadataRequest.GetDetailUrl() != tc.want {
				t.Fatal("detail_url must preserve absent, empty and configured values")
			}
		})
	}
}

func (s *bundleMallClientStub) AdminSyncBundleDisplayPricing(
	_ context.Context,
	_ *mallpb.AdminSyncBundleDisplayPricingRequest,
	_ ...grpc.CallOption,
) (*mallpb.AdminSyncBundleDisplayPricingResponse, error) {
	s.syncCalled = true
	return &mallpb.AdminSyncBundleDisplayPricingResponse{}, nil
}

func TestAdminSyncBundleDisplayPricingRejectsMalformedBody(t *testing.T) {
	mall := &bundleMallClientStub{}
	h := &Handler{Mall: mall}
	req := httptest.NewRequest(http.MethodPost, "/api/mall/bundles/sync-display-pricing", strings.NewReader(`{"bundle_ulid":`))
	rec := httptest.NewRecorder()

	h.AdminSyncBundleDisplayPricing(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	if mall.syncCalled {
		t.Fatal("AdminSyncBundleDisplayPricing() called downstream service for malformed body")
	}
}

func TestAdminSyncBundleDisplayPricingAllowsEmptyBody(t *testing.T) {
	mall := &bundleMallClientStub{}
	h := &Handler{Mall: mall}
	req := httptest.NewRequest(http.MethodPost, "/api/mall/bundles/sync-display-pricing", nil)
	rec := httptest.NewRecorder()

	h.AdminSyncBundleDisplayPricing(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !mall.syncCalled {
		t.Fatal("AdminSyncBundleDisplayPricing() did not call downstream service for empty body")
	}
}
