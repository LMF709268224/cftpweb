package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	"google.golang.org/grpc"
)

type invoiceReadClientStub struct {
	gpaypb.PayServiceClient
	listRequest  *gpaypb.ListInvoicesRequest
	countRequest *gpaypb.GetInvoiceCountRequest
}

func (s *invoiceReadClientStub) ListInvoices(_ context.Context, req *gpaypb.ListInvoicesRequest, _ ...grpc.CallOption) (*gpaypb.ListInvoicesResponse, error) {
	s.listRequest = req
	return &gpaypb.ListInvoicesResponse{
		Invoices: []*gpaypb.InvoiceSummary{{
			StripeInvoiceId: "invoice-1",
			OrderUlid:       "order-1",
			CustomerUlid:    "candidate-1",
			Amount:          12900,
			Currency:        "USD",
			Status:          gpaypb.OrderStatus(3),
			PaidAt:          1786406400,
			CreatedAt:       1786402800,
		}},
		NextCursor: "next-page",
		HasMore:    true,
	}, nil
}

func (s *invoiceReadClientStub) GetInvoiceCount(_ context.Context, req *gpaypb.GetInvoiceCountRequest, _ ...grpc.CallOption) (*gpaypb.GetInvoiceCountResponse, error) {
	s.countRequest = req
	return &gpaypb.GetInvoiceCountResponse{Count: 1}, nil
}

func TestListInvoicesReturnsNormalizedReadOnlyPage(t *testing.T) {
	client := &invoiceReadClientStub{}
	h := &Handler{Gpay: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/mall/invoices?cursor=current-page&page_size=10&sort=1", nil)

	h.ListInvoices(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest.GetCursor() != "current-page" || client.listRequest.GetPageSize() != 10 || int32(client.listRequest.GetSortOrder()) != 1 {
		t.Fatalf("list request = %+v", client.listRequest)
	}
	if client.countRequest == nil || client.countRequest.GetLimit() == 0 {
		t.Fatalf("count request = %+v", client.countRequest)
	}

	var payload struct {
		Data struct {
			Total      uint32 `json:"total"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
			Invoices   []struct {
				ID        string  `json:"id"`
				OrderID   string  `json:"order_id"`
				Email     string  `json:"email"`
				Amount    float64 `json:"amount"`
				Currency  string  `json:"currency"`
				Status    string  `json:"status"`
				CreatedAt string  `json:"created_at"`
				PaidAt    string  `json:"paid_at"`
			} `json:"invoices"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 1 || !payload.Data.HasMore || payload.Data.NextCursor != "next-page" || len(payload.Data.Invoices) != 1 {
		t.Fatalf("invoice page = %+v", payload.Data)
	}
	invoice := payload.Data.Invoices[0]
	if invoice.ID != "invoice-1" || invoice.OrderID != "order-1" || invoice.Email != "candidate-1" || invoice.Amount != 129 || invoice.Currency != "USD" {
		t.Fatalf("invoice = %+v", invoice)
	}
	if invoice.Status != gpaypb.OrderStatus(3).String() || invoice.CreatedAt == "" || invoice.PaidAt == "" {
		t.Fatalf("normalized invoice = %+v", invoice)
	}
}
