package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestGetInvoicePDFProxiesInlinePDF(t *testing.T) {
	client := &invoicePDFClientStub{response: &gpaypb.GetInvoiceResponse{
		HostedInvoiceUrl: "https://invoice.stripe.com/i/acct_123/invoice_456?s=ap",
	}}
	originalClient := adminInvoiceHTTPClient
	t.Cleanup(func() { adminInvoiceHTTPClient = originalClient })
	pdfBody := "%PDF-1.7\ntest invoice"
	requestedPDFStorage := false
	adminInvoiceHTTPClient = &http.Client{
		Transport: adminInvoiceRoundTripper(func(req *http.Request) (*http.Response, error) {
			switch req.URL.String() {
			case "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=ap":
				return &http.Response{
					StatusCode: http.StatusSeeOther,
					Body:       http.NoBody,
					Header: http.Header{
						"Location": []string{"https://stripe-upload-api.s3.us-west-1.amazonaws.com/file-api/prod/invoice.pdf"},
					},
				}, nil
			case "https://stripe-upload-api.s3.us-west-1.amazonaws.com/file-api/prod/invoice.pdf":
				requestedPDFStorage = true
				return &http.Response{
					StatusCode:    http.StatusOK,
					Body:          io.NopCloser(strings.NewReader(pdfBody)),
					Header:        http.Header{"Content-Type": []string{"application/pdf"}},
					ContentLength: int64(len(pdfBody)),
				}, nil
			default:
				t.Fatalf("unexpected request URL = %q", req.URL.String())
				return nil, nil
			}
		}),
		Timeout: adminInvoicePDFFetchTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return validateAdminInvoicePDFRedirectURL(req.URL)
		},
	}

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
	if !requestedPDFStorage {
		t.Fatal("Stripe PDF storage URL was not requested")
	}
	if recorder.Header().Get("Content-Type") != "application/pdf" {
		t.Fatalf("Content-Type = %q", recorder.Header().Get("Content-Type"))
	}
	if recorder.Header().Get("Content-Disposition") != `inline; filename="invoice.pdf"` {
		t.Fatalf("Content-Disposition = %q", recorder.Header().Get("Content-Disposition"))
	}
	if !strings.HasPrefix(recorder.Body.String(), "%PDF-") {
		t.Fatalf("body is not a PDF: %q", recorder.Body.String())
	}
}

func TestResolveAdminStripeInvoicePDFURL(t *testing.T) {
	got, err := resolveAdminStripeInvoicePDFURL("https://invoice.stripe.com/i/acct_123/invoice_456?s=ap")
	if err != nil {
		t.Fatalf("resolveAdminStripeInvoicePDFURL() error = %v", err)
	}
	if want := "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=ap"; got != want {
		t.Fatalf("resolveAdminStripeInvoicePDFURL() = %q, want %q", got, want)
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

func TestValidateAdminInvoicePDFRedirectURL(t *testing.T) {
	trusted, err := url.Parse("https://stripe-upload-api.s3.us-west-1.amazonaws.com/file-api/prod/file")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAdminInvoicePDFRedirectURL(trusted); err != nil {
		t.Fatalf("trusted redirect rejected: %v", err)
	}
	untrusted, err := url.Parse("https://example.com/internal")
	if err != nil {
		t.Fatal(err)
	}
	if err := validateAdminInvoicePDFRedirectURL(untrusted); err == nil {
		t.Fatal("untrusted redirect accepted")
	}
}

func TestFetchAdminInvoicePDFRejectsNonPDF(t *testing.T) {
	originalClient := adminInvoiceHTTPClient
	t.Cleanup(func() { adminInvoiceHTTPClient = originalClient })
	adminInvoiceHTTPClient = &http.Client{
		Transport: adminInvoiceRoundTripper(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("<html>not a PDF</html>")),
				Header:     make(http.Header),
			}, nil
		}),
	}
	if _, err := fetchAdminInvoicePDF(context.Background(), "https://pay.stripe.com/invoice/acct_123/invoice_456/pdf?s=ap"); err == nil {
		t.Fatal("fetchAdminInvoicePDF() error = nil, want non-PDF rejection")
	}
}
