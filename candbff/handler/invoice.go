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

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	"github.com/go-chi/chi/v5"
)

const invoicePDFFetchTimeout = 30 * time.Second

// invoiceHTTPClient is a dedicated client for fetching Stripe invoice pages/PDFs.
// Using a named client (not http.DefaultClient) ensures connection and TLS timeouts are applied.
var invoiceHTTPClient = &http.Client{
	Timeout: invoicePDFFetchTimeout,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 {
			return fmt.Errorf("too many invoice redirects")
		}
		_, err := validateStripeHostedInvoiceURL(req.URL.String())
		return err
	},
}

var stripeInvoicePDFPattern = regexp.MustCompile(`https://(?:pay\.stripe\.com/invoice|invoice\.stripe\.com/i)/[A-Za-z0-9_/-]+/pdf(?:\?[^"' <]*)?`)
var stripeRelativeInvoicePDFPattern = regexp.MustCompile(`/(?:invoice|i)/[A-Za-z0-9_/-]+/pdf(?:\?[^"' <]*)?`)

func (h *Handler) verifyInvoiceableOrder(ctx context.Context, candidateID, orderID string) error {
	const limit uint32 = 50
	cursor := ""
	for {
		resp, err := h.Mall.ListOrders(ctx, &mallpb.ListOrdersRequest{
			Filters: &mallpb.OrderFilters{
				CandidateUlid: candidateID,
			},
			Cursor:   cursor,
			PageSize: limit,
		})
		if err != nil {
			return err
		}
		for _, item := range resp.GetItems() {
			if item.GetOrderUlid() == orderID {
				if !isOrderCompleted(item.GetOrderStatus()) {
					return NewError(http.StatusConflict, ErrPrecondition, "invoice is only available for completed orders")
				}
				return nil
			}
		}
		if !resp.GetHasMore() || resp.GetNextCursor() == "" {
			break
		}
		cursor = resp.GetNextCursor()
	}
	return NewError(http.StatusForbidden, ErrForbidden, "access denied")
}

func writeInvoiceOrderVerificationError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*AppError); ok {
		HandleAppError(w, appErr)
		return
	}
	HandleGrpcError(w, err)
}

// QueryInvoice GET /api/invoices/{orderId}
func (h *Handler) QueryInvoice(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderId")

	if err := h.verifyInvoiceableOrder(r.Context(), CandidateID(r), orderID); err != nil {
		writeInvoiceOrderVerificationError(w, err)
		return
	}

	resp, err := h.Gpay.GetInvoice(r.Context(), &gpaypb.GetInvoiceRequest{
		Lookup: &gpaypb.GetInvoiceRequest_OrderUlid{
			OrderUlid: orderID,
		},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	invoiceURL, err := validateStripeInvoiceURL(resp.GetHostedInvoiceUrl())
	if err != nil {
		slog.Error("Invalid Stripe hosted invoice URL", "error", err, "order_id", orderID)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "invoice is not available")
		return
	}

	WriteJSON(w, http.StatusOK, QueryInvoiceRsp{
		InvoiceNumber: resp.InvoiceNumber,
		Status:        resp.Status,
		SubTotal:      float64(resp.Subtotal) / 100.0,
		TotalTax:      float64(resp.Tax) / 100.0,
		Total:         float64(resp.Total) / 100.0,
		Currency:      resp.Currency,
		InvoiceUrl:    invoiceURL,
	})
}

// DownloadPdf GET /api/invoices/{orderId}/pdf
func (h *Handler) DownloadPdf(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderId")
	if strings.TrimSpace(orderID) == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "order_id is required")
		return
	}

	if err := h.verifyInvoiceableOrder(r.Context(), CandidateID(r), orderID); err != nil {
		writeInvoiceOrderVerificationError(w, err)
		return
	}

	resp, err := h.Gpay.GetInvoice(r.Context(), &gpaypb.GetInvoiceRequest{
		Lookup: &gpaypb.GetInvoiceRequest_OrderUlid{
			OrderUlid: orderID,
		},
	})
	if err != nil {
		slog.Error("Failed to GetInvoice for PDF download", "error", err, "order_id", orderID)
		HandleGrpcError(w, err)
		return
	}

	pdfURL, err := resolveStripeInvoicePDFURL(r.Context(), resp.GetHostedInvoiceUrl())
	if err != nil {
		slog.Error("Failed to resolve invoice PDF URL", "error", err, "order_id", orderID)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "invoice PDF is not available")
		return
	}

	http.Redirect(w, r, pdfURL, http.StatusTemporaryRedirect)
}

func parseStripeURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, fmt.Errorf("Stripe invoice url is empty")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Stripe invoice url: %w", err)
	}
	if parsed.Scheme != "https" || parsed.User != nil || parsed.Port() != "" {
		return nil, fmt.Errorf("Stripe invoice url must use plain HTTPS")
	}
	if parsed.RawFragment != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("Stripe invoice url must not contain a fragment")
	}
	return parsed, nil
}

func validateStripeHostedInvoiceURL(rawURL string) (string, error) {
	parsed, err := parseStripeURL(rawURL)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(parsed.Hostname(), "invoice.stripe.com") ||
		!strings.HasPrefix(parsed.EscapedPath(), "/i/") ||
		strings.HasSuffix(strings.TrimRight(parsed.EscapedPath(), "/"), "/pdf") {
		return "", fmt.Errorf("url is not a Stripe hosted invoice page")
	}
	return parsed.String(), nil
}

func validateStripeInvoicePDFURL(rawURL string) (string, error) {
	parsed, err := parseStripeURL(rawURL)
	if err != nil {
		return "", err
	}
	path := strings.TrimRight(parsed.EscapedPath(), "/")
	isPDFPath := strings.HasSuffix(path, "/pdf")
	isPayInvoice := strings.EqualFold(parsed.Hostname(), "pay.stripe.com") && strings.HasPrefix(path, "/invoice/")
	isHostedInvoicePDF := strings.EqualFold(parsed.Hostname(), "invoice.stripe.com") && strings.HasPrefix(path, "/i/")
	if !isPDFPath || (!isPayInvoice && !isHostedInvoicePDF) {
		return "", fmt.Errorf("url is not a Stripe invoice PDF")
	}
	return parsed.String(), nil
}

func validateStripeInvoiceURL(rawURL string) (string, error) {
	if hostedURL, err := validateStripeHostedInvoiceURL(rawURL); err == nil {
		return hostedURL, nil
	}
	return validateStripeInvoicePDFURL(rawURL)
}

func resolveStripeInvoicePDFURL(ctx context.Context, hostedInvoiceURL string) (string, error) {
	if pdfURL, err := validateStripeInvoicePDFURL(hostedInvoiceURL); err == nil {
		return pdfURL, nil
	}
	hostedInvoiceURL, err := validateStripeHostedInvoiceURL(hostedInvoiceURL)
	if err != nil {
		return "", err
	}

	ctx, cancel := context.WithTimeout(ctx, invoicePDFFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hostedInvoiceURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := invoiceHTTPClient.Do(req)
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
	match := stripeInvoicePDFPattern.FindString(htmlText)
	if match == "" {
		match = stripeRelativeInvoicePDFPattern.FindString(htmlText)
		if match == "" {
			return "", fmt.Errorf("invoice pdf link not found")
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
	return validateStripeInvoicePDFURL(match)
}
