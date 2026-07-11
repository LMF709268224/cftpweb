package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetStageOrderStatus(w http.ResponseWriter, r *http.Request) {
	stageULID := strings.TrimSpace(chi.URLParam(r, "stage_ulid"))
	if !requireRequestField(w, stageULID, "stage_ulid") {
		return
	}

	resp, err := h.Mall.GetStageOrderStatus(r.Context(), &mallpb.GetStageOrderStatusRequest{
		StageOrderUlid: stageULID, // this was StageUlid, but the field is actually StageOrderUlid according to the pb definition... wait, the route parameter was stage_ulid. So we map it to StageOrderUlid for now.
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListStageOrders(w http.ResponseWriter, r *http.Request) {
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	pipelineCCULID := strings.TrimSpace(r.URL.Query().Get("pipeline_cc_ulid"))
	stageCCULID := strings.TrimSpace(r.URL.Query().Get("stage_cc_ulid"))
	if stageCCULID == "" {
		stageCCULID = strings.TrimSpace(r.URL.Query().Get("stage_ulid"))
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	if !requireRequestFields(w, pipelineCCULID, "pipeline_cc_ulid", stageCCULID, "stage_cc_ulid") {
		return
	}

	req := &mallpb.ListStageOrdersRequest{
		Filters: &mallpb.StageOrderFilters{
			CandidateUlid:  candidateULID,
			PipelineCcUlid: pipelineCCULID,
			StageCcUlid:    stageCCULID,
			OrderStatus:    status,
		},
		Cursor:   strings.TrimSpace(r.URL.Query().Get("cursor")),
		PageSize: parseCursorPage(r, 20).PageSize,
	}

	resp, err := h.Mall.ListStageOrders(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	query := adminOrderListQuery{
		CandidateULID: strings.TrimSpace(r.URL.Query().Get("candidate_ulid")),
		BizType:       strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("biz_type"))),
		BizRefULID:    strings.TrimSpace(r.URL.Query().Get("biz_ref_ulid")),
		OrderStatus:   strings.TrimSpace(r.URL.Query().Get("order_status")),
		PaymentStatus: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("payment_status"))),
	}
	page := parseCursorPage(r, 20)

	req := &mallpb.ListOrdersRequest{
		Filters: &mallpb.OrderFilters{
			CandidateUlid: query.CandidateULID,
			BizType:       query.BizType,
			BizRefUlid:    query.BizRefULID,
			OrderStatus:   query.OrderStatus,
			PaymentStatus: query.PaymentStatus,
		},
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
		SortOrder: mallpb.SortOrder(page.Sort),
	}

	resp, err := h.Mall.ListOrders(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Mall.GetOrderCount(ctx, &mallpb.GetOrderCountRequest{
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

	items := make([]adminOrderSummary, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}
		items = append(items, adminOrderSummary{
			OrderULID:     item.GetOrderUlid(),
			ProductName:   item.GetMeta().GetProductName(),
			CandidateULID: item.GetCandidateUlid(),
			CandidateName: h.candidateName(item.GetCandidateUlid()),
			BizType:       item.GetBizType(),
			BizRefULID:    item.GetBizRefUlid(),
			AmountMinor:   item.GetAmountMinor(),
			CurrencyCode:  strings.ToUpper(item.GetCurrencyCode()),
			OrderStatus:   strings.ToUpper(item.GetOrderStatus()),
			PaymentStatus: deriveAdminPaymentStatus(item.GetOrderStatus(), item.GetPaymentStatus()),
			CreatedAt:     item.GetCreatedAt(),
		})
	}

	WriteJSON(w, http.StatusOK, adminOrderListResponse{
		Items:      items,
		Total:      int32(total.Total),
		TotalLabel: total.Label(),
		TotalExact: total.Exact,
		NextCursor: resp.GetNextCursor(),
		HasMore:    resp.GetHasMore(),
	})
}

type adminOrderListQuery struct {
	CandidateULID string
	BizType       string
	BizRefULID    string
	OrderStatus   string
	PaymentStatus string
}

type adminOrderListResponse struct {
	Items      []adminOrderSummary `json:"items"`
	Total      int32               `json:"total"`
	TotalLabel string              `json:"total_label,omitempty"`
	TotalExact bool                `json:"total_exact"`
	NextCursor string              `json:"next_cursor,omitempty"`
	HasMore    bool                `json:"has_more"`
}

type adminOrderSummary struct {
	OrderULID     string `json:"order_ulid"`
	ProductName   string `json:"product_name,omitempty"`
	CandidateULID string `json:"candidate_ulid"`
	CandidateName string `json:"candidate_name,omitempty"`
	BizType       string `json:"biz_type"`
	BizRefULID    string `json:"biz_ref_ulid"`
	AmountMinor   int64  `json:"amount_minor"`
	CurrencyCode  string `json:"currency_code"`
	OrderStatus   string `json:"order_status"`
	PaymentStatus string `json:"payment_status"`
	CreatedAt     string `json:"created_at"`
}

func deriveAdminPaymentStatus(orderStatus, paymentStatus string) string {
	status := strings.ToUpper(strings.TrimSpace(paymentStatus))
	if status != "" && status != "UNSPECIFIED" {
		return status
	}

	status = strings.ToUpper(strings.TrimSpace(orderStatus))
	switch {
	case strings.Contains(status, "FAILED"):
		return "FAILED"
	case strings.Contains(status, "CANCELLED"):
		return "CANCELLED"
	case strings.Contains(status, "EXPIRED"):
		return "EXPIRED"
	case strings.Contains(status, "WAIT") || strings.Contains(status, "PENDING"):
		return "WAIT_PAY"
	case strings.Contains(status, "PAID") || strings.Contains(status, "COMPLETED"):
		return "PAID"
	default:
		return ""
	}
}

// AdminSyncOrderMeta POST /api/mall/orders/sync-meta
func (h *Handler) AdminSyncOrderMeta(w http.ResponseWriter, r *http.Request) {
	var req mallpb.AdminSyncOrderMetaRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
			return
		}
	}

	resp, err := h.Mall.AdminSyncOrderMeta(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
