package handler

import (
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
	stageULID := strings.TrimSpace(r.URL.Query().Get("stage_ulid"))
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	req := &mallpb.ListStageOrdersRequest{
		CandidateUlid: candidateULID,
		StageCcUlid:   stageULID, // The proto uses stage_cc_ulid
		OrderStatus:   status,    // The proto uses order_status string
		Limit:         int32(parseUint32Query(r, "limit")),
		Offset:        int32(parseUint32Query(r, "offset")),
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
		Limit:         int32Query(r, "limit", 20),
		Offset:        int32Query(r, "offset", 0),
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}

	req := &mallpb.ListOrdersRequest{
		CandidateUlid: query.CandidateULID,
		BizType:       query.BizType,
		BizRefUlid:    query.BizRefULID,
		OrderStatus:   query.OrderStatus,
		PaymentStatus: query.PaymentStatus,
		Limit:         query.Limit,
		Offset:        query.Offset,
	}

	resp, err := h.Mall.ListOrders(r.Context(), req)
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
			PaymentStatus: deriveAdminPaymentStatus(item.GetOrderStatus(), item.GetPaymentStatus(), item.GetOrderUlid()),
			CreatedAt:     item.GetCreatedAt(),
			PayOrderULID:  item.GetOrderUlid(),
		})
	}

	WriteJSON(w, http.StatusOK, adminOrderListResponse{Items: items, Total: resp.GetTotal()})
}

type adminOrderListQuery struct {
	CandidateULID string
	BizType       string
	BizRefULID    string
	OrderStatus   string
	PaymentStatus string
	Limit         int32
	Offset        int32
}

type adminOrderListResponse struct {
	Items []adminOrderSummary `json:"items"`
	Total int32               `json:"total"`
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
	PayOrderULID  string `json:"pay_order_ulid,omitempty"`
}

func deriveAdminPaymentStatus(orderStatus, paymentStatus, payOrderULID string) string {
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
	case strings.TrimSpace(payOrderULID) != "":
		return "WAIT_PAY"
	default:
		return ""
	}
}
