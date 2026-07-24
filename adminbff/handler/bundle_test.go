package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"google.golang.org/grpc"
)

type bundleMallClientStub struct {
	mallpb.MallServiceClient
	syncCalled bool
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
