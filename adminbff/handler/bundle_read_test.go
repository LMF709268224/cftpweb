package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"google.golang.org/grpc"
)

type bundleReadMallClientStub struct {
	mallpb.MallServiceClient
	countRequest  *mallpb.GetBundlesAdminCountRequest
	listRequest   *mallpb.ListBundlesAdminRequest
	detailRequest *mallpb.AdminGetBundleRequest
}

func (s *bundleReadMallClientStub) GetBundlesAdminCount(_ context.Context, req *mallpb.GetBundlesAdminCountRequest, _ ...grpc.CallOption) (*mallpb.GetBundlesAdminCountResponse, error) {
	s.countRequest = req
	return &mallpb.GetBundlesAdminCountResponse{Count: 1}, nil
}

func (s *bundleReadMallClientStub) ListBundlesAdmin(_ context.Context, req *mallpb.ListBundlesAdminRequest, _ ...grpc.CallOption) (*mallpb.ListBundlesAdminResponse, error) {
	s.listRequest = req
	return &mallpb.ListBundlesAdminResponse{
		Bundles: []*mallpb.AdminBundleInfo{{
			BundleUlid:  "bundle-1",
			BundleGpath: "/bundles/regression",
			Name:        "Regression Bundle",
			Description: "Read-only bundle summary",
			Status:      "Active",
			Version:     2,
		}},
		HasMore:    true,
		NextCursor: "next-bundle-page",
	}, nil
}

func (s *bundleReadMallClientStub) AdminGetBundle(_ context.Context, req *mallpb.AdminGetBundleRequest, _ ...grpc.CallOption) (*mallpb.AdminGetBundleResponse, error) {
	s.detailRequest = req
	return &mallpb.AdminGetBundleResponse{Bundle: &mallpb.AdminBundleInfo{
		BundleUlid:  "bundle-1",
		BundleGpath: "/bundles/regression",
		Name:        "Regression Bundle",
		Description: "Read-only bundle detail",
		ItemsJson:   `[{"target_type":"pipeline","target_ulid":"pipeline-1"}]`,
		PricingJson: `{"currency":"USD","amount":12900}`,
		Status:      "Active",
		Version:     2,
	}}, nil
}

func TestListBundlesReturnsFilteredReadOnlyPage(t *testing.T) {
	client := &bundleReadMallClientStub{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/mall/bundles?status=Active&is_current_only=true&cursor=current-bundle&page_size=10", nil)

	(&Handler{Mall: client}).ListBundles(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.countRequest == nil || client.listRequest == nil {
		t.Fatal("ListBundles() did not call both read-only bundle queries")
	}
	filters := client.listRequest.GetFilters()
	if filters.GetStatus() != "Active" || !filters.GetIsCurrentOnly() || client.listRequest.GetCursor() != "current-bundle" || client.listRequest.GetPageSize() != 10 {
		t.Fatalf("bundle request = %+v", client.listRequest)
	}
	if client.countRequest.GetFilters().GetStatus() != "Active" || !client.countRequest.GetFilters().GetIsCurrentOnly() {
		t.Fatalf("bundle count request = %+v", client.countRequest)
	}

	var payload struct {
		Data struct {
			Total   uint64 `json:"total"`
			Bundles []struct {
				BundleULID string `json:"bundle_ulid"`
				Name       string `json:"name"`
			} `json:"bundles"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 1 || len(payload.Data.Bundles) != 1 || payload.Data.Bundles[0].BundleULID != "bundle-1" || payload.Data.Bundles[0].Name != "Regression Bundle" || !payload.Data.HasMore || payload.Data.NextCursor != "next-bundle-page" {
		t.Fatalf("bundle page = %+v", payload.Data)
	}
}

func TestGetBundleReturnsReadOnlyDetail(t *testing.T) {
	client := &bundleReadMallClientStub{}
	recorder := httptest.NewRecorder()

	(&Handler{Mall: client}).GetBundle(recorder, requestWithURLParam(http.MethodGet, "/api/mall/bundles/bundle-1", "bundle_ulid", "bundle-1"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest == nil || client.detailRequest.GetBundleUlid() != "bundle-1" {
		t.Fatalf("bundle detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			Bundle struct {
				BundleULID  string `json:"bundle_ulid"`
				Description string `json:"description"`
			} `json:"bundle"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Bundle.BundleULID != "bundle-1" || payload.Data.Bundle.Description != "Read-only bundle detail" {
		t.Fatalf("bundle detail = %+v", payload.Data.Bundle)
	}
}
