package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
)

const (
	adminInvoicePDFFetchTimeout = 30 * time.Second
	adminInvoicePDFMaxBytes     = 20 << 20
)

var adminStripePDFStorageHostPattern = regexp.MustCompile(`^stripe-upload-api\.s3(?:[.-][a-z0-9-]+)?\.amazonaws\.com$`)

var adminInvoiceHTTPClient = &http.Client{
	Timeout: adminInvoicePDFFetchTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return fmt.Errorf("too many invoice PDF redirects")
		}
		return validateAdminInvoicePDFRedirectURL(req.URL)
	},
}

// GetInvoicePDF proxies the Stripe PDF so browsers can render it inline.
func (h *Handler) GetInvoicePDF(w http.ResponseWriter, r *http.Request) {
	orderULID, ok := requiredURLParam(w, r, "order_ulid")
	if !ok {
		return
	}

	resp, err := h.Gpay.GetInvoice(r.Context(), &gpaypb.GetInvoiceRequest{
		Lookup: &gpaypb.GetInvoiceRequest_OrderUlid{OrderUlid: orderULID},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	pdfURL, err := resolveAdminStripeInvoicePDFURL(resp.GetHostedInvoiceUrl())
	if err != nil {
		slog.Error("Failed to resolve admin invoice PDF URL", "error", err, "order_id", orderULID)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "invoice PDF is not available")
		return
	}
	pdf, err := fetchAdminInvoicePDF(r.Context(), pdfURL)
	if err != nil {
		slog.Error("Failed to fetch admin invoice PDF", "error", err, "order_id", orderULID)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "invoice PDF is not available")
		return
	}

	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Content-Disposition", `inline; filename="invoice.pdf"`)
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(pdf); err != nil {
		slog.Error("Failed to write admin invoice PDF", "error", err, "order_id", orderULID)
	}
}

func parseAdminHTTPSURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("URL is empty")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" {
		return nil, fmt.Errorf("URL must use plain HTTPS")
	}
	if parsed.RawFragment != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("URL must not contain a fragment")
	}
	return parsed, nil
}

func validateAdminStripeHostedInvoiceURL(rawURL string) (*url.URL, error) {
	parsed, err := parseAdminHTTPSURL(rawURL)
	if err != nil {
		return nil, err
	}
	segments := adminInvoicePathSegments(parsed.EscapedPath())
	if !strings.EqualFold(parsed.Hostname(), "invoice.stripe.com") ||
		len(segments) != 3 || segments[0] != "i" {
		return nil, fmt.Errorf("URL is not a Stripe hosted invoice page")
	}
	return parsed, nil
}

func validateAdminStripeInvoicePDFURL(rawURL string) (string, error) {
	parsed, err := parseAdminHTTPSURL(rawURL)
	if err != nil {
		return "", err
	}
	segments := adminInvoicePathSegments(parsed.EscapedPath())
	isPDFPath := len(segments) == 4 && segments[3] == "pdf"
	isPayInvoice := strings.EqualFold(parsed.Hostname(), "pay.stripe.com") && isPDFPath && segments[0] == "invoice"
	isHostedInvoicePDF := strings.EqualFold(parsed.Hostname(), "invoice.stripe.com") && isPDFPath && segments[0] == "i"
	if !isPayInvoice && !isHostedInvoicePDF {
		return "", fmt.Errorf("URL is not a Stripe invoice PDF")
	}
	return parsed.String(), nil
}

func adminInvoicePathSegments(escapedPath string) []string {
	segments := strings.Split(strings.Trim(escapedPath, "/"), "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, "%2f") || strings.Contains(segment, "%2F") {
			return nil
		}
	}
	return segments
}

func resolveAdminStripeInvoicePDFURL(hostedInvoiceURL string) (string, error) {
	if pdfURL, err := validateAdminStripeInvoicePDFURL(hostedInvoiceURL); err == nil {
		return pdfURL, nil
	}
	hostedURL, err := validateAdminStripeHostedInvoiceURL(hostedInvoiceURL)
	if err != nil {
		return "", err
	}

	invoiceToken := strings.TrimPrefix(strings.TrimRight(hostedURL.Path, "/"), "/i/")
	hostedURL.Host = "pay.stripe.com"
	hostedURL.Path = "/invoice/" + invoiceToken + "/pdf"
	hostedURL.RawPath = ""
	return validateAdminStripeInvoicePDFURL(hostedURL.String())
}

func validateAdminInvoicePDFRedirectURL(redirectURL *url.URL) error {
	parsed, err := parseAdminHTTPSURL(redirectURL.String())
	if err != nil {
		return err
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "pay.stripe.com" || host == "files.stripe.com" || adminStripePDFStorageHostPattern.MatchString(host) {
		return nil
	}
	return fmt.Errorf("untrusted invoice PDF redirect host %q", host)
}

func fetchAdminInvoicePDF(ctx context.Context, pdfURL string) ([]byte, error) {
	pdfURL, err := validateAdminStripeInvoicePDFURL(pdfURL)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, adminInvoicePDFFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/pdf")
	resp, err := adminInvoiceHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("invoice PDF returned status %d", resp.StatusCode)
	}
	if resp.ContentLength > adminInvoicePDFMaxBytes {
		return nil, fmt.Errorf("invoice PDF exceeds %d bytes", adminInvoicePDFMaxBytes)
	}
	pdf, err := io.ReadAll(io.LimitReader(resp.Body, adminInvoicePDFMaxBytes+1))
	if err != nil {
		return nil, err
	}
	if len(pdf) > adminInvoicePDFMaxBytes {
		return nil, fmt.Errorf("invoice PDF exceeds %d bytes", adminInvoicePDFMaxBytes)
	}
	if !bytes.HasPrefix(pdf, []byte("%PDF-")) {
		return nil, fmt.Errorf("invoice response is not a PDF")
	}
	return pdf, nil
}
