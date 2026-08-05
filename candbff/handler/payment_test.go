package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"google.golang.org/grpc"
)

type paymentOrderMallClientStub struct {
	mallpb.MallServiceClient
	listRequest    *mallpb.ListOrdersRequest
	listResponse   *mallpb.ListOrdersResponse
	cancelRequest  *mallpb.CancelBusinessOrderRequest
	cancelResponse *mallpb.CancelBusinessOrderResponse
}

func (s *paymentOrderMallClientStub) ListOrders(
	_ context.Context,
	request *mallpb.ListOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListOrdersResponse, error) {
	s.listRequest = request
	return s.listResponse, nil
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
