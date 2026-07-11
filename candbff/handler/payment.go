package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

type candidateCancelableOrder struct {
	OrderID    string
	BizType    string
	BizRefUlid string
	Status     string
	Candidate  string
}

// ListOrders GET /api/orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	page := parseCursorPage(r, 10)
	if page.PageSize > defaultCandidateOrderPageMax {
		page.PageSize = defaultCandidateOrderPageMax
	}

	bizType := normalizeOrderBizType(r.URL.Query().Get("biz_type"))
	if bizType != "" && !isCandidateOrderBizType(bizType) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "unsupported biz_type")
		return
	}
	orderStatus := normalizeOrderStatusFilter(r)

	req := &mallpb.ListOrdersRequest{
		Filters: &mallpb.OrderFilters{
			CandidateUlid: candidateID,
			BizType:       bizType,
			OrderStatus:   orderStatus,
		},
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
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

		payOrderID := strings.TrimSpace(item.GetOrderUlid())
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
			PayOrderUlid:         payOrderID,
			PipelinePayOrderUlid: payOrderID,
			CanViewInvoice:       statusStr == "completed" && payOrderID != "",
			CanCancel:            canCancelBusinessOrder(item.GetBizType(), rawStatus),
		}

		outOrders = append(outOrders, orderItem)
	}

	completed, totalAmount, err := h.candidateOrderAggregates(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, OrderListRsp{
		TotalOrders: int(total.Total),
		TotalLabel:  total.Label(),
		TotalExact:  total.Exact,
		Completed:   completed,
		TotalAmount: totalAmount,
		Page:        1,
		PageSize:    int(page.PageSize),
		TotalPages:  0,
		NextCursor:  resp.GetNextCursor(),
		PrevCursor:  resp.GetPrevCursor(),
		HasMore:     resp.GetHasMore(),
		Orders:      outOrders,
	})
}

func (h *Handler) candidateOrderAggregates(ctx context.Context, baseReq *mallpb.ListOrdersRequest) (int, float64, error) {
	if baseReq == nil {
		return 0, 0, nil
	}
	const limit uint32 = 50
	completed := 0
	totalAmount := 0.0
	cursor := ""
	for {
		resp, err := h.Mall.ListOrders(ctx, &mallpb.ListOrdersRequest{
			Filters:  baseReq.GetFilters(),
			Cursor:   cursor,
			PageSize: limit,
		})
		if err != nil {
			return 0, 0, err
		}
		items := resp.GetItems()
		for _, item := range items {
			if item == nil {
				continue
			}
			if candidateOrderStatus(item.GetOrderStatus()) == "completed" {
				completed++
				totalAmount += float64(item.GetAmountMinor()) / 100.0
			}
		}
		if !resp.GetHasMore() || resp.GetNextCursor() == "" {
			break
		}
		cursor = resp.GetNextCursor()
	}
	return completed, totalAmount, nil
}

// GetOrder GET /api/orders/{orderId}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	orderID := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if !requireRequestFields(w, candidateID, "candidate_id", orderID, "order_id") {
		return
	}

	resp, err := h.Mall.GetOrderDetail(r.Context(), &mallpb.GetOrderDetailRequest{OrderUlid: orderID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	detail := resp.GetDetail()
	summary := detail.GetSummary()
	if !resp.GetFound() || detail == nil || summary == nil || strings.TrimSpace(summary.GetCandidateUlid()) != candidateID {
		WriteError(w, http.StatusNotFound, ErrNotFound, "order not found or access denied")
		return
	}
	if !isCandidateOrderBizType(normalizeOrderBizType(summary.GetBizType())) {
		WriteError(w, http.StatusForbidden, ErrForbidden, "unsupported order type")
		return
	}

	WriteJSON(w, http.StatusOK, h.orderDetailResponse(resp))
}

// CancelOrder POST /api/orders/{orderId}/cancel
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	orderID := strings.TrimSpace(chi.URLParam(r, "orderId"))
	if !requireRequestFields(w, candidateID, "candidate_id", orderID, "order_id") {
		return
	}

	order, err := h.candidateCancelableOrder(r.Context(), orderID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if order == nil || order.Candidate != candidateID {
		WriteError(w, http.StatusNotFound, ErrNotFound, "order not found or access denied")
		return
	}
	if !isCandidateOrderBizType(order.BizType) {
		WriteError(w, http.StatusForbidden, ErrForbidden, "unsupported order type")
		return
	}
	if !canCancelBusinessOrder(order.BizType, order.Status) {
		WriteError(w, http.StatusConflict, ErrPrecondition, "order cannot be cancelled in current status")
		return
	}

	resp, err := h.Mall.CancelBusinessOrder(r.Context(), &mallpb.CancelBusinessOrderRequest{
		CandidateUlid: candidateID,
		BizType:       order.BizType,
		BizRefUlid:    order.BizRefUlid,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, CancelOrderRsp{
		Success:    true,
		Message:    resp.GetMessage(),
		OrderID:    orderID,
		BizType:    resp.GetBizType(),
		BizRefUlid: resp.GetBizRefUlid(),
		Status:     resp.GetOrderStatus(),
	})
}

func (h *Handler) orderDetailResponse(resp *mallpb.GetOrderDetailResponse) OrderDetailRsp {
	detail := resp.GetDetail()
	summary := detail.GetSummary()
	rawStatus := candidateOrderRawStatus(summary.GetOrderStatus())
	meta := summary.GetMeta()
	out := OrderDetailRsp{
		Found: resp.GetFound(),
		Summary: OrderSummaryDetail{
			OrderID:       strings.TrimSpace(summary.GetOrderUlid()),
			CandidateID:   strings.TrimSpace(summary.GetCandidateUlid()),
			BizType:       normalizeOrderBizType(summary.GetBizType()),
			BizRefUlid:    strings.TrimSpace(summary.GetBizRefUlid()),
			Currency:      strings.ToUpper(strings.TrimSpace(summary.GetCurrencyCode())),
			Amount:        float64(summary.GetAmountMinor()) / 100.0,
			AmountMinor:   summary.GetAmountMinor(),
			Status:        candidateOrderStatus(rawStatus),
			RawStatus:     rawStatus,
			PaymentStatus: strings.TrimSpace(summary.GetPaymentStatus()),
			CreatedAt:     formatOrderCreatedAt(summary.GetCreatedAt()),
		},
		GpayOrderUlid:       strings.TrimSpace(detail.GetGpayOrderUlid()),
		HasPaymentKey:       strings.TrimSpace(detail.GetPaymentKey()) != "",
		PaidAt:              formatOrderCreatedAt(detail.GetPaidAt()),
		ClosedAt:            formatOrderCreatedAt(detail.GetClosedAt()),
		LastReconciledAt:    formatOrderCreatedAt(detail.GetLastReconciledAt()),
		Version:             detail.GetVersion(),
		UpdatedAt:           formatOrderCreatedAt(detail.GetUpdatedAt()),
		OrderStatusAt:       formatOrderCreatedAt(detail.GetOrderStatusAt()),
		PaymentStatusAt:     formatOrderCreatedAt(detail.GetPaymentStatusAt()),
		DiscountUnsupported: true,
	}
	if meta != nil {
		out.Summary.Meta.ProductName = strings.TrimSpace(meta.GetProductName())
	}
	out.Raw = map[string]any{
		"summary":            out.Summary,
		"gpay_order_ulid":    out.GpayOrderUlid,
		"has_payment_key":    out.HasPaymentKey,
		"paid_at":            out.PaidAt,
		"closed_at":          out.ClosedAt,
		"last_reconciled_at": out.LastReconciledAt,
		"version":            out.Version,
		"updated_at":         out.UpdatedAt,
		"order_status_at":    out.OrderStatusAt,
		"payment_status_at":  out.PaymentStatusAt,
	}
	return out
}

func (h *Handler) candidateCancelableOrder(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	order, err := h.candidateBusinessOrder(ctx, orderID)
	if err != nil || order == nil {
		return order, err
	}
	if order.BizType == orderBizPipelinePayment {
		return nil, nil
	}
	return order, nil
}

func (h *Handler) candidateBusinessOrder(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	summaryResp, err := h.Mall.GetOrderSummary(ctx, &mallpb.GetOrderSummaryRequest{OrderUlid: orderID})
	if err == nil {
		if summary := summaryResp.GetSummary(); summaryResp.GetFound() && summary != nil {
			bizType := normalizeOrderBizType(summary.GetBizType())
			return &candidateCancelableOrder{
				OrderID:    strings.TrimSpace(summary.GetOrderUlid()),
				BizType:    bizType,
				BizRefUlid: strings.TrimSpace(summary.GetBizRefUlid()),
				Status:     summary.GetOrderStatus(),
				Candidate:  strings.TrimSpace(summary.GetCandidateUlid()),
			}, nil
		}
	} else if status.Code(err) != codes.NotFound {
		return nil, err
	}

	return h.candidateCancelableOrderByBizID(ctx, orderID)
}

func (h *Handler) verifyCandidatePaymentBizRef(ctx context.Context, candidateID, bizType, bizRefULID string) error {
	candidateID = strings.TrimSpace(candidateID)
	bizType = normalizePaymentBizType(bizType)
	bizRefULID = strings.TrimSpace(bizRefULID)
	if candidateID == "" || bizType == "" || bizRefULID == "" {
		return NewError(http.StatusBadRequest, ErrInvalidRequest, "candidate_id, biz_type and biz_ref_ulid are required")
	}

	order, err := h.candidateBusinessOrderForBiz(ctx, bizType, bizRefULID)
	if err != nil {
		return err
	}
	if order == nil || order.Candidate != candidateID {
		return NewError(http.StatusNotFound, ErrNotFound, "order not found or access denied")
	}
	return nil
}

func (h *Handler) candidateBusinessOrderForBiz(ctx context.Context, bizType string, bizRefULID string) (*candidateCancelableOrder, error) {
	switch normalizePaymentBizType(bizType) {
	case orderBizBundlePurchase:
		return h.bundleCancelableOrder(ctx, bizRefULID)
	case orderBizPipelineUnlock:
		return h.pipelineUnlockCancelableOrder(ctx, bizRefULID)
	case orderBizCredentialApply:
		return h.credentialApplicationCancelableOrder(ctx, bizRefULID)
	case orderBizCourseRetakePayment:
		return h.courseRetakeCancelableOrder(ctx, bizRefULID)
	case orderBizStagePayment:
		return h.stageCancelableOrder(ctx, bizRefULID)
	default:
		return nil, NewError(http.StatusBadRequest, ErrInvalidRequest, "unsupported biz_type")
	}
}

func normalizePaymentBizType(raw string) string {
	bizType := normalizeOrderBizType(raw)
	if bizType == orderBizPipelinePayment {
		return orderBizBundlePurchase
	}
	return bizType
}

func (h *Handler) candidateCancelableOrderByBizID(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	if order, err := h.bundleCancelableOrder(ctx, orderID); err != nil || order != nil {
		return order, err
	}
	if order, err := h.pipelineUnlockCancelableOrder(ctx, orderID); err != nil || order != nil {
		return order, err
	}
	if order, err := h.credentialApplicationCancelableOrder(ctx, orderID); err != nil || order != nil {
		return order, err
	}
	if order, err := h.courseRetakeCancelableOrder(ctx, orderID); err != nil || order != nil {
		return order, err
	}
	if order, err := h.stageCancelableOrder(ctx, orderID); err != nil || order != nil {
		return order, err
	}
	return nil, nil
}

func (h *Handler) bundleCancelableOrder(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	resp, err := h.Mall.GetBundleOrderSummary(ctx, &mallpb.GetBundleOrderSummaryRequest{BundleOrderUlid: orderID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	summary := resp.GetSummary()
	if !resp.GetFound() || summary == nil {
		return nil, nil
	}
	return &candidateCancelableOrder{
		OrderID:    strings.TrimSpace(summary.GetBundleOrderUlid()),
		BizType:    orderBizBundlePurchase,
		BizRefUlid: strings.TrimSpace(summary.GetBundleOrderUlid()),
		Status:     summary.GetOrderStatus(),
		Candidate:  strings.TrimSpace(summary.GetCandidateUlid()),
	}, nil
}

func (h *Handler) pipelineUnlockCancelableOrder(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	resp, err := h.Mall.GetPipelineUnlockOrderSummary(ctx, &mallpb.GetPipelineUnlockOrderSummaryRequest{PipelineUnlockOrderUlid: orderID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	summary := resp.GetSummary()
	if !resp.GetFound() || summary == nil {
		return nil, nil
	}
	return &candidateCancelableOrder{
		OrderID:    strings.TrimSpace(summary.GetPipelineUnlockOrderUlid()),
		BizType:    orderBizPipelineUnlock,
		BizRefUlid: strings.TrimSpace(summary.GetPipelineUnlockOrderUlid()),
		Status:     summary.GetOrderStatus(),
		Candidate:  strings.TrimSpace(summary.GetCandidateUlid()),
	}, nil
}

func (h *Handler) credentialApplicationCancelableOrder(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	resp, err := h.Mall.GetCredentialApplicationOrderSummary(ctx, &mallpb.GetCredentialApplicationOrderSummaryRequest{ApplicationOrderUlid: orderID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	summary := resp.GetSummary()
	if !resp.GetFound() || summary == nil {
		return nil, nil
	}
	return &candidateCancelableOrder{
		OrderID:    strings.TrimSpace(summary.GetApplicationOrderUlid()),
		BizType:    orderBizCredentialApply,
		BizRefUlid: strings.TrimSpace(summary.GetApplicationOrderUlid()),
		Status:     summary.GetOrderStatus(),
		Candidate:  strings.TrimSpace(summary.GetCandidateUlid()),
	}, nil
}

func (h *Handler) courseRetakeCancelableOrder(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	resp, err := h.Mall.GetCourseRetakeOrderSummary(ctx, &mallpb.GetCourseRetakeOrderSummaryRequest{CourseRetakeOrderUlid: orderID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	summary := resp.GetSummary()
	if !resp.GetFound() || summary == nil {
		return nil, nil
	}
	return &candidateCancelableOrder{
		OrderID:    strings.TrimSpace(summary.GetCourseRetakeOrderUlid()),
		BizType:    orderBizCourseRetakePayment,
		BizRefUlid: strings.TrimSpace(summary.GetCourseRetakeOrderUlid()),
		Status:     summary.GetOrderStatus(),
		Candidate:  strings.TrimSpace(summary.GetCandidateUlid()),
	}, nil
}

func (h *Handler) stageCancelableOrder(ctx context.Context, orderID string) (*candidateCancelableOrder, error) {
	resp, err := h.Mall.GetStageOrderSummary(ctx, &mallpb.GetStageOrderSummaryRequest{StageOrderUlid: orderID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	summary := resp.GetSummary()
	if !resp.GetFound() || summary == nil {
		return nil, nil
	}
	return &candidateCancelableOrder{
		OrderID:    strings.TrimSpace(summary.GetStageOrderUlid()),
		BizType:    orderBizStagePayment,
		BizRefUlid: strings.TrimSpace(summary.GetStageOrderUlid()),
		Status:     summary.GetOrderStatus(),
		Candidate:  strings.TrimSpace(summary.GetCandidateUlid()),
	}, nil
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

func canCancelCandidateOrder(raw string) bool {
	switch candidateOrderStatus(raw) {
	case "completed", "cancelled":
		return false
	default:
		return true
	}
}

func canCancelBusinessOrder(bizType, rawStatus string) bool {
	status := strings.ToUpper(strings.TrimSpace(rawStatus))
	switch normalizeOrderBizType(bizType) {
	case orderBizBundlePurchase:
		return status == "WAIT_PAYMENT"
	case orderBizStagePayment:
		return status == "WAIT_EXEMPTION_SELECTION" || status == "WAIT_STAGE_PAYMENT"
	case orderBizCourseRetakePayment:
		return status == "WAIT_PAYMENT"
	case orderBizPipelineUnlock:
		return status == "WAIT_PAYMENT"
	case orderBizCredentialApply:
		return status == "WAIT_REVIEW_FEE_PAYMENT"
	default:
		return false
	}
}

func candidateOrderRawStatus(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
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
