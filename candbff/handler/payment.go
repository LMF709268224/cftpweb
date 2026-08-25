package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
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

	for _, item := range resp.GetItems() {
		if item == nil {
			continue
		}

		rawStatus := candidateOrderRawStatus(item.GetOrderStatus())
		amount := float64(item.GetAmountMinor()) / 100.0
		currency := item.GetCurrencyCode()

		name := ""
		if meta := item.GetMeta(); meta != nil && meta.GetProductName() != "" {
			name = strings.TrimSpace(meta.GetProductName())
		}
		if name == "" {
			name = orderBizTypeLabel(item.GetBizType())
		}

		payOrderID := strings.TrimSpace(item.GetOrderUlid())
		orderItem := OrderItem{
			OrderID:              item.GetOrderUlid(),
			ProductName:          name,
			BizType:              item.GetBizType(),
			BizRefUlid:           item.GetBizRefUlid(),
			OrderStatus:          rawStatus,
			PaymentStatus:        strings.TrimSpace(item.GetPaymentStatus()),
			CreatedAt:            item.GetCreatedAt(),
			Amount:               amount,
			Currency:             currency,
			PayOrderUlid:         payOrderID,
			PipelinePayOrderUlid: payOrderID,
			CanViewInvoice:       rawStatus == "COMPLETED" && payOrderID != "",
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
	guard := newCursorScanGuard()
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
			if isOrderCompleted(item.GetOrderStatus()) {
				completed++
				totalAmount += float64(item.GetAmountMinor()) / 100.0
			}
		}
		nextCursor, done, guardErr := guard.next(cursor, resp.GetHasMore(), resp.GetNextCursor())
		if guardErr != nil {
			return 0, 0, status.Error(codes.Internal, guardErr.Error())
		}
		if done {
			break
		}
		cursor = nextCursor
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

	businessDetail, err := h.businessOrderDetail(r.Context(), summary.GetBizType(), summary.GetBizRefUlid())
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			HandleAppError(w, appErr)
		} else {
			HandleGrpcError(w, err)
		}
		return
	}

	out := h.orderDetailResponse(resp)
	out.BusinessDetail = businessDetail
	out.Pricing = h.candidateOrderPricing(r.Context(), out.GpayOrderUlid, summary)
	out.Exemptions = candidateOrderExemptions(businessDetail)
	WriteJSON(w, http.StatusOK, out)
}

// CancelOrder POST /api/orders/cancel
func (h *Handler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	var req CancelOrderReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.BizType = normalizeOrderBizType(req.BizType)
	req.BizRefUlid = strings.TrimSpace(req.BizRefUlid)
	if !requireRequestFields(w, candidateID, "candidate_id", req.BizType, "biz_type", req.BizRefUlid, "biz_ref_ulid") {
		return
	}
	if !isCandidateOrderBizType(req.BizType) {
		WriteError(w, http.StatusForbidden, ErrForbidden, "unsupported order type")
		return
	}

	order, err := h.candidateCommonOrderForBiz(r.Context(), candidateID, req.BizType, req.BizRefUlid)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if order == nil ||
		order.Candidate != candidateID ||
		order.BizType != req.BizType ||
		order.BizRefUlid != req.BizRefUlid {
		WriteError(w, http.StatusNotFound, ErrNotFound, "order not found or access denied")
		return
	}
	if !canCancelCommonOrderStatus(order.Status) {
		WriteError(w, http.StatusConflict, ErrPrecondition, "order cannot be cancelled in current status")
		return
	}

	resp, err := h.Mall.CancelBusinessOrder(r.Context(), &mallpb.CancelBusinessOrderRequest{
		CandidateUlid: candidateID,
		BizType:       req.BizType,
		BizRefUlid:    req.BizRefUlid,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, CancelOrderRsp{
		Success:    true,
		Message:    resp.GetMessage(),
		OrderID:    order.OrderID,
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
			OrderStatus:   rawStatus,
			PaymentStatus: strings.TrimSpace(summary.GetPaymentStatus()),
			CreatedAt:     summary.GetCreatedAt(),
		},
		GpayOrderUlid:    strings.TrimSpace(detail.GetGpayOrderUlid()),
		HasPaymentKey:    strings.TrimSpace(detail.GetPaymentKey()) != "",
		PaidAt:           detail.GetPaidAt(),
		ClosedAt:         detail.GetClosedAt(),
		LastReconciledAt: detail.GetLastReconciledAt(),
		Version:          detail.GetVersion(),
		UpdatedAt:        detail.GetUpdatedAt(),
		OrderStatusAt:    detail.GetOrderStatusAt(),
		PaymentStatusAt:  detail.GetPaymentStatusAt(),
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

func (h *Handler) candidateOrderPricing(ctx context.Context, gpayOrderULID string, summary *mallpb.OrderSummary) *OrderPricingDetail {
	pricing := &OrderPricingDetail{
		Available:               true,
		Source:                  "GMALL_ORDER_SUMMARY",
		CurrencyCode:            strings.ToUpper(strings.TrimSpace(summary.GetCurrencyCode())),
		TotalMinor:              orderInt64Pointer(summary.GetAmountMinor()),
		ExemptionAmountRecorded: false,
	}
	// Exempt units are removed before the payment order is created, so GPAY has
	// no persisted monetary value for the exemption discount.
	gpayOrderULID = strings.TrimSpace(gpayOrderULID)
	if gpayOrderULID == "" {
		pricing.UnavailableReason = "payment order reference is unavailable"
		return pricing
	}
	if h.Gpay == nil {
		pricing.UnavailableReason = "gpay client is unavailable"
		return pricing
	}

	order, err := h.Gpay.GetOrder(ctx, &gpaypb.GetOrderRequest{
		Lookup: &gpaypb.GetOrderRequest_OrderUlid{OrderUlid: gpayOrderULID},
	})
	if err != nil {
		slog.WarnContext(ctx, "candidate order pricing query failed", "gpay_order_ulid", gpayOrderULID, "error", err)
		pricing.UnavailableReason = "payment order detail is unavailable"
		return pricing
	}

	pricing.Source = "GPAY_ORDER"
	pricing.CurrencyCode = strings.ToUpper(strings.TrimSpace(order.GetCurrency()))
	pricing.TotalMinor = orderInt64Pointer(order.GetAmount())
	if order.GetPaidAt() > 0 || strings.EqualFold(order.GetStripePaymentStatus(), "paid") {
		pricing.AmountPaidMinor = orderInt64Pointer(order.GetAmount())
	}
	pricing.PromoCodes = append([]string(nil), order.GetPromoCodes()...)
	for _, coupon := range order.GetCoupons() {
		if coupon == nil {
			continue
		}
		pricing.Coupons = append(pricing.Coupons, OrderCoupon{
			Code:           coupon.GetCode(),
			Name:           coupon.GetName(),
			PercentOff:     coupon.GetPercentOff(),
			AmountOffMinor: coupon.GetAmountOff(),
			CurrencyCode:   strings.ToUpper(strings.TrimSpace(coupon.GetCurrency())),
		})
	}

	items, itemsErr := h.Gpay.ListOrderItems(ctx, &gpaypb.ListOrderItemsRequest{OrderUlid: gpayOrderULID})
	if itemsErr != nil {
		slog.WarnContext(ctx, "candidate order item query failed", "gpay_order_ulid", gpayOrderULID, "error", itemsErr)
		pricing.UnavailableReason = "payment order items are unavailable"
	} else {
		var subtotal int64
		for _, item := range items.GetItems() {
			if item == nil {
				continue
			}
			quantity := item.GetQuantity()
			itemSubtotal := item.GetBasePrice() * int64(quantity)
			subtotal += itemSubtotal
			pricing.Items = append(pricing.Items, OrderPriceItem{
				ItemType:       item.GetItemType(),
				ItemULID:       item.GetItemId(),
				Title:          item.GetTitle(),
				UnitPriceMinor: item.GetBasePrice(),
				Quantity:       quantity,
				SubtotalMinor:  itemSubtotal,
			})
		}
		if len(pricing.Items) > 0 {
			pricing.Source = "GPAY_ORDER_ITEMS"
			pricing.BillableSubtotalMinor = orderInt64Pointer(subtotal)
		}
	}

	if strings.TrimSpace(order.GetStripeInvoiceId()) == "" {
		return pricing
	}
	invoice, invoiceErr := h.Gpay.GetInvoice(ctx, &gpaypb.GetInvoiceRequest{
		Lookup: &gpaypb.GetInvoiceRequest_StripeInvoiceId{StripeInvoiceId: order.GetStripeInvoiceId()},
	})
	if invoiceErr != nil {
		slog.WarnContext(ctx, "candidate order invoice query failed", "gpay_order_ulid", gpayOrderULID, "stripe_invoice_id", order.GetStripeInvoiceId(), "error", invoiceErr)
		pricing.UnavailableReason = "payment invoice detail is unavailable"
		return pricing
	}

	pricing.Source = "GPAY_INVOICE"
	pricing.CurrencyCode = strings.ToUpper(strings.TrimSpace(invoice.GetCurrency()))
	pricing.BillableSubtotalMinor = orderInt64Pointer(invoice.GetSubtotal())
	pricing.TaxMinor = orderInt64Pointer(invoice.GetTax())
	pricing.TotalMinor = orderInt64Pointer(invoice.GetTotal())
	pricing.AmountPaidMinor = orderInt64Pointer(invoice.GetAmountPaid())
	// GPAY invoices currently have no shipping adjustment, so this identity is
	// the exact invoice-level promotion discount rather than an estimate.
	if discount := invoice.GetSubtotal() + invoice.GetTax() - invoice.GetTotal(); discount >= 0 {
		pricing.PromotionDiscountMinor = orderInt64Pointer(discount)
	}
	return pricing
}

func orderInt64Pointer(value int64) *int64 {
	return &value
}

func candidateOrderExemptions(detail any) []OrderExemption {
	var raw string
	switch response := detail.(type) {
	case *mallpb.GetPipelineOrderDetailResponse:
		raw = response.GetDetail().GetFinalExemptionsJson()
	case *mallpb.GetStageOrderDetailResponse:
		raw = response.GetDetail().GetFinalExemptionsJson()
	}
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	type exemptionItem struct {
		CourseCCULID   string `json:"course_cc_ulid"`
		CredentialULID string `json:"credential_ulid"`
		Approved       bool   `json:"approved"`
	}
	type exemptionStage struct {
		Course []exemptionItem `json:"course"`
	}
	var payload struct {
		Course []exemptionItem  `json:"course"`
		Stages []exemptionStage `json:"stages"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	result := make([]OrderExemption, 0, len(payload.Course))
	appendApproved := func(items []exemptionItem) {
		for _, item := range items {
			courseULID := strings.TrimSpace(item.CourseCCULID)
			if !item.Approved || courseULID == "" {
				continue
			}
			credentialULID := strings.TrimSpace(item.CredentialULID)
			key := courseULID + "\x00" + credentialULID
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, OrderExemption{
				CourseCCULID:   courseULID,
				CredentialULID: credentialULID,
			})
		}
	}
	appendApproved(payload.Course)
	for _, stage := range payload.Stages {
		appendApproved(stage.Course)
	}
	return result
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

func (h *Handler) candidateCommonOrderForBiz(ctx context.Context, candidateID, bizType, bizRefULID string) (*candidateCancelableOrder, error) {
	resp, err := h.Mall.ListOrders(ctx, &mallpb.ListOrdersRequest{
		Filters: &mallpb.OrderFilters{
			CandidateUlid: strings.TrimSpace(candidateID),
			BizType:       normalizeOrderBizType(bizType),
			BizRefUlid:    strings.TrimSpace(bizRefULID),
		},
		PageSize: 1,
	})
	if err != nil {
		return nil, err
	}
	for _, summary := range resp.GetItems() {
		if summary == nil {
			continue
		}
		return &candidateCancelableOrder{
			OrderID:    strings.TrimSpace(summary.GetOrderUlid()),
			BizType:    normalizeOrderBizType(summary.GetBizType()),
			BizRefUlid: strings.TrimSpace(summary.GetBizRefUlid()),
			Status:     candidateOrderRawStatus(summary.GetOrderStatus()),
			Candidate:  strings.TrimSpace(summary.GetCandidateUlid()),
		}, nil
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

func isOrderCompleted(raw string) bool {
	status := strings.ToUpper(strings.TrimSpace(raw))
	return status == "COMPLETED"
}

func canCancelCommonOrderStatus(raw string) bool {
	status := strings.ToUpper(strings.TrimSpace(raw))
	return status == "WAIT_PAYMENT" || status == "PENDING"
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

func (h *Handler) businessOrderDetail(ctx context.Context, bizType, bizRefULID string) (any, error) {
	bizRefULID = strings.TrimSpace(bizRefULID)
	switch normalizeOrderBizType(bizType) {
	case orderBizPipelinePayment:
		return h.Mall.GetPipelineOrderDetail(ctx, &mallpb.GetPipelineOrderDetailRequest{PipelineOrderUlid: bizRefULID})
	case orderBizStagePayment:
		return h.Mall.GetStageOrderDetail(ctx, &mallpb.GetStageOrderDetailRequest{StageOrderUlid: bizRefULID})
	case orderBizCourseRetakePayment:
		return h.Mall.GetCourseRetakeOrderDetail(ctx, &mallpb.GetCourseRetakeOrderDetailRequest{CourseRetakeOrderUlid: bizRefULID})
	case orderBizPipelineUnlock:
		return h.Mall.GetPipelineUnlockOrderDetail(ctx, &mallpb.GetPipelineUnlockOrderDetailRequest{PipelineUnlockOrderUlid: bizRefULID})
	case orderBizCredentialApply:
		return h.Mall.GetCredentialApplicationOrderDetail(ctx, &mallpb.GetCredentialApplicationOrderDetailRequest{ApplicationOrderUlid: bizRefULID})
	case orderBizBundlePurchase:
		return h.Mall.GetBundleOrderDetail(ctx, &mallpb.GetBundleOrderDetailRequest{BundleOrderUlid: bizRefULID})
	default:
		return nil, NewError(http.StatusBadRequest, ErrInvalidRequest, "unsupported biz_type")
	}
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
