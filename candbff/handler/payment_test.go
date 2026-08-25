package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	"google.golang.org/grpc"
)

type paymentOrderMallClientStub struct {
	mallpb.MallServiceClient
	listRequest          *mallpb.ListOrdersRequest
	listRequests         []*mallpb.ListOrdersRequest
	listResponse         *mallpb.ListOrdersResponse
	listResponses        []*mallpb.ListOrdersResponse
	countRequest         *mallpb.GetOrderCountRequest
	countResponse        *mallpb.GetOrderCountResponse
	detailRequest        *mallpb.GetOrderDetailRequest
	detailResponse       *mallpb.GetOrderDetailResponse
	bundleDetailRequest  *mallpb.GetBundleOrderDetailRequest
	bundleDetailResponse *mallpb.GetBundleOrderDetailResponse
	cancelRequest        *mallpb.CancelBusinessOrderRequest
	cancelResponse       *mallpb.CancelBusinessOrderResponse
}

type paymentOrderPayClientStub struct {
	gpaypb.PayServiceClient
	orderRequest *gpaypb.GetOrderRequest
	itemsRequest *gpaypb.ListOrderItemsRequest
}

func (s *paymentOrderPayClientStub) GetOrder(
	_ context.Context,
	request *gpaypb.GetOrderRequest,
	_ ...grpc.CallOption,
) (*gpaypb.GetOrderResponse, error) {
	s.orderRequest = request
	return &gpaypb.GetOrderResponse{
		OrderUlid:           request.GetOrderUlid(),
		Amount:              63000,
		Currency:            "usd",
		StripePaymentStatus: "paid",
		Coupons: []*gpaypb.CouponInfo{{
			Code:       "PACKAGE20",
			Name:       "Package discount",
			PercentOff: 20,
		}},
		PromoCodes: []string{"WELCOME"},
	}, nil
}

func (s *paymentOrderPayClientStub) ListOrderItems(
	_ context.Context,
	request *gpaypb.ListOrderItemsRequest,
	_ ...grpc.CallOption,
) (*gpaypb.ListOrderItemsResponse, error) {
	s.itemsRequest = request
	return &gpaypb.ListOrderItemsResponse{Items: []*gpaypb.OrderItemSummary{{
		OrderUlid: request.GetOrderUlid(),
		ItemType:  "course",
		ItemId:    "course-1",
		Title:     "Course One",
		BasePrice: 70000,
		Quantity:  1,
	}}}, nil
}

func (s *paymentOrderMallClientStub) ListOrders(
	_ context.Context,
	request *mallpb.ListOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListOrdersResponse, error) {
	s.listRequest = request
	s.listRequests = append(s.listRequests, request)
	if len(s.listResponses) > 0 {
		response := s.listResponses[0]
		s.listResponses = s.listResponses[1:]
		return response, nil
	}
	return s.listResponse, nil
}

func (s *paymentOrderMallClientStub) GetOrderCount(
	_ context.Context,
	request *mallpb.GetOrderCountRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetOrderCountResponse, error) {
	s.countRequest = request
	return s.countResponse, nil
}

func (s *paymentOrderMallClientStub) GetOrderDetail(
	_ context.Context,
	request *mallpb.GetOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetOrderDetailResponse, error) {
	s.detailRequest = request
	return s.detailResponse, nil
}

func (s *paymentOrderMallClientStub) GetBundleOrderDetail(
	_ context.Context,
	request *mallpb.GetBundleOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetBundleOrderDetailResponse, error) {
	s.bundleDetailRequest = request
	return s.bundleDetailResponse, nil
}

func (s *paymentOrderMallClientStub) CancelBusinessOrder(
	_ context.Context,
	request *mallpb.CancelBusinessOrderRequest,
	_ ...grpc.CallOption,
) (*mallpb.CancelBusinessOrderResponse, error) {
	s.cancelRequest = request
	return s.cancelResponse, nil
}

func TestCandidateOrderRawStatus(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "numeric pending payment", raw: "2", want: "2"},
		{name: "prefixed pending payment", raw: "ORDER_STATUS_PENDING_PAYMENT", want: "ORDER_STATUS_PENDING_PAYMENT"},
		{name: "plain pending", raw: "PENDING", want: "PENDING"},
		{name: "success alias", raw: "SUCCESS", want: "SUCCESS"},
		{name: "cancel alias", raw: "CANCEL", want: "CANCEL"},
		{name: "trim space", raw: " PENDING ", want: "PENDING"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := candidateOrderRawStatus(tt.raw); got != tt.want {
				t.Fatalf("candidateOrderRawStatus(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestCanCancelCommonOrderStatus(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   bool
	}{
		{name: "wait payment", status: "WAIT_PAYMENT", want: true},
		{name: "pending", status: "PENDING", want: true},
		{name: "completed", status: "COMPLETED", want: false},
		{name: "cancelled", status: "CANCELLED", want: false},
		{name: "closed", status: "CLOSED", want: false},
		{name: "business wait state is not common state", status: "WAIT_STAGE_PAYMENT", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := canCancelCommonOrderStatus(tt.status); got != tt.want {
				t.Fatalf("canCancelCommonOrderStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}

func TestListOrdersScopesFiltersAndCalculatesCompletedTotals(t *testing.T) {
	mall := &paymentOrderMallClientStub{
		listResponses: []*mallpb.ListOrdersResponse{
			{
				Items: []*mallpb.OrderSummary{
					{
						OrderUlid:     "order-1",
						CandidateUlid: "candidate-1",
						BizType:       orderBizBundlePurchase,
						BizRefUlid:    "bundle-1",
						CurrencyCode:  "USD",
						AmountMinor:   2565,
						OrderStatus:   " completed ",
						PaymentStatus: "PAID",
						CreatedAt:     "2026-08-05T01:02:03Z",
						Meta:          &mallpb.OrderMeta{ProductName: "Candidate Bundle"},
					},
					nil,
				},
				NextCursor: "next-order",
				PrevCursor: "prev-order",
				HasMore:    true,
			},
			{
				Items: []*mallpb.OrderSummary{
					{
						OrderUlid:   "order-1",
						AmountMinor: 2565,
						OrderStatus: "COMPLETED",
					},
					{
						OrderUlid:   "order-2",
						AmountMinor: 1000,
						OrderStatus: "WAIT_PAYMENT",
					},
				},
			},
		},
		countResponse: &mallpb.GetOrderCountResponse{Count: 2},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/orders?biz_type=bundle_purchase&status=completed&page_size=999",
		"",
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall}).ListOrders(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if len(mall.listRequests) != 2 {
		t.Fatalf("ListOrders calls = %d, want 2", len(mall.listRequests))
	}
	listRequest := mall.listRequests[0]
	if listRequest.GetFilters().GetCandidateUlid() != "candidate-1" ||
		listRequest.GetFilters().GetBizType() != orderBizBundlePurchase ||
		listRequest.GetFilters().GetOrderStatus() != "COMPLETED" {
		t.Fatalf("ListOrders filters = %+v", listRequest.GetFilters())
	}
	if listRequest.GetPageSize() != defaultCandidateOrderPageMax {
		t.Fatalf("page_size = %d, want %d", listRequest.GetPageSize(), defaultCandidateOrderPageMax)
	}
	if mall.countRequest == nil || mall.countRequest.GetFilters().GetCandidateUlid() != "candidate-1" {
		t.Fatalf("GetOrderCount request = %+v", mall.countRequest)
	}

	var response struct {
		Data OrderListRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	if response.Data.TotalOrders != 2 ||
		response.Data.Completed != 1 ||
		response.Data.TotalAmount != 25.65 ||
		response.Data.NextCursor != "next-order" ||
		response.Data.PrevCursor != "prev-order" ||
		!response.Data.HasMore {
		t.Fatalf("order list response = %+v", response.Data)
	}
	if len(response.Data.Orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(response.Data.Orders))
	}
	order := response.Data.Orders[0]
	if order.OrderID != "order-1" ||
		order.ProductName != "Candidate Bundle" ||
		order.OrderStatus != "COMPLETED" ||
		order.Amount != 25.65 ||
		!order.CanViewInvoice {
		t.Fatalf("order response = %+v", order)
	}
}

func TestListOrdersRejectsUnsupportedBusinessTypeBeforeCallingMall(t *testing.T) {
	mall := &paymentOrderMallClientStub{}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/orders?biz_type=admin_order",
		"",
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall}).ListOrders(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
	if len(mall.listRequests) != 0 {
		t.Fatal("unsupported business type must not call ListOrders")
	}
}

func TestGetOrderRejectsAnotherCandidatesOrder(t *testing.T) {
	mall := &paymentOrderMallClientStub{
		detailResponse: &mallpb.GetOrderDetailResponse{
			Found: true,
			Detail: &mallpb.OrderDetail{
				Summary: &mallpb.OrderSummary{
					OrderUlid:     "order-1",
					CandidateUlid: "candidate-2",
					BizType:       orderBizBundlePurchase,
					BizRefUlid:    "bundle-order-1",
				},
			},
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/orders/order-1",
		"",
		"candidate-1",
		map[string]string{"orderId": "order-1"},
	)
	recorder := httptest.NewRecorder()
	pay := &paymentOrderPayClientStub{}

	(&Handler{Mall: mall, Gpay: pay}).GetOrder(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusNotFound, ErrNotFound)
	if mall.bundleDetailRequest != nil {
		t.Fatal("another candidate's order must not load business detail")
	}
	if pay.orderRequest != nil {
		t.Fatal("another candidate's order must not load payment detail")
	}
}

func TestGetOrderReturnsCandidateScopedBusinessDetail(t *testing.T) {
	mall := &paymentOrderMallClientStub{
		detailResponse: &mallpb.GetOrderDetailResponse{
			Found: true,
			Detail: &mallpb.OrderDetail{
				Summary: &mallpb.OrderSummary{
					OrderUlid:     "order-1",
					CandidateUlid: "candidate-1",
					BizType:       orderBizBundlePurchase,
					BizRefUlid:    "bundle-order-1",
					CurrencyCode:  "sgd",
					AmountMinor:   63000,
					OrderStatus:   "COMPLETED",
					PaymentStatus: "PAID",
					Meta:          &mallpb.OrderMeta{ProductName: "Membership Package"},
				},
				GpayOrderUlid: "pay-order-1",
				PaymentKey:    "secret-value",
				PriceDetail: &mallpb.OrderPriceDetail{
					CurrencyCode:       "sgd",
					SubtotalMinor:      70000,
					DiscountTotalMinor: 7000,
					TaxTotalMinor:      0,
					TotalMinor:         63000,
					CouponCodes:        []string{"PACKAGE20"},
					PromoCodes:         []string{"WELCOME"},
				},
			},
		},
		bundleDetailResponse: &mallpb.GetBundleOrderDetailResponse{Found: true},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/orders/order-1",
		"",
		"candidate-1",
		map[string]string{"orderId": " order-1 "},
	)
	recorder := httptest.NewRecorder()
	pay := &paymentOrderPayClientStub{}

	(&Handler{Mall: mall, Gpay: pay}).GetOrder(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if mall.detailRequest == nil || mall.detailRequest.GetOrderUlid() != "order-1" {
		t.Fatalf("GetOrderDetail request = %+v", mall.detailRequest)
	}
	if mall.bundleDetailRequest == nil || mall.bundleDetailRequest.GetBundleOrderUlid() != "bundle-order-1" {
		t.Fatalf("GetBundleOrderDetail request = %+v", mall.bundleDetailRequest)
	}
	if pay.orderRequest == nil || pay.orderRequest.GetOrderUlid() != "pay-order-1" {
		t.Fatalf("GetOrder request = %+v", pay.orderRequest)
	}
	if pay.itemsRequest == nil || pay.itemsRequest.GetOrderUlid() != "pay-order-1" {
		t.Fatalf("ListOrderItems request = %+v", pay.itemsRequest)
	}
	var response struct {
		Data OrderDetailRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	if response.Data.Summary.OrderID != "order-1" ||
		response.Data.Summary.CandidateID != "candidate-1" ||
		response.Data.Summary.Currency != "SGD" ||
		response.Data.Summary.Amount != 630 ||
		!response.Data.HasPaymentKey {
		t.Fatalf("order detail response = %+v", response.Data)
	}
	pricing := response.Data.Pricing
	if pricing == nil || !pricing.Available || pricing.Source != "GMALL_ORDER_PRICE_DETAIL" ||
		pricing.BillableSubtotalMinor == nil || *pricing.BillableSubtotalMinor != 70000 ||
		pricing.PromotionDiscountMinor == nil || *pricing.PromotionDiscountMinor != 7000 ||
		pricing.TaxMinor == nil || *pricing.TaxMinor != 0 ||
		pricing.TotalMinor == nil || *pricing.TotalMinor != 63000 ||
		pricing.AmountPaidMinor == nil || *pricing.AmountPaidMinor != 63000 {
		t.Fatalf("order pricing = %+v", pricing)
	}
	if len(pricing.Items) != 1 || pricing.Items[0].Title != "Course One" || pricing.Items[0].SubtotalMinor != 70000 {
		t.Fatalf("order price items = %+v", pricing.Items)
	}
	if len(pricing.Coupons) != 1 || pricing.Coupons[0].Code != "PACKAGE20" || len(pricing.PromoCodes) != 1 || pricing.PromoCodes[0] != "WELCOME" {
		t.Fatalf("order promotions = coupons=%+v promo_codes=%+v", pricing.Coupons, pricing.PromoCodes)
	}
}

func TestCandidateOrderPricingFallsBackWithoutPaymentReference(t *testing.T) {
	pricing := (&Handler{}).candidateOrderPricing(context.Background(), "", &mallpb.OrderSummary{
		OrderUlid:    "order-1",
		AmountMinor:  5000,
		CurrencyCode: "usd",
	}, nil)

	if !pricing.Available || pricing.TotalMinor == nil || *pricing.TotalMinor != 5000 {
		t.Fatalf("fallback pricing = %+v", pricing)
	}
	if pricing.Source != "GMALL_ORDER_SUMMARY" || pricing.CurrencyCode != "USD" || pricing.UnavailableReason == "" {
		t.Fatalf("fallback metadata = %+v", pricing)
	}
}

func TestCandidateOrderExemptionsReturnsOnlyApprovedUniqueItems(t *testing.T) {
	detail := &mallpb.GetPipelineOrderDetailResponse{Detail: &mallpb.PipelineOrderDetail{
		FinalExemptionsJson: `{"stages":[{"course":[{"course_cc_ulid":"course-1","credential_ulid":"credential-1","approved":true},{"course_cc_ulid":"course-2","credential_ulid":"credential-2","approved":false}]},{"course":[{"course_cc_ulid":"course-1","credential_ulid":"credential-1","approved":true}]}]}`,
	}}

	items := candidateOrderExemptions(detail)
	if len(items) != 1 || items[0].CourseCCULID != "course-1" || items[0].CredentialULID != "credential-1" {
		t.Fatalf("approved exemptions = %+v", items)
	}
}

func TestCancelOrderRejectsUnsupportedBusinessType(t *testing.T) {
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/orders/cancel",
		`{"biz_type":"UNKNOWN","biz_ref_ulid":"order-1"}`,
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{}).CancelOrder(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusForbidden, ErrForbidden)
}

func TestCancelOrderRejectsCompletedOrder(t *testing.T) {
	mall := &paymentOrderMallClientStub{
		listResponse: &mallpb.ListOrdersResponse{
			Items: []*mallpb.OrderSummary{{
				OrderUlid:     "common-order-1",
				CandidateUlid: "candidate-1",
				BizType:       orderBizBundlePurchase,
				BizRefUlid:    "bundle-order-1",
				OrderStatus:   "COMPLETED",
			}},
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/orders/cancel",
		`{"biz_type":"BUNDLE_PURCHASE","biz_ref_ulid":"bundle-order-1"}`,
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall}).CancelOrder(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusConflict, ErrPrecondition)
	if mall.cancelRequest != nil {
		t.Fatal("completed order must not be sent to CancelBusinessOrder")
	}
}

func TestCancelOrderRejectsAnotherCandidateOrder(t *testing.T) {
	mall := &paymentOrderMallClientStub{
		listResponse: &mallpb.ListOrdersResponse{
			Items: []*mallpb.OrderSummary{{
				OrderUlid:     "common-order-1",
				CandidateUlid: "candidate-2",
				BizType:       orderBizBundlePurchase,
				BizRefUlid:    "bundle-order-1",
				OrderStatus:   "WAIT_PAYMENT",
			}},
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/orders/cancel",
		`{"biz_type":"BUNDLE_PURCHASE","biz_ref_ulid":"bundle-order-1"}`,
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall}).CancelOrder(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusNotFound, ErrNotFound)
	if mall.cancelRequest != nil {
		t.Fatal("another candidate's order must not be sent to CancelBusinessOrder")
	}
}

func TestCancelOrderForwardsCandidateScopedPendingOrder(t *testing.T) {
	mall := &paymentOrderMallClientStub{
		listResponse: &mallpb.ListOrdersResponse{
			Items: []*mallpb.OrderSummary{{
				OrderUlid:     "common-order-1",
				CandidateUlid: "candidate-1",
				BizType:       orderBizBundlePurchase,
				BizRefUlid:    "bundle-order-1",
				OrderStatus:   "WAIT_PAYMENT",
			}},
		},
		cancelResponse: &mallpb.CancelBusinessOrderResponse{
			BizType:     orderBizBundlePurchase,
			BizRefUlid:  "bundle-order-1",
			OrderStatus: "CANCELLED",
			Message:     "cancelled",
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/orders/cancel",
		`{"biz_type":" bundle_purchase ","biz_ref_ulid":" bundle-order-1 "}`,
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall}).CancelOrder(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if mall.listRequest == nil {
		t.Fatal("ListOrders was not called")
	}
	filters := mall.listRequest.GetFilters()
	if filters.GetCandidateUlid() != "candidate-1" ||
		filters.GetBizType() != orderBizBundlePurchase ||
		filters.GetBizRefUlid() != "bundle-order-1" {
		t.Fatalf("ListOrders filters = %+v", filters)
	}
	if mall.cancelRequest == nil {
		t.Fatal("CancelBusinessOrder was not called")
	}
	if mall.cancelRequest.GetCandidateUlid() != "candidate-1" ||
		mall.cancelRequest.GetBizType() != orderBizBundlePurchase ||
		mall.cancelRequest.GetBizRefUlid() != "bundle-order-1" {
		t.Fatalf("CancelBusinessOrder request = %+v", mall.cancelRequest)
	}
}
