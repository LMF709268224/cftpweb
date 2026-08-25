package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
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

type orderReadPayClientStub struct {
	gpaypb.PayServiceClient
	orderRequest   *gpaypb.GetOrderRequest
	itemsRequest   *gpaypb.ListOrderItemsRequest
	invoiceRequest *gpaypb.GetInvoiceRequest
}

func (s *orderReadPayClientStub) GetOrder(
	_ context.Context,
	req *gpaypb.GetOrderRequest,
	_ ...grpc.CallOption,
) (*gpaypb.GetOrderResponse, error) {
	s.orderRequest = req
	return &gpaypb.GetOrderResponse{
		OrderUlid:           req.GetOrderUlid(),
		Amount:              10000,
		Currency:            "usd",
		StripeInvoiceId:     "in_order_1",
		StripePaymentStatus: "paid",
		Coupons: []*gpaypb.CouponInfo{{
			Code:       "PACKAGE20",
			Name:       "Package discount",
			PercentOff: 20,
		}},
		PromoCodes: []string{"WELCOME"},
	}, nil
}

func (s *orderReadPayClientStub) ListOrderItems(
	_ context.Context,
	req *gpaypb.ListOrderItemsRequest,
	_ ...grpc.CallOption,
) (*gpaypb.ListOrderItemsResponse, error) {
	s.itemsRequest = req
	return &gpaypb.ListOrderItemsResponse{Items: []*gpaypb.OrderItemSummary{{
		OrderUlid: req.GetOrderUlid(),
		ItemType:  "course",
		ItemId:    "course-1",
		Title:     "Course One",
		BasePrice: 12000,
		Quantity:  1,
	}}}, nil
}

func (s *orderReadPayClientStub) GetInvoice(
	_ context.Context,
	req *gpaypb.GetInvoiceRequest,
	_ ...grpc.CallOption,
) (*gpaypb.GetInvoiceResponse, error) {
	s.invoiceRequest = req
	return &gpaypb.GetInvoiceResponse{
		StripeInvoiceId: req.GetStripeInvoiceId(),
		Subtotal:        12000,
		Tax:             1000,
		Total:           10000,
		AmountPaid:      10000,
		Currency:        "usd",
	}, nil
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
	payClient := &orderReadPayClientStub{}
	h := &Handler{Mall: client, Gpay: payClient}
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
	if payClient.orderRequest == nil || payClient.orderRequest.GetOrderUlid() != "order-1" {
		t.Fatalf("payment order request = %+v", payClient.orderRequest)
	}
	if payClient.itemsRequest == nil || payClient.itemsRequest.GetOrderUlid() != "order-1" {
		t.Fatalf("payment item request = %+v", payClient.itemsRequest)
	}
	if payClient.invoiceRequest == nil || payClient.invoiceRequest.GetStripeInvoiceId() != "in_order_1" {
		t.Fatalf("invoice request = %+v", payClient.invoiceRequest)
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
			Pricing struct {
				Available              bool  `json:"available"`
				BillableSubtotalMinor  int64 `json:"billable_subtotal_minor"`
				PromotionDiscountMinor int64 `json:"promotion_discount_minor"`
				TaxMinor               int64 `json:"tax_minor"`
				TotalMinor             int64 `json:"total_minor"`
				AmountPaidMinor        int64 `json:"amount_paid_minor"`
				Items                  []struct {
					Title         string `json:"title"`
					SubtotalMinor int64  `json:"subtotal_minor"`
				} `json:"items"`
				Coupons []struct {
					Code string `json:"code"`
				} `json:"coupons"`
				PromoCodes []string `json:"promo_codes"`
			} `json:"pricing"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Summary.OrderULID != "order-1" || !payload.Data.BusinessDetail.Found || payload.Data.BusinessDetail.Detail.UpdatedAt != "2026-08-11T01:00:00Z" {
		t.Fatalf("order detail = %+v", payload.Data)
	}
	pricing := payload.Data.Pricing
	if !pricing.Available || pricing.BillableSubtotalMinor != 12000 || pricing.PromotionDiscountMinor != 3000 || pricing.TaxMinor != 1000 || pricing.TotalMinor != 10000 || pricing.AmountPaidMinor != 10000 {
		t.Fatalf("order pricing = %+v", pricing)
	}
	if len(pricing.Items) != 1 || pricing.Items[0].Title != "Course One" || pricing.Items[0].SubtotalMinor != 12000 {
		t.Fatalf("order price items = %+v", pricing.Items)
	}
	if len(pricing.Coupons) != 1 || pricing.Coupons[0].Code != "PACKAGE20" || len(pricing.PromoCodes) != 1 || pricing.PromoCodes[0] != "WELCOME" {
		t.Fatalf("order promotions = coupons=%+v promo_codes=%+v", pricing.Coupons, pricing.PromoCodes)
	}
}

func TestApprovedOrderExemptionsReturnsOnlyApprovedUniqueItems(t *testing.T) {
	detail := &mallpb.GetPipelineOrderDetailResponse{Detail: &mallpb.PipelineOrderDetail{
		FinalExemptionsJson: `{"stages":[{"course":[{"course_cc_ulid":"course-1","credential_ulid":"credential-1","approved":true},{"course_cc_ulid":"course-2","credential_ulid":"credential-2","approved":false}]},{"course":[{"course_cc_ulid":"course-1","credential_ulid":"credential-1","approved":true}]}]}`,
	}}

	items := approvedOrderExemptions(detail)
	if len(items) != 1 || items[0].CourseCCULID != "course-1" || items[0].CredentialULID != "credential-1" {
		t.Fatalf("approved exemptions = %+v", items)
	}
}

func TestAdminOrderPricingFallsBackToSummaryWithoutGpay(t *testing.T) {
	h := &Handler{}
	pricing := h.adminOrderPricing(context.Background(), &mallpb.OrderSummary{
		OrderUlid:    "order-1",
		AmountMinor:  5000,
		CurrencyCode: "usd",
	})

	if !pricing.Available || pricing.TotalMinor == nil || *pricing.TotalMinor != 5000 {
		t.Fatalf("fallback pricing = %+v", pricing)
	}
	if pricing.Source != "GMALL_ORDER_SUMMARY" || pricing.CurrencyCode != "USD" || pricing.UnavailableReason == "" {
		t.Fatalf("fallback metadata = %+v", pricing)
	}
}
