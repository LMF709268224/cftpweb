package handler

import (
	"context"
	"net/http"
	"time"

	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
)

type InvoiceItem struct {
	ID            string  `json:"id"`
	OrderUlid     string  `json:"order_id"`
	Email         string  `json:"email"`
	CandidateName string  `json:"candidate_name,omitempty"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"created_at"`
	PaidAt        string  `json:"paid_at,omitempty"`
}

type ListInvoicesRsp struct {
	Total      uint32        `json:"total"`
	TotalLabel string        `json:"total_label,omitempty"`
	TotalExact bool          `json:"total_exact"`
	Invoices   []InvoiceItem `json:"invoices"`
	NextCursor string        `json:"next_cursor,omitempty"`
	HasMore    bool          `json:"has_more"`
}

// ListInvoices GET /api/invoices
func (h *Handler) ListInvoices(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)

	req := &gpaypb.ListInvoicesRequest{
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: gpaypb.SortOrder(page.Sort),
	}

	resp, err := h.Gpay.ListInvoices(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Gpay.GetInvoiceCount(ctx, &gpaypb.GetInvoiceCountRequest{
			Filters: req.GetFilters(),
			Limit:   limit,
			Cursor:  cursor,
		})
		if err != nil {
			return 0, "", err
		}
		return resp.GetCount(), resp.GetNextCursor(), nil
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	rsp := ListInvoicesRsp{
		Total:      total.Total,
		TotalLabel: total.Label(),
		TotalExact: total.Exact,
		Invoices:   make([]InvoiceItem, 0, len(resp.GetInvoices())),
		NextCursor: resp.GetNextCursor(),
		HasMore:    resp.GetHasMore(),
	}

	for _, inv := range resp.GetInvoices() {
		item := InvoiceItem{
			ID:            inv.GetStripeInvoiceId(),
			OrderUlid:     inv.GetOrderUlid(),
			Email:         inv.GetCustomerUlid(),
			CandidateName: h.candidateName(inv.GetCustomerUlid()),
			Amount:        float64(inv.GetAmount()) / 100.0,
			Currency:      inv.GetCurrency(),
			Status:        inv.GetStatus().String(),
			CreatedAt:     time.Unix(inv.GetCreatedAt(), 0).Format(time.RFC3339),
		}
		if inv.GetPaidAt() > 0 {
			item.PaidAt = time.Unix(inv.GetPaidAt(), 0).Format(time.RFC3339)
		}
		rsp.Invoices = append(rsp.Invoices, item)
	}

	WriteJSON(w, http.StatusOK, rsp)
}
