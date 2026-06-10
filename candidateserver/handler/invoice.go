package handler

import (
	"net/http"
)

// QueryInvoice  GET /api/invoices/{orderId}
func (h *Handler) QueryInvoice(w http.ResponseWriter, r *http.Request) {
	// candidateID := CandidateID(r)

	//TODO
	// resp, err := h.Invoice.QueryInvoice(r.Context(), &invoicepb.QueryInvoiceRequest{
	// 	OrderId: orderID,
	// })
	// if err != nil {
	// 	HandleGrpcError(w, err)
	// 	return
	// }

	WriteJSON(w, http.StatusOK, QueryInvoiceRsp{
		// InvoiceID:     resp.InvoiceId,
		// PaymentID:     resp.PaymentId,
		// RequestStatus: resp.RequestStatus,
		// SubTotal:      resp.SubTotal,
		// TotalTax:      resp.TotalTax,
		// Total:         resp.Total,
		// ErrorMsg:      resp.ErrorMsg,
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
