package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
)

// GetOrder GET /api/orders/{orderId}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// Reserved for an order detail page. The order list now reads directly from gmall.
}

// ListOrders GET /api/orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	page := parsePositiveIntQuery(r, "page", 1)
	pageSize := parsePositiveIntQuery(r, "page_size", parsePositiveIntQuery(r, "limit", 10))
	if pageSize > 50 {
		pageSize = 50
	}
	offset := (page - 1) * pageSize

	mallResp, err := h.Mall.ListPipelineOrders(r.Context(), &mallpb.ListPipelineOrdersRequest{
		CandidateUlid: candidateID,
		Limit:         int32(pageSize),
		Offset:        int32(offset),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	orders := make([]OrderItem, 0, len(mallResp.GetItems()))
	completed := 0
	totalAmount := 0.0
	pipelineNames := make(map[string]string)

	for _, item := range mallResp.GetItems() {
		pipelineUlid := item.GetPipelineCcUlid()
		name := pipelineNames[pipelineUlid]
		if name == "" && pipelineUlid != "" {
			gccResp, err := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
				Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineUlid},
			})
			if err == nil {
				name = gccResp.GetName()
				pipelineNames[pipelineUlid] = name
			} else {
				name = pipelineUlid
			}
		}

		status := "pending"
		orderStatus := strings.ToUpper(item.GetOrderStatus())
		if strings.Contains(orderStatus, "COMPLETED") || strings.Contains(orderStatus, "SUCCESS") {
			status = "completed"
			completed++
		} else if strings.Contains(orderStatus, "CANCEL") || strings.Contains(orderStatus, "FAILED") {
			status = "cancelled"
		} else {
			status = "processing"
		}

		amount := 0.0
		currency := "USD"
		payOrderUlid := strings.TrimSpace(item.GetPipelinePayOrderUlid())
		actualOrderUlid := item.GetPipelineOrderUlid()
		canViewInvoice := payOrderUlid != ""

		if payOrderUlid != "" {
			orderResp, err := h.Mall.GetOrderSummary(r.Context(), &mallpb.GetOrderSummaryRequest{
				OrderUlid: payOrderUlid,
			})
			if err == nil && orderResp != nil && orderResp.GetFound() && orderResp.GetSummary() != nil {
				realOrder := orderResp.GetSummary()
				amount = float64(realOrder.GetAmountMinor()) / 100.0
				if strings.TrimSpace(realOrder.GetCurrencyCode()) != "" {
					currency = strings.ToUpper(strings.TrimSpace(realOrder.GetCurrencyCode()))
				}
				actualOrderUlid = realOrder.GetOrderUlid()
			}
		}
		if amount <= 0 {
			previewResp, err := h.Mall.PreviewPayment(r.Context(), &mallpb.PreviewPaymentRequest{
				BizType:    "PIPELINE_PAYMENT",
				BizRefUlid: item.GetPipelineOrderUlid(),
			})
			if err == nil && previewResp != nil {
				amount = float64(previewResp.GetTotal()) / 100.0
				if strings.TrimSpace(previewResp.GetCurrency()) != "" {
					currency = strings.ToUpper(strings.TrimSpace(previewResp.GetCurrency()))
				}
			}
		}

		if status == "completed" {
			totalAmount += amount
		}

		createdAt := item.GetCreatedAt()
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			createdAt = t.Format("2006-01-02 15:04")
		}

		orders = append(orders, OrderItem{
			OrderID:              actualOrderUlid,
			ProductName:          name,
			Status:               status,
			RawStatus:            item.GetOrderStatus(),
			PipelineID:           pipelineUlid,
			CreatedAt:            createdAt,
			PaymentMethod:        item.GetPaymentMode(),
			Amount:               amount,
			Currency:             currency,
			PipelinePayOrderUlid: payOrderUlid,
			CanViewInvoice:       canViewInvoice,
		})
	}

	totalOrders := int(mallResp.GetTotal())
	if totalOrders <= 0 {
		totalOrders = len(orders)
	}
	totalPages := 0
	if totalOrders > 0 {
		totalPages = (totalOrders + pageSize - 1) / pageSize
	}

	WriteJSON(w, http.StatusOK, OrderListRsp{
		TotalOrders: totalOrders,
		Completed:   completed,
		TotalAmount: totalAmount,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		Orders:      orders,
	})
}

func parsePositiveIntQuery(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
