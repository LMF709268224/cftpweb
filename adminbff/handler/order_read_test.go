package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type orderReadMallClientStub struct {
	mallpb.MallServiceClient
	listRequest         *mallpb.ListOrdersRequest
	countRequest        *mallpb.GetOrderCountRequest
	summaryRequest      *mallpb.GetOrderSummaryRequest
	bundleDetailRequest *mallpb.AdminGetBundleOrderDetailRequest
}

func (s *orderReadMallClientStub) ListOrders(
	_ context.Context,
	req *mallpb.ListOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListOrdersResponse, error) {
	s.listRequest = req
	return &mallpb.ListOrdersResponse{
		Items: []*mallpb.OrderSummary{{
			OrderUlid:     "order-1",
			CandidateUlid: "candidate-1",
			BizType:       "BUNDLE_PURCHASE",
			BizRefUlid:    "bundle-order-1",
			CurrencyCode:  "usd",
			AmountMinor:   12900,
			OrderStatus:   "paid",
			PaymentStatus: "paid",
			CreatedAt:     "2026-08-11T00:00:00Z",
			Meta:          &mallpb.OrderMeta{ProductName: "Regression Bundle"},
		}},
		NextCursor: "next-page",
		HasMore:    true,
	}, nil
}

func (s *orderReadMallClientStub) GetOrderCount(
	_ context.Context,
	req *mallpb.GetOrderCountRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetOrderCountResponse, error) {
	s.countRequest = req
	return &mallpb.GetOrderCountResponse{Count: 1}, nil
}

func (s *orderReadMallClientStub) GetOrderSummary(
	_ context.Context,
	req *mallpb.GetOrderSummaryRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetOrderSummaryResponse, error) {
	s.summaryRequest = req
	return &mallpb.GetOrderSummaryResponse{
		Found: true,
		Summary: &mallpb.OrderSummary{
			OrderUlid:  req.GetOrderUlid(),
			BizType:    "BUNDLE_PURCHASE",
			BizRefUlid: "bundle-order-1",
		},
	}, nil
}

func (s *orderReadMallClientStub) AdminGetBundleOrderDetail(
	_ context.Context,
	req *mallpb.AdminGetBundleOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.AdminGetBundleOrderDetailResponse, error) {
	s.bundleDetailRequest = req
	return &mallpb.AdminGetBundleOrderDetailResponse{
		Found: true,
		Detail: &mallpb.AdminBundleOrderDetail{
			UpdatedAt: "2026-08-11T01:00:00Z",
		},
	}, nil
}

func TestListOrdersReturnsFilteredReadOnlyPage(t *testing.T) {
	client := &orderReadMallClientStub{}
	h := &Handler{Mall: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/mall/orders?candidate_ulid=candidate-1&biz_type=bundle_purchase&order_status=paid&payment_status=paid&page_size=10",
		nil,
	)

	h.ListOrders(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil || client.countRequest == nil {
		t.Fatal("ListOrders() did not call both read-only order queries")
	}
	filters := client.listRequest.GetFilters()
	if filters.GetCandidateUlid() != "candidate-1" || filters.GetBizType() != "BUNDLE_PURCHASE" || filters.GetOrderStatus() != "PAID" || filters.GetPaymentStatus() != "PAID" {
		t.Fatalf("order filters = %+v", filters)
	}
	if client.listRequest.GetPageSize() != 10 {
		t.Fatalf("page_size = %d, want 10", client.listRequest.GetPageSize())
	}

	var payload struct {
		Data struct {
			Items []struct {
				OrderULID    string `json:"order_ulid"`
				ProductName  string `json:"product_name"`
				CurrencyCode string `json:"currency_code"`
				OrderStatus  string `json:"order_status"`
			} `json:"items"`
			Total      int32  `json:"total"`
			NextCursor string `json:"next_cursor"`
			HasMore    bool   `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Items) != 1 || payload.Data.Total != 1 || payload.Data.NextCursor != "next-page" || !payload.Data.HasMore {
		t.Fatalf("order page = %+v", payload.Data)
	}
	item := payload.Data.Items[0]
	if item.OrderULID != "order-1" || item.ProductName != "Regression Bundle" || item.CurrencyCode != "USD" || item.OrderStatus != "PAID" {
		t.Fatalf("order item = %+v", item)
	}
}

func TestGetOrderDetailReturnsReadOnlyBusinessDetail(t *testing.T) {
	client := &orderReadMallClientStub{}
	h := &Handler{Mall: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/mall/orders/order-1", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("order_ulid", "order-1")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

	h.GetOrderDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.summaryRequest == nil || client.summaryRequest.GetOrderUlid() != "order-1" {
		t.Fatalf("summary request = %+v", client.summaryRequest)
	}
	if client.bundleDetailRequest == nil || client.bundleDetailRequest.GetBundleOrderUlid() != "bundle-order-1" {
		t.Fatalf("bundle detail request = %+v", client.bundleDetailRequest)
	}

	var payload struct {
		Data struct {
			Summary struct {
				OrderULID string `json:"order_ulid"`
			} `json:"summary"`
			BusinessDetail struct {
				Found  bool `json:"found"`
				Detail struct {
					UpdatedAt string `json:"updated_at"`
				} `json:"detail"`
			} `json:"business_detail"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Summary.OrderULID != "order-1" || !payload.Data.BusinessDetail.Found || payload.Data.BusinessDetail.Detail.UpdatedAt != "2026-08-11T01:00:00Z" {
		t.Fatalf("order detail = %+v", payload.Data)
	}
}
