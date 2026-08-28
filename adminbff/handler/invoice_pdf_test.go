package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type invoicePDFClientStub struct {
	gpaypb.PayServiceClient
	request  *gpaypb.GetInvoiceRequest
	response *gpaypb.GetInvoiceResponse
}

func (s *invoicePDFClientStub) GetInvoice(_ context.Context, req *gpaypb.GetInvoiceRequest, _ ...grpc.CallOption) (*gpaypb.GetInvoiceResponse, error) {
	s.request = req
	return s.response, nil
}

type adminInvoiceRoundTripper func(*http.Request) (*http.Response, error)

func (f adminInvoiceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestGetInvoicePDFReturnsValidatedStripeURL(t *testing.T) {
	client := &invoicePDFClientStub{response: &gpaypb.GetInvoiceResponse{
		HostedInvoiceUrl: "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=token",
	}}
	request := httptest.NewRequest(http.MethodGet, "/api/mall/invoices/order-1/pdf", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("order_ulid", "order-1")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	recorder := httptest.NewRecorder()

	(&Handler{Gpay: client}).GetInvoicePDF(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request == nil || client.request.GetOrderUlid() != "order-1" {
		t.Fatalf("GetInvoice request = %+v", client.request)
	}
	var payload struct {
		Data InvoicePDFResponse `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.PDFURL != "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=token" {
		t.Fatalf("PDF URL = %q", payload.Data.PDFURL)
	}
}

func TestValidateAdminStripeInvoicePDFURLRejectsUntrustedURLs(t *testing.T) {
	for _, rawURL := range []string{
		"http://invoice.stripe.com/i/acct_123/invoice_456/pdf",
		"https://invoice.stripe.com.evil.example/i/acct_123/invoice_456/pdf",
		"https://user@invoice.stripe.com/i/acct_123/invoice_456/pdf",
		"https://invoice.stripe.com:8443/i/acct_123/invoice_456/pdf",
		"https://pay.stripe.com/not-an-invoice/pdf",
	} {
		if _, err := validateAdminStripeInvoicePDFURL(rawURL); err == nil {
			t.Fatalf("validateAdminStripeInvoicePDFURL(%q) error = nil, want rejection", rawURL)
		}
	}
}

func TestResolveAdminStripeInvoicePDFURLExtractsPDFLink(t *testing.T) {
	originalClient := adminInvoiceHTTPClient
	t.Cleanup(func() { adminInvoiceHTTPClient = originalClient })
	adminInvoiceHTTPClient = &http.Client{
		Transport: adminInvoiceRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://invoice.stripe.com/i/acct_123/invoice_456" {
				t.Fatalf("request URL = %q", req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(
					`<a href="https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=one\u0026x=two">PDF</a>`,
				)),
				Header: make(http.Header),
			}, nil
		}),
		Timeout: adminInvoicePDFFetchTimeout,
	}

	got, err := resolveAdminStripeInvoicePDFURL(context.Background(), "https://invoice.stripe.com/i/acct_123/invoice_456")
	if err != nil {
		t.Fatalf("resolveAdminStripeInvoicePDFURL() error = %v", err)
	}
	if want := "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=one&x=two"; got != want {
		t.Fatalf("resolveAdminStripeInvoicePDFURL() = %q, want %q", got, want)
	}
}
