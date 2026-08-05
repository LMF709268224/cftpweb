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

type invoiceMallClientStub struct {
	mallpb.MallServiceClient
	listRequests  []*mallpb.ListOrdersRequest
	listResponses []*mallpb.ListOrdersResponse
}

func (s *invoiceMallClientStub) ListOrders(
	_ context.Context,
	request *mallpb.ListOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListOrdersResponse, error) {
	s.listRequests = append(s.listRequests, request)
	if len(s.listResponses) == 0 {
		return &mallpb.ListOrdersResponse{}, nil
	}
	response := s.listResponses[0]
	s.listResponses = s.listResponses[1:]
	return response, nil
}

type invoiceGpayClientStub struct {
	gpaypb.PayServiceClient
	getInvoiceRequest  *gpaypb.GetInvoiceRequest
	getInvoiceResponse *gpaypb.GetInvoiceResponse
}

func (s *invoiceGpayClientStub) GetInvoice(
	_ context.Context,
	request *gpaypb.GetInvoiceRequest,
	_ ...grpc.CallOption,
) (*gpaypb.GetInvoiceResponse, error) {
	s.getInvoiceRequest = request
	return s.getInvoiceResponse, nil
}

func TestVerifyInvoiceableOrderUsesCandidateScopeAndPagination(t *testing.T) {
	mall := &invoiceMallClientStub{
		listResponses: []*mallpb.ListOrdersResponse{
			{
				Items: []*mallpb.OrderSummary{{
					OrderUlid:   "another-order",
					OrderStatus: "COMPLETED",
				}},
				NextCursor: "next-page",
				HasMore:    true,
			},
			{
				Items: []*mallpb.OrderSummary{{
					OrderUlid:   "order-1",
					OrderStatus: " completed ",
				}},
			},
		},
	}

	if err := (&Handler{Mall: mall}).verifyInvoiceableOrder(
		context.Background(),
		" candidate-1 ",
		" order-1 ",
	); err != nil {
		t.Fatalf("verifyInvoiceableOrder() error = %v", err)
	}
	if len(mall.listRequests) != 2 {
		t.Fatalf("ListOrders calls = %d, want 2", len(mall.listRequests))
	}
	if mall.listRequests[0].GetFilters().GetCandidateUlid() != "candidate-1" ||
		mall.listRequests[0].GetCursor() != "" ||
		mall.listRequests[1].GetCursor() != "next-page" {
		t.Fatalf("ListOrders requests = %+v", mall.listRequests)
	}
}

func TestQueryInvoiceRejectsMissingOrderBeforeCallingServices(t *testing.T) {
	mall := &invoiceMallClientStub{}
	gpay := &invoiceGpayClientStub{}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/invoices/",
		"",
		"candidate-1",
		map[string]string{"orderId": "   "},
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall, Gpay: gpay}).QueryInvoice(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
	if len(mall.listRequests) != 0 || gpay.getInvoiceRequest != nil {
		t.Fatal("missing order ID must not call downstream services")
	}
}

func TestQueryInvoiceRejectsIncompleteOrderBeforeCallingGpay(t *testing.T) {
	mall := &invoiceMallClientStub{
		listResponses: []*mallpb.ListOrdersResponse{{
			Items: []*mallpb.OrderSummary{{
				OrderUlid:   "order-1",
				OrderStatus: "WAIT_PAYMENT",
			}},
		}},
	}
	gpay := &invoiceGpayClientStub{}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/invoices/order-1",
		"",
		"candidate-1",
		map[string]string{"orderId": "order-1"},
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall, Gpay: gpay}).QueryInvoice(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusConflict, ErrPrecondition)
	if gpay.getInvoiceRequest != nil {
		t.Fatal("incomplete order must not call GetInvoice")
	}
}

func TestQueryInvoiceRejectsOrderOutsideCandidateHistory(t *testing.T) {
	mall := &invoiceMallClientStub{
		listResponses: []*mallpb.ListOrdersResponse{{
			Items: []*mallpb.OrderSummary{{
				OrderUlid:   "another-order",
				OrderStatus: "COMPLETED",
			}},
		}},
	}
	gpay := &invoiceGpayClientStub{}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/invoices/order-1",
		"",
		"candidate-1",
		map[string]string{"orderId": "order-1"},
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall, Gpay: gpay}).QueryInvoice(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusForbidden, ErrForbidden)
	if gpay.getInvoiceRequest != nil {
		t.Fatal("unowned order must not call GetInvoice")
	}
}

func TestQueryInvoiceReturnsCandidateCompletedInvoice(t *testing.T) {
	mall := &invoiceMallClientStub{
		listResponses: []*mallpb.ListOrdersResponse{{
			Items: []*mallpb.OrderSummary{{
				OrderUlid:   "order-1",
				OrderStatus: "COMPLETED",
			}},
		}},
	}
	gpay := &invoiceGpayClientStub{
		getInvoiceResponse: &gpaypb.GetInvoiceResponse{
			InvoiceNumber:    "INV-001",
			Status:           "paid",
			Subtotal:         250000,
			Tax:              6500,
			Total:            256500,
			Currency:         "usd",
			HostedInvoiceUrl: "https://invoice.stripe.com/i/acct_123/invoice_456",
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/invoices/order-1",
		"",
		"candidate-1",
		map[string]string{"orderId": " order-1 "},
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall, Gpay: gpay}).QueryInvoice(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if gpay.getInvoiceRequest == nil || gpay.getInvoiceRequest.GetOrderUlid() != "order-1" {
		t.Fatalf("GetInvoice request = %+v", gpay.getInvoiceRequest)
	}
	var response struct {
		Data QueryInvoiceRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	if response.Data.InvoiceNumber != "INV-001" ||
		response.Data.Status != "paid" ||
		response.Data.SubTotal != 2500 ||
		response.Data.TotalTax != 65 ||
		response.Data.Total != 2565 ||
		response.Data.Currency != "usd" ||
		response.Data.InvoiceUrl != "https://invoice.stripe.com/i/acct_123/invoice_456" {
		t.Fatalf("invoice response = %+v", response.Data)
	}
}

func TestDownloadPdfRedirectsCompletedCandidateOrderToStripe(t *testing.T) {
	mall := &invoiceMallClientStub{
		listResponses: []*mallpb.ListOrdersResponse{{
			Items: []*mallpb.OrderSummary{{
				OrderUlid:   "order-1",
				OrderStatus: "COMPLETED",
			}},
		}},
	}
	gpay := &invoiceGpayClientStub{
		getInvoiceResponse: &gpaypb.GetInvoiceResponse{
			HostedInvoiceUrl: "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=token",
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/invoices/order-1/pdf",
		"",
		"candidate-1",
		map[string]string{"orderId": "order-1"},
	)
	recorder := httptest.NewRecorder()

	(&Handler{Mall: mall, Gpay: gpay}).DownloadPdf(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusTemporaryRedirect, recorder.Body.String())
	}
	if location := recorder.Header().Get("Location"); location != "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=token" {
		t.Fatalf("Location = %q", location)
	}
}
