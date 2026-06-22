package handler

import (
	"net/http"
	"strconv"
	"time"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
)

type InvoiceItem struct {
	ID        string  `json:"id"`
	OrderUlid string  `json:"order_id"`
	Email     string  `json:"email"` // If we fetch user info, else just customer id
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	Status    string  `json:"status"`
	CreatedAt string  `json:"created_at"`
	PaidAt    string  `json:"paid_at,omitempty"`
}

type ListInvoicesRsp struct {
	Total    uint32        `json:"total"`
	Invoices []InvoiceItem `json:"invoices"`
}

// ListInvoices GET /api/invoices
func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	req := &gpaypb.ListInvoicesRequest{
		Page:     uint32(page),
		PageSize: uint32(pageSize),
	}

	resp, err := h.Gpay.ListInvoices(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	rsp := ListInvoicesRsp{
		Total:    resp.GetTotal(),
		Invoices: make([]InvoiceItem, 0, len(resp.GetInvoices())),
	}

	for _, inv := range resp.GetInvoices() {
		item := InvoiceItem{
			ID:        inv.GetStripeInvoiceId(),
			OrderUlid: inv.GetOrderUlid(),
			Email:     inv.GetCustomerUlid(), // using customer_id as email placeholder for now
			Amount:    float64(inv.GetAmount()) / 100.0,
			Currency:  inv.GetCurrency(),
			Status:    inv.GetStatus().String(),
			CreatedAt: time.Unix(inv.GetCreatedAt(), 0).Format(time.RFC3339),
		}
		if inv.GetPaidAt() > 0 {
			item.PaidAt = time.Unix(inv.GetPaidAt(), 0).Format(time.RFC3339)
		}
		rsp.Invoices = append(rsp.Invoices, item)
	}

	WriteJSON(w, http.StatusOK, rsp)
}
