package handler

import (
	"context"
	"net/http"
	"sort"
	"strings"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	items, total, err := h.listAdminOrders(r.Context(), query)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, adminOrderListResponse{Items: items, Total: total})
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
	CandidateULID string `json:"candidate_ulid"`
	BizType       string `json:"biz_type"`
	BizRefULID    string `json:"biz_ref_ulid"`
	AmountMinor   int64  `json:"amount_minor"`
	CurrencyCode  string `json:"currency_code"`
	OrderStatus   string `json:"order_status"`
	PaymentStatus string `json:"payment_status"`
	CreatedAt     string `json:"created_at"`
	PayOrderULID  string `json:"pay_order_ulid,omitempty"`
}

type adminOrderLister func(context.Context, adminOrderListQuery) ([]adminOrderSummary, int32, error)

const (
	adminBizTypePipelinePayment       = "PIPELINE_PAYMENT"
	adminBizTypeStagePayment          = "STAGE_PAYMENT"
	adminBizTypeCourseRetakePayment   = "COURSE_RETAKE_PAYMENT"
	adminBizTypePipelineUnlock        = "PIPELINE_UNLOCK"
	adminBizTypeCredentialApplication = "CREDENTIAL_APPLICATION"
	adminBizTypeBundlePurchase        = "BUNDLE_PURCHASE"
)

func (h *Handler) listAdminOrders(ctx context.Context, query adminOrderListQuery) ([]adminOrderSummary, int32, error) {
	listers := map[string]adminOrderLister{
		adminBizTypePipelinePayment:       h.listAdminPipelineOrders,
		adminBizTypeStagePayment:          h.listAdminStageOrders,
		adminBizTypeCourseRetakePayment:   h.listAdminCourseRetakeOrders,
		adminBizTypePipelineUnlock:        h.listAdminPipelineUnlockOrders,
		adminBizTypeCredentialApplication: h.listAdminCredentialApplicationOrders,
		adminBizTypeBundlePurchase:        h.listAdminBundleOrders,
	}

	if query.BizType != "" {
		lister, ok := listers[query.BizType]
		if !ok {
			return nil, 0, status.Error(codes.InvalidArgument, "unsupported biz_type")
		}
		return h.listAdminOrdersFromOneSource(ctx, query, lister)
	}

	fetchQuery := query
	fetchQuery.Limit = query.Offset + query.Limit
	fetchQuery.Offset = 0

	var items []adminOrderSummary
	var total int32
	for _, bizType := range []string{
		adminBizTypeBundlePurchase,
		adminBizTypePipelinePayment,
		adminBizTypeStagePayment,
		adminBizTypeCourseRetakePayment,
		adminBizTypePipelineUnlock,
		adminBizTypeCredentialApplication,
	} {
		got, gotTotal, err := listers[bizType](ctx, fetchQuery)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, got...)
		total += gotTotal
	}

	items = filterAdminOrders(items, query)
	sortAdminOrders(items)
	filteredTotal := int32(len(items))
	items = sliceAdminOrders(items, query.Offset, query.Limit)
	if query.BizRefULID != "" || query.PaymentStatus != "" {
		return items, filteredTotal, nil
	}
	return items, total, nil
}

func (h *Handler) listAdminOrdersFromOneSource(ctx context.Context, query adminOrderListQuery, lister adminOrderLister) ([]adminOrderSummary, int32, error) {
	if query.BizRefULID == "" && query.PaymentStatus == "" {
		return lister(ctx, query)
	}

	fetchQuery := query
	fetchQuery.Limit = query.Offset + query.Limit
	fetchQuery.Offset = 0
	items, _, err := lister(ctx, fetchQuery)
	if err != nil {
		return nil, 0, err
	}
	items = filterAdminOrders(items, query)
	sortAdminOrders(items)
	total := int32(len(items))
	return sliceAdminOrders(items, query.Offset, query.Limit), total, nil
}

func (h *Handler) listAdminPipelineOrders(ctx context.Context, query adminOrderListQuery) ([]adminOrderSummary, int32, error) {
	resp, err := h.Mall.ListPipelineOrders(ctx, &mallpb.ListPipelineOrdersRequest{
		CandidateUlid: query.CandidateULID,
		OrderStatus:   query.OrderStatus,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]adminOrderSummary, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		bizRef := item.GetPipelineOrderUlid()
		payOrder := item.GetPipelinePayOrderUlid()
		items = append(items, newAdminOrderSummary(adminBizTypePipelinePayment, bizRef, payOrder, item.GetCandidateUlid(), item.GetOrderStatus(), item.GetCreatedAt()))
	}
	return items, resp.GetTotal(), nil
}

func (h *Handler) listAdminStageOrders(ctx context.Context, query adminOrderListQuery) ([]adminOrderSummary, int32, error) {
	resp, err := h.Mall.ListStageOrders(ctx, &mallpb.ListStageOrdersRequest{
		CandidateUlid: query.CandidateULID,
		OrderStatus:   query.OrderStatus,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]adminOrderSummary, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		bizRef := item.GetStageOrderUlid()
		payOrder := item.GetStagePayOrderUlid()
		items = append(items, newAdminOrderSummary(adminBizTypeStagePayment, bizRef, payOrder, item.GetCandidateUlid(), item.GetOrderStatus(), item.GetCreatedAt()))
	}
	return items, resp.GetTotal(), nil
}

func (h *Handler) listAdminCourseRetakeOrders(ctx context.Context, query adminOrderListQuery) ([]adminOrderSummary, int32, error) {
	resp, err := h.Mall.ListCourseRetakeOrders(ctx, &mallpb.ListCourseRetakeOrdersRequest{
		CandidateUlid: query.CandidateULID,
		OrderStatus:   query.OrderStatus,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]adminOrderSummary, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		bizRef := item.GetCourseRetakeOrderUlid()
		payOrder := item.GetPayOrderUlid()
		items = append(items, newAdminOrderSummary(adminBizTypeCourseRetakePayment, bizRef, payOrder, item.GetCandidateUlid(), item.GetOrderStatus(), item.GetCreatedAt()))
	}
	return items, resp.GetTotal(), nil
}

func (h *Handler) listAdminPipelineUnlockOrders(ctx context.Context, query adminOrderListQuery) ([]adminOrderSummary, int32, error) {
	resp, err := h.Mall.ListPipelineUnlockOrders(ctx, &mallpb.ListPipelineUnlockOrdersRequest{
		CandidateUlid: query.CandidateULID,
		OrderStatus:   query.OrderStatus,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]adminOrderSummary, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		bizRef := item.GetPipelineUnlockOrderUlid()
		payOrder := item.GetPayOrderUlid()
		items = append(items, newAdminOrderSummary(adminBizTypePipelineUnlock, bizRef, payOrder, item.GetCandidateUlid(), item.GetOrderStatus(), item.GetCreatedAt()))
	}
	return items, resp.GetTotal(), nil
}

func (h *Handler) listAdminCredentialApplicationOrders(ctx context.Context, query adminOrderListQuery) ([]adminOrderSummary, int32, error) {
	resp, err := h.Mall.ListCredentialApplicationOrders(ctx, &mallpb.ListCredentialApplicationOrdersRequest{
		CandidateUlid: query.CandidateULID,
		OrderStatus:   query.OrderStatus,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]adminOrderSummary, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		bizRef := item.GetApplicationOrderUlid()
		payOrder := item.GetPayOrderUlid()
		items = append(items, newAdminOrderSummary(adminBizTypeCredentialApplication, bizRef, payOrder, item.GetCandidateUlid(), item.GetOrderStatus(), item.GetCreatedAt()))
	}
	return items, resp.GetTotal(), nil
}

func (h *Handler) listAdminBundleOrders(ctx context.Context, query adminOrderListQuery) ([]adminOrderSummary, int32, error) {
	resp, err := h.Mall.ListBundleOrders(ctx, &mallpb.ListBundleOrdersRequest{
		CandidateUlid: query.CandidateULID,
		OrderStatus:   query.OrderStatus,
		Limit:         query.Limit,
		Offset:        query.Offset,
	})
	if err != nil {
		return nil, 0, err
	}
	items := make([]adminOrderSummary, 0, len(resp.GetItems()))
	for _, item := range resp.GetItems() {
		bizRef := item.GetBundleOrderUlid()
		payOrder := item.GetBundlePayOrderUlid()
		items = append(items, newAdminOrderSummary(adminBizTypeBundlePurchase, bizRef, payOrder, item.GetCandidateUlid(), item.GetOrderStatus(), item.GetCreatedAt()))
	}
	return items, resp.GetTotal(), nil
}

func newAdminOrderSummary(bizType, bizRefULID, payOrderULID, candidateULID, orderStatus, createdAt string) adminOrderSummary {
	return adminOrderSummary{
		OrderULID:     firstNonEmpty(payOrderULID, bizRefULID),
		CandidateULID: candidateULID,
		BizType:       bizType,
		BizRefULID:    bizRefULID,
		OrderStatus:   orderStatus,
		PaymentStatus: deriveAdminPaymentStatus(orderStatus, payOrderULID),
		CreatedAt:     createdAt,
		PayOrderULID:  payOrderULID,
	}
}

func filterAdminOrders(items []adminOrderSummary, query adminOrderListQuery) []adminOrderSummary {
	if query.BizRefULID == "" && query.PaymentStatus == "" {
		return items
	}
	filtered := make([]adminOrderSummary, 0, len(items))
	for _, item := range items {
		if query.BizRefULID != "" && item.BizRefULID != query.BizRefULID && item.OrderULID != query.BizRefULID {
			continue
		}
		if query.PaymentStatus != "" && strings.ToUpper(item.PaymentStatus) != query.PaymentStatus {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func sortAdminOrders(items []adminOrderSummary) {
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})
}

func sliceAdminOrders(items []adminOrderSummary, offset, limit int32) []adminOrderSummary {
	if limit <= 0 || offset >= int32(len(items)) {
		return []adminOrderSummary{}
	}
	start := int(offset)
	end := start + int(limit)
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}

func deriveAdminPaymentStatus(orderStatus, payOrderULID string) string {
	status := strings.ToUpper(strings.TrimSpace(orderStatus))
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
