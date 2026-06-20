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

	gpaypb "github.com/LMF709268224/cftpproto/gpay"
	"github.com/go-chi/chi/v5"
)

const invoicePDFFetchTimeout = 30 * time.Second

var stripeInvoicePDFPattern = regexp.MustCompile(`https://invoice\.stripe\.com/i/[A-Za-z0-9_/-]+/pdf(?:\?[^"' <]*)?`)
var stripeRelativeInvoicePDFPattern = regexp.MustCompile(`/i/[A-Za-z0-9_/-]+/pdf(?:\?[^"' <]*)?`)

// QueryInvoice  GET /api/invoices/{orderId}
func (h *Handler) QueryInvoice(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderId")

	resp, err := h.Gpay.GetInvoice(r.Context(), &gpaypb.GetInvoiceRequest{
		Lookup: &gpaypb.GetInvoiceRequest_OrderId{
			OrderId: orderID,
		},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, QueryInvoiceRsp{
		InvoiceNumber: resp.InvoiceNumber,
		Status:        resp.Status,
		SubTotal:      float64(resp.Subtotal) / 100.0,
		TotalTax:      float64(resp.Tax) / 100.0,
		Total:         float64(resp.Total) / 100.0,
		Currency:      resp.Currency,
		InvoiceUrl:    resp.HostedInvoiceUrl,
	})
}

// DownloadPdf  GET /api/invoices/{orderId}/pdf
func (h *Handler) DownloadPdf(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "orderId")
	if strings.TrimSpace(orderID) == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "order_id is required")
		return
	}

	resp, err := h.Gpay.GetInvoice(r.Context(), &gpaypb.GetInvoiceRequest{
		Lookup: &gpaypb.GetInvoiceRequest_OrderId{
			OrderId: orderID,
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

	ctx, cancel := context.WithTimeout(r.Context(), invoicePDFFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pdfURL, nil)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create invoice PDF request")
		return
	}

	pdfResp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("Failed to fetch invoice PDF", "error", err, "order_id", orderID)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "failed to fetch invoice PDF")
		return
	}
	defer pdfResp.Body.Close()

	if pdfResp.StatusCode < 200 || pdfResp.StatusCode >= 300 {
		slog.Error("Invoice PDF endpoint returned non-2xx", "status", pdfResp.StatusCode, "order_id", orderID)
		WriteError(w, http.StatusServiceUnavailable, ErrServiceUnavailable, "invoice PDF is not available")
		return
	}

	filename := sanitizeFilename(resp.GetInvoiceNumber())
	if filename == "" {
		filename = sanitizeFilename(orderID)
	}
	if filename == "" {
		filename = "invoice"
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.pdf"`, filename))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := io.Copy(w, pdfResp.Body); err != nil {
		slog.Error("Failed to write invoice PDF response", "error", err, "order_id", orderID)
	}
}

func resolveStripeInvoicePDFURL(ctx context.Context, hostedInvoiceURL string) (string, error) {
	hostedInvoiceURL = strings.TrimSpace(hostedInvoiceURL)
	if hostedInvoiceURL == "" {
		return "", fmt.Errorf("hosted invoice url is empty")
	}
	if strings.Contains(hostedInvoiceURL, "/pdf") {
		return hostedInvoiceURL, nil
	}

	ctx, cancel := context.WithTimeout(ctx, invoicePDFFetchTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, hostedInvoiceURL, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
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

	htmlText := html.UnescapeString(string(htmlBody))
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
	return strings.ReplaceAll(match, `\u0026`, "&"), nil
}
