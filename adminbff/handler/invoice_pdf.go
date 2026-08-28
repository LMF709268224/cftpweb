package handler

import (
	"context"
	"fmt"
	"html"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
)

const adminInvoicePDFFetchTimeout = 30 * time.Second

var adminInvoiceHTTPClient = &http.Client{
	Timeout: adminInvoicePDFFetchTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many invoice redirects")
		}
		_, err := validateAdminStripeHostedInvoiceURL(req.URL.String())
		return err
	},
}

var adminStripeInvoicePDFPattern = regexp.MustCompile(`https://(?:pay\.stripe\.com/invoice|invoice\.stripe\.com/i)/[A-Za-z0-9_/-]+/pdf(?:\?[^"' <]*)?`)
var adminStripeRelativeInvoicePDFPattern = regexp.MustCompile(`/(?:invoice|i)/[A-Za-z0-9_/-]+/pdf(?:\?[^"' <]*)?`)

type InvoicePDFResponse struct {
	PDFURL string `json:"pdf_url"`
}

// GetInvoicePDF resolves the signed Stripe PDF URL for an admin invoice preview.
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

	pdfURL, err := resolveAdminStripeInvoicePDFURL(r.Context(), resp.GetHostedInvoiceUrl())
	if err != nil {
		slog.Error("Failed to resolve admin invoice PDF URL", "error", err, "order_id", orderULID)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "invoice PDF is not available")
		return
	}

	WriteJSON(w, http.StatusOK, InvoicePDFResponse{PDFURL: pdfURL})
}

func parseAdminStripeURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("Stripe invoice URL is empty")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Stripe invoice URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" {
		return nil, fmt.Errorf("Stripe invoice URL must use plain HTTPS")
	}
	if parsed.RawFragment != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("Stripe invoice URL must not contain a fragment")
	}
	return parsed, nil
}

func validateAdminStripeHostedInvoiceURL(rawURL string) (string, error) {
	parsed, err := parseAdminStripeURL(rawURL)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(parsed.Hostname(), "invoice.stripe.com") ||
		!strings.HasPrefix(parsed.EscapedPath(), "/i/") ||
		strings.HasSuffix(strings.TrimRight(parsed.EscapedPath(), "/"), "/pdf") {
		return "", fmt.Errorf("URL is not a Stripe hosted invoice page")
	}
	return parsed.String(), nil
}

func validateAdminStripeInvoicePDFURL(rawURL string) (string, error) {
	parsed, err := parseAdminStripeURL(rawURL)
	if err != nil {
		return "", err
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	isPDFPath := strings.HasSuffix(path, "/pdf")
	isPayInvoice := strings.EqualFold(parsed.Hostname(), "pay.stripe.com") && strings.HasPrefix(path, "/invoice/")
	isHostedInvoicePDF := strings.EqualFold(parsed.Hostname(), "invoice.stripe.com") && strings.HasPrefix(path, "/i/")
	if !isPDFPath || (!isPayInvoice && !isHostedInvoicePDF) {
		return "", fmt.Errorf("URL is not a Stripe invoice PDF")
	}
	return parsed.String(), nil
}

func resolveAdminStripeInvoicePDFURL(ctx context.Context, hostedInvoiceURL string) (string, error) {
	if pdfURL, err := validateAdminStripeInvoicePDFURL(hostedInvoiceURL); err == nil {
		return pdfURL, nil
	}
	hostedInvoiceURL, err := validateAdminStripeHostedInvoiceURL(hostedInvoiceURL)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, adminInvoicePDFFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hostedInvoiceURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := adminInvoiceHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("hosted invoice returned status %d", resp.StatusCode)
	}
	htmlBody, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return "", err
	}

	htmlText := strings.ReplaceAll(html.UnescapeString(string(htmlBody)), `\u0026`, "&")
	match := adminStripeInvoicePDFPattern.FindString(htmlText)
	if match == "" {
		match = adminStripeRelativeInvoicePDFPattern.FindString(htmlText)
		if match == "" {
			return "", fmt.Errorf("invoice PDF link not found")
		}
		baseURL, err := url.Parse(hostedInvoiceURL)
		if err != nil {
			return "", err
		}
		relativeURL, err := url.Parse(match)
		if err != nil {
			return "", err
		}
		match = baseURL.ResolveReference(relativeURL).String()
	}
	return validateAdminStripeInvoicePDFURL(match)
}
