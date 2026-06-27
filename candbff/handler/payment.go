package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
)

const (
	orderBizPipelinePayment      = "PIPELINE_PAYMENT"
	orderBizStagePayment         = "STAGE_PAYMENT"
	orderBizCourseRetakePayment  = "COURSE_RETAKE_PAYMENT"
	orderBizPipelineUnlock       = "PIPELINE_UNLOCK"
	orderBizCredentialApply      = "CREDENTIAL_APPLICATION"
	orderBizBundlePurchase       = "BUNDLE_PURCHASE"
	defaultCandidateOrderPageMax = 50
)

var candidateOrderBizTypes = []string{
	orderBizPipelinePayment,
	orderBizStagePayment,
	orderBizCourseRetakePayment,
	orderBizPipelineUnlock,
	orderBizCredentialApply,
	orderBizBundlePurchase,
}



// ListOrders GET /api/orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	page := parsePositiveIntQuery(r, "page", 1)
	pageSize := parsePositiveIntQuery(r, "page_size", parsePositiveIntQuery(r, "limit", 10))
	if pageSize > defaultCandidateOrderPageMax {
		pageSize = defaultCandidateOrderPageMax
	}

	bizType := normalizeOrderBizType(r.URL.Query().Get("biz_type"))
	if bizType != "" && !isCandidateOrderBizType(bizType) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "unsupported biz_type")
		return
	}
	orderStatus := normalizeOrderStatusFilter(r)

	var offset int64 = int64(page-1) * int64(pageSize)
	if offset > 2147483647 { // math.MaxInt32
		offset = 2147483647
	}

	req := &mallpb.ListOrdersRequest{
		CandidateUlid: candidateID,
		BizType:       bizType,
		OrderStatus:   orderStatus,
		Limit:         int32(pageSize),
		Offset:        int32(offset),
	}

	resp, err := h.Mall.ListOrders(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	completed := 0
	totalAmount := 0.0
	outOrders := make([]OrderItem, 0, len(resp.GetItems()))

	nameCache := make(map[string]string)
	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}

		rawStatus := candidateOrderRawStatus(item.GetOrderStatus())
		statusStr := candidateOrderStatus(rawStatus)
		amount := float64(item.GetAmountMinor()) / 100.0
		currency := item.GetCurrencyCode()

		if amount == 0 && item.GetBizRefUlid() != "" && item.GetBizType() != "" {
			prevAmt, prevCur := h.previewPayment(r.Context(), item.GetBizType(), item.GetBizRefUlid())
			if prevAmt > 0 {
				amount = prevAmt
				if currency == "" {
					currency = prevCur
				}
			}
		}

		var name string
		if meta := item.GetMeta(); meta != nil && meta.GetProductName() != "" {
			name = meta.GetProductName()
		} else if item.GetBizType() == orderBizPipelinePayment || item.GetBizType() == orderBizPipelineUnlock {
			pName := h.pipelineName(r, item.GetBizRefUlid(), nameCache)
			if pName != "" && pName != item.GetBizRefUlid() {
				name = orderBizTypeLabel(item.GetBizType()) + " - " + pName
			} else {
				name = orderProductName(item.GetBizType(), item.GetBizRefUlid())
			}
		} else {
			name = orderProductName(item.GetBizType(), item.GetBizRefUlid())
		}

		orderItem := OrderItem{
			OrderID:              item.GetOrderUlid(),
			ProductName:          name,
			BizType:              item.GetBizType(),
			BizRefUlid:           item.GetBizRefUlid(),
			Status:               statusStr,
			RawStatus:            rawStatus,
			CreatedAt:            formatOrderCreatedAt(item.GetCreatedAt()),
			Amount:               amount,
			Currency:             currency,
			PayOrderUlid:         item.GetOrderUlid(),
			PipelinePayOrderUlid: item.GetOrderUlid(),
			CanViewInvoice:       statusStr == "completed" && strings.TrimSpace(item.GetOrderUlid()) != "",
		}

		outOrders = append(outOrders, orderItem)

		if statusStr == "completed" {
			completed++
			totalAmount += amount
		}
	}

	totalOrders := int(resp.GetTotal())
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
		Orders:      outOrders,
	})
}

func normalizeOrderBizType(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func normalizeOrderStatusFilter(r *http.Request) string {
	status := strings.TrimSpace(r.URL.Query().Get("order_status"))
	if status == "" {
		status = strings.TrimSpace(r.URL.Query().Get("status"))
	}
	return strings.ToUpper(status)
}

func isCandidateOrderBizType(bizType string) bool {
	for _, allowed := range candidateOrderBizTypes {
		if bizType == allowed {
			return true
		}
	}
	return false
}

func candidateOrderStatus(raw string) string {
	orderStatus := candidateOrderRawStatus(raw)
	switch orderStatus {
	case "COMPLETED", "SUCCESS", "PAID":
		return "completed"
	case "CANCEL", "CANCELLED", "CANCELED", "FAILED":
		return "cancelled"
	case "":
		return "pending"
	default:
		return "processing"
	}
}

func candidateOrderRawStatus(raw string) string {
	orderStatus := strings.ToUpper(strings.TrimSpace(raw))
	orderStatus = strings.NewReplacer("-", "_", " ", "_").Replace(orderStatus)
	switch orderStatus {
	case "0", "UNSPECIFIED", "ORDER_STATUS_UNSPECIFIED":
		return "PENDING"
	case "1", "ORDER_STATUS_PENDING_CREATE":
		return "PENDING_CREATE"
	case "2", "PENDING", "WAIT_PAY", "UNPAID", "ORDER_STATUS_PENDING_PAYMENT":
		return "PENDING_PAYMENT"
	case "3", "SUCCESS", "ORDER_STATUS_COMPLETED":
		return "COMPLETED"
	case "4", "CANCEL", "CANCELED", "ORDER_STATUS_CANCELLED", "ORDER_STATUS_CANCELED":
		return "CANCELLED"
	case "5", "ORDER_STATUS_FAILED":
		return "FAILED"
	default:
		return orderStatus
	}
}

func formatOrderCreatedAt(createdAt string) string {
	createdAt = strings.TrimSpace(createdAt)
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		return t.Format("2006-01-02 15:04")
	}
	return createdAt
}

func orderProductName(bizType string, bizRefULID string) string {
	label := orderBizTypeLabel(bizType)
	if strings.TrimSpace(bizRefULID) == "" {
		return label
	}
	return label + " - " + strings.TrimSpace(bizRefULID)
}

func orderBizTypeLabel(bizType string) string {
	switch strings.ToUpper(strings.TrimSpace(bizType)) {
	case orderBizPipelinePayment:
		return "Pipeline Order"
	case orderBizStagePayment:
		return "Stage Order"
	case orderBizCourseRetakePayment:
		return "Retake Order"
	case orderBizPipelineUnlock:
		return "Pipeline Unlock Order"
	case orderBizCredentialApply:
		return "Credential Application Order"
	case orderBizBundlePurchase:
		return "Bundle Purchase"
	default:
		return strings.TrimSpace(bizType)
	}
}

func (h *Handler) pipelineName(r *http.Request, pipelineULID string, cache map[string]string) string {
	pipelineULID = strings.TrimSpace(pipelineULID)
	if pipelineULID == "" {
		return ""
	}
	if name, ok := cache[pipelineULID]; ok {
		return name
	}
	name := pipelineULID
	if resp, err := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
		Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipelineULID},
	}); err == nil && resp != nil && strings.TrimSpace(resp.GetName()) != "" {
		name = resp.GetName()
	}
	cache[pipelineULID] = name
	return name
}

func (h *Handler) previewPayment(ctx context.Context, bizType, bizRefULID string) (float64, string) {
	req := &mallpb.PreviewPaymentRequest{
		BizType:    bizType,
		BizRefUlid: bizRefULID,
	}
	resp, err := h.Mall.PreviewPayment(ctx, req)
	if err == nil && resp != nil {
		return float64(resp.GetTotal()) / 100.0, resp.GetCurrency()
	}
	return 0, ""
}
