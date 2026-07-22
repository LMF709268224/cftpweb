package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"candbff/config"
)

func TestValidatePaymentReturnURLsAllowsRequestOrigin(t *testing.T) {
	t.Setenv(config.EnvPaymentReturnAllowedOrigins, "")
	req := httptest.NewRequest("POST", "http://candbff:8080/api/mall/payments/initiate", nil)
	req.Host = "cftpcand.llwan.top"
	req.Header.Set("X-Forwarded-Proto", "https")

	err := validatePaymentReturnURLs(
		req,
		"https://cftpcand.llwan.top/orders?payment_status=success",
		"https://cftpcand.llwan.top/orders?payment_status=cancelled",
	)
	if err != nil {
		t.Fatalf("validatePaymentReturnURLs() error = %v", err)
	}
}

func TestValidatePaymentReturnURLsRejectsExternalOrigin(t *testing.T) {
	t.Setenv(config.EnvPaymentReturnAllowedOrigins, "")
	req := httptest.NewRequest("POST", "https://cftpcand.llwan.top/api/mall/payments/initiate", nil)

	err := validatePaymentReturnURLs(
		req,
		"https://evil.example/payment-complete",
		"https://cftpcand.llwan.top/orders?payment_status=cancelled",
	)
	if err == nil {
		t.Fatal("validatePaymentReturnURLs() error = nil, want external origin rejection")
	}
}

func TestValidatePaymentReturnURLsUsesConfiguredOrigins(t *testing.T) {
	t.Setenv(config.EnvPaymentReturnAllowedOrigins, "http://localhost:8081, https://cftpcand.llwan.top")
	req := httptest.NewRequest("POST", "http://candbff:8080/api/mall/payments/initiate", nil)

	err := validatePaymentReturnURLs(
		req,
		"http://localhost:8081/orders?payment_status=success",
		"http://localhost:8081/orders?payment_status=cancelled",
	)
	if err != nil {
		t.Fatalf("validatePaymentReturnURLs() error = %v", err)
	}
}

func TestValidatePaymentReturnURLsRejectsMalformedURL(t *testing.T) {
	t.Setenv(config.EnvPaymentReturnAllowedOrigins, "https://cftpcand.llwan.top")
	req := httptest.NewRequest("POST", "https://cftpcand.llwan.top/api/mall/payments/initiate", nil)

	tests := []struct {
		name string
		url  string
	}{
		{name: "relative path", url: "/orders"},
		{name: "protocol relative", url: "//evil.example/orders"},
		{name: "unsupported scheme", url: "javascript:alert(1)"},
		{name: "credentials", url: "https://user:password@cftpcand.llwan.top/orders"},
		{name: "lookalike host", url: "https://cftpcand.llwan.top.evil.example/orders"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePaymentReturnURLs(req, tt.url, "https://cftpcand.llwan.top/orders")
			if err == nil {
				t.Fatalf("validatePaymentReturnURLs(%q) error = nil, want rejection", tt.url)
			}
		})
	}
}

func TestInitiatePaymentRejectsExternalReturnURL(t *testing.T) {
	t.Setenv(config.EnvPaymentReturnAllowedOrigins, "")
	body := `{
		"biz_type":"BUNDLE_PURCHASE",
		"biz_ref_ulid":"01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"success_url":"https://evil.example/payment-complete",
		"cancel_url":"https://cftpcand.llwan.top/orders"
	}`
	req := httptest.NewRequest(http.MethodPost, "https://cftpcand.llwan.top/api/mall/payments/initiate", strings.NewReader(body))
	rec := httptest.NewRecorder()

	(&Handler{}).InitiatePayment(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
