package handler

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type invoiceRoundTripper func(*http.Request) (*http.Response, error)

func (f invoiceRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestValidateStripeInvoiceURL(t *testing.T) {
	valid := []string{
		"https://invoice.stripe.com/i/acct_123/test_456",
		"https://pay.stripe.com/invoice/acct_123/test_456/pdf?s=token",
		"https://invoice.stripe.com/i/acct_123/test_456/pdf?s=token",
	}
	for _, rawURL := range valid {
		if _, err := validateStripeInvoiceURL(rawURL); err != nil {
			t.Errorf("validateStripeInvoiceURL(%q) error = %v", rawURL, err)
		}
	}

	invalid := []string{
		"http://invoice.stripe.com/i/acct_123/test_456",
		"https://127.0.0.1/i/acct_123/test_456/pdf",
		"https://invoice.stripe.com.evil.example/i/acct_123/test_456",
		"https://invoice.stripe.com:8443/i/acct_123/test_456",
		"https://user@invoice.stripe.com/i/acct_123/test_456",
		"https://pay.stripe.com/invoice/acct_123/test_456",
		"https://invoice.stripe.com/not-an-invoice",
	}
	for _, rawURL := range invalid {
		if _, err := validateStripeInvoiceURL(rawURL); err == nil {
			t.Errorf("validateStripeInvoiceURL(%q) error = nil, want rejection", rawURL)
		}
	}
}

func TestResolveStripeInvoicePDFURLRejectsUntrustedHost(t *testing.T) {
	if _, err := resolveStripeInvoicePDFURL(context.Background(), "http://127.0.0.1/internal/pdf"); err == nil {
		t.Fatal("resolveStripeInvoicePDFURL() error = nil, want untrusted host rejection")
	}
}

func TestResolveStripeInvoicePDFURLExtractsStripePDF(t *testing.T) {
	originalClient := invoiceHTTPClient
	t.Cleanup(func() {
		invoiceHTTPClient = originalClient
	})
	invoiceHTTPClient = &http.Client{
		Transport: invoiceRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "https://invoice.stripe.com/i/acct_123/test_456" {
				t.Fatalf("request URL = %q, want Stripe hosted invoice URL", req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body: io.NopCloser(strings.NewReader(
					`<a href="https://pay.stripe.com/invoice/acct_123/test_456/pdf?s=one\u0026x=two">PDF</a>`,
				)),
				Request: req,
			}, nil
		}),
		Timeout: invoicePDFFetchTimeout,
	}

	got, err := resolveStripeInvoicePDFURL(context.Background(), "https://invoice.stripe.com/i/acct_123/test_456")
	if err != nil {
		t.Fatalf("resolveStripeInvoicePDFURL() error = %v", err)
	}
	want := "https://pay.stripe.com/invoice/acct_123/test_456/pdf?s=one&x=two"
	if got != want {
		t.Fatalf("resolveStripeInvoicePDFURL() = %q, want %q", got, want)
	}
}
