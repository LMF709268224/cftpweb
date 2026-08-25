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
		OrderStatus:   strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("order_status"))),
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
			PaymentStatus: strings.ToUpper(strings.TrimSpace(item.GetPaymentStatus())),
			CreatedAt:     item.GetCreatedAt(),
		})
	}

	WriteJSON(w, http.StatusOK, adminOrderListResponse{
		Items:      items,
		Total:      int32(total.Total),
		TotalLabel: total.Label(),
		TotalExact: total.Exact,
		NextCursor: resp.GetNextCursor(),
		PrevCursor: resp.GetPrevCursor(),
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
	PrevCursor string              `json:"prev_cursor,omitempty"`
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

type adminOrderDetailResponse struct {
	Summary        *mallpb.OrderSummary  `json:"summary"`
	BusinessDetail any                   `json:"business_detail"`
	Pricing        *adminOrderPricing    `json:"pricing"`
	Exemptions     []adminOrderExemption `json:"exemptions,omitempty"`
}

type adminOrderPricing struct {
	Available               bool                  `json:"available"`
	Source                  string                `json:"source,omitempty"`
	CurrencyCode            string                `json:"currency_code,omitempty"`
	BillableSubtotalMinor   *int64                `json:"billable_subtotal_minor,omitempty"`
	ExemptionDiscountMinor  *int64                `json:"exemption_discount_minor,omitempty"`
	PromotionDiscountMinor  *int64                `json:"promotion_discount_minor,omitempty"`
	TaxMinor                *int64                `json:"tax_minor,omitempty"`
	TotalMinor              *int64                `json:"total_minor,omitempty"`
	AmountPaidMinor         *int64                `json:"amount_paid_minor,omitempty"`
	ExemptionAmountRecorded bool                  `json:"exemption_amount_recorded"`
	Items                   []adminOrderPriceItem `json:"items,omitempty"`
	Coupons                 []adminOrderCoupon    `json:"coupons,omitempty"`
	PromoCodes              []string              `json:"promo_codes,omitempty"`
	UnavailableReason       string                `json:"unavailable_reason,omitempty"`
}

type adminOrderPriceItem struct {
	ItemType       string `json:"item_type,omitempty"`
	ItemULID       string `json:"item_ulid,omitempty"`
	Title          string `json:"title,omitempty"`
	UnitPriceMinor int64  `json:"unit_price_minor"`
	Quantity       int32  `json:"quantity"`
	SubtotalMinor  int64  `json:"subtotal_minor"`
}

type adminOrderCoupon struct {
	Code           string  `json:"code,omitempty"`
	Name           string  `json:"name,omitempty"`
	PercentOff     float64 `json:"percent_off,omitempty"`
	AmountOffMinor int64   `json:"amount_off_minor,omitempty"`
	CurrencyCode   string  `json:"currency_code,omitempty"`
}

type adminOrderExemption struct {
	CourseCCULID   string `json:"course_cc_ulid"`
	CredentialULID string `json:"credential_ulid,omitempty"`
}

func (h *Handler) GetOrderDetail(w http.ResponseWriter, r *http.Request) {
	orderULID := strings.TrimSpace(chi.URLParam(r, "order_ulid"))
	if !requireRequestField(w, orderULID, "order_ulid") {
		return
	}

	summaryResp, err := h.Mall.GetOrderSummary(r.Context(), &mallpb.GetOrderSummaryRequest{OrderUlid: orderULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	summary := summaryResp.GetSummary()
	if !summaryResp.GetFound() || summary == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, "order not found")
		return
	}

	detail, err := h.adminBusinessOrderDetail(r.Context(), summary.GetBizType(), summary.GetBizRefUlid())
	if err != nil {
		if appErr, ok := err.(*AppError); ok {
			HandleAppError(w, appErr)
		} else {
			HandleGrpcError(w, err)
		}
		return
	}
	pricing := h.adminOrderPricing(r.Context(), summary)
	WriteJSON(w, http.StatusOK, adminOrderDetailResponse{
		Summary:        summary,
		BusinessDetail: detail,
		Pricing:        pricing,
		Exemptions:     approvedOrderExemptions(detail),
	})
}

func (h *Handler) adminOrderPricing(ctx context.Context, summary *mallpb.OrderSummary) *adminOrderPricing {
	pricing := &adminOrderPricing{
		Available:               true,
		Source:                  "GMALL_ORDER_SUMMARY",
		CurrencyCode:            strings.ToUpper(strings.TrimSpace(summary.GetCurrencyCode())),
		TotalMinor:              int64Pointer(summary.GetAmountMinor()),
		ExemptionAmountRecorded: false,
	}
	// Exempt units are removed before the payment order is created, so GPAY has
	// no persisted monetary value for the exemption discount.
	if h.Gpay == nil {
		pricing.UnavailableReason = "gpay client is unavailable"
		return pricing
	}

	order, err := h.Gpay.GetOrder(ctx, &gpaypb.GetOrderRequest{
		Lookup: &gpaypb.GetOrderRequest_OrderUlid{OrderUlid: summary.GetOrderUlid()},
	})
	if err != nil {
		slog.WarnContext(ctx, "admin order pricing query failed", "order_ulid", summary.GetOrderUlid(), "error", err)
		pricing.UnavailableReason = "payment order detail is unavailable"
		return pricing
	}

	pricing.Source = "GPAY_ORDER"
	pricing.CurrencyCode = strings.ToUpper(strings.TrimSpace(order.GetCurrency()))
	pricing.TotalMinor = int64Pointer(order.GetAmount())
	if order.GetPaidAt() > 0 || strings.EqualFold(order.GetStripePaymentStatus(), "paid") {
		pricing.AmountPaidMinor = int64Pointer(order.GetAmount())
	}
	pricing.PromoCodes = append([]string(nil), order.GetPromoCodes()...)
	for _, coupon := range order.GetCoupons() {
		if coupon == nil {
			continue
		}
		pricing.Coupons = append(pricing.Coupons, adminOrderCoupon{
			Code:           coupon.GetCode(),
			Name:           coupon.GetName(),
			PercentOff:     coupon.GetPercentOff(),
			AmountOffMinor: coupon.GetAmountOff(),
			CurrencyCode:   strings.ToUpper(strings.TrimSpace(coupon.GetCurrency())),
		})
	}

	items, itemsErr := h.Gpay.ListOrderItems(ctx, &gpaypb.ListOrderItemsRequest{OrderUlid: summary.GetOrderUlid()})
	if itemsErr != nil {
		slog.WarnContext(ctx, "admin order item query failed", "order_ulid", summary.GetOrderUlid(), "error", itemsErr)
	} else {
		var subtotal int64
		for _, item := range items.GetItems() {
			if item == nil {
				continue
			}
			quantity := item.GetQuantity()
			itemSubtotal := item.GetBasePrice() * int64(quantity)
			subtotal += itemSubtotal
			pricing.Items = append(pricing.Items, adminOrderPriceItem{
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
			pricing.BillableSubtotalMinor = int64Pointer(subtotal)
		}
	}

	if strings.TrimSpace(order.GetStripeInvoiceId()) == "" {
		return pricing
	}
	invoice, invoiceErr := h.Gpay.GetInvoice(ctx, &gpaypb.GetInvoiceRequest{
		Lookup: &gpaypb.GetInvoiceRequest_StripeInvoiceId{StripeInvoiceId: order.GetStripeInvoiceId()},
	})
	if invoiceErr != nil {
		slog.WarnContext(ctx, "admin order invoice query failed", "order_ulid", summary.GetOrderUlid(), "stripe_invoice_id", order.GetStripeInvoiceId(), "error", invoiceErr)
		return pricing
	}

	pricing.Source = "GPAY_INVOICE"
	pricing.CurrencyCode = strings.ToUpper(strings.TrimSpace(invoice.GetCurrency()))
	pricing.BillableSubtotalMinor = int64Pointer(invoice.GetSubtotal())
	pricing.TaxMinor = int64Pointer(invoice.GetTax())
	pricing.TotalMinor = int64Pointer(invoice.GetTotal())
	pricing.AmountPaidMinor = int64Pointer(invoice.GetAmountPaid())
	// GPAY invoices currently have no shipping adjustment, so this identity is
	// the exact invoice-level promotion discount rather than an estimate.
	if discount := invoice.GetSubtotal() + invoice.GetTax() - invoice.GetTotal(); discount >= 0 {
		pricing.PromotionDiscountMinor = int64Pointer(discount)
	}
	return pricing
}

func int64Pointer(value int64) *int64 {
	return &value
}

func approvedOrderExemptions(detail any) []adminOrderExemption {
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
	result := make([]adminOrderExemption, 0, len(payload.Course))
	appendApproved := func(items []exemptionItem) {
		for _, item := range items {
			courseULID := strings.TrimSpace(item.CourseCCULID)
			if !item.Approved || courseULID == "" {
				continue
			}
			key := courseULID + "\x00" + strings.TrimSpace(item.CredentialULID)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, adminOrderExemption{
				CourseCCULID:   courseULID,
				CredentialULID: strings.TrimSpace(item.CredentialULID),
			})
		}
	}
	appendApproved(payload.Course)
	for _, stage := range payload.Stages {
		appendApproved(stage.Course)
	}
	return result
}

func (h *Handler) adminBusinessOrderDetail(ctx context.Context, bizType, bizRefULID string) (any, error) {
	bizRefULID = strings.TrimSpace(bizRefULID)
	switch strings.ToUpper(strings.TrimSpace(bizType)) {
	case "PIPELINE_PAYMENT":
		return h.Mall.GetPipelineOrderDetail(ctx, &mallpb.GetPipelineOrderDetailRequest{PipelineOrderUlid: bizRefULID})
	case "STAGE_PAYMENT":
		return h.Mall.GetStageOrderDetail(ctx, &mallpb.GetStageOrderDetailRequest{StageOrderUlid: bizRefULID})
	case "COURSE_RETAKE_PAYMENT":
		return h.Mall.GetCourseRetakeOrderDetail(ctx, &mallpb.GetCourseRetakeOrderDetailRequest{CourseRetakeOrderUlid: bizRefULID})
	case "PIPELINE_UNLOCK":
		return h.Mall.GetPipelineUnlockOrderDetail(ctx, &mallpb.GetPipelineUnlockOrderDetailRequest{PipelineUnlockOrderUlid: bizRefULID})
	case "CREDENTIAL_APPLICATION":
		return h.Mall.GetCredentialApplicationOrderDetail(ctx, &mallpb.GetCredentialApplicationOrderDetailRequest{ApplicationOrderUlid: bizRefULID})
	case "BUNDLE_PURCHASE":
		return h.Mall.AdminGetBundleOrderDetail(ctx, &mallpb.AdminGetBundleOrderDetailRequest{BundleOrderUlid: bizRefULID})
	default:
		return nil, NewError(http.StatusBadRequest, ErrInvalidRequest, "unsupported biz_type")
	}
}

// AdminSyncOrderMeta POST /api/mall/orders/sync-meta
func (h *Handler) AdminSyncOrderMeta(w http.ResponseWriter, r *http.Request) {
	var req mallpb.AdminSyncOrderMetaRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := ReadJSON(r, &req); err != nil {
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
