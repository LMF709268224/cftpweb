package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
)

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

	// candidateID := CandidateID(r)

	//TODO
	// resp, err := h.Invoice.DownloadPdf(r.Context(), &invoicepb.DownloadPdfRequest{
	// 	OrderId: orderID,
	// })
	// if err != nil {
	// 	HandleGrpcError(w, err)
	// 	return
	// }

	// w.Header().Set("Content-Type", "application/pdf")
	// w.Header().Set("Content-Disposition", "attachment; filename=\""+sanitizeFilename(orderID)+".pdf\"")
	// w.WriteHeader(http.StatusOK)
	// if _, err := w.Write(resp.PdfContent); err != nil {
	// 	slog.Error("Failed to write PDF response", "error", err)
	// }
}
