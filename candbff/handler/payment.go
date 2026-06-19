package handler

import (
	"net/http"
	"sort"
	"strconv"
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
	defaultCandidateOrderPageMax = 50
)

var candidateOrderBizTypes = []string{
	orderBizPipelinePayment,
	orderBizStagePayment,
	orderBizCourseRetakePayment,
	orderBizPipelineUnlock,
	orderBizCredentialApply,
}

// GetOrder GET /api/orders/{orderId}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// Reserved for an order detail page. The order list is built from business-order list APIs.
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

	pipelineNames := make(map[string]string)
	paymentSummaryCache := make(map[string]paymentSummary)
	previewCache := make(map[string]paymentPreview)

	orders, totalOrders, err := h.listCandidateOrders(r, candidateID, bizType, page, pageSize, pipelineNames, paymentSummaryCache, previewCache)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	completed := 0
	totalAmount := 0.0
	for _, order := range orders {
		if order.Status == "completed" {
			completed++
			totalAmount += order.Amount
		}
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

func (h *Handler) listCandidateOrders(
	r *http.Request,
	candidateID string,
	bizType string,
	page int,
	pageSize int,
	pipelineNames map[string]string,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) ([]OrderItem, int, error) {
	if bizType != "" {
		return h.listCandidateOrdersByType(r, candidateID, bizType, page, pageSize, pipelineNames, paymentSummaryCache, previewCache)
	}

	allOrders := make([]OrderItem, 0)
	totalOrders := 0
	for _, currentBizType := range candidateOrderBizTypes {
		orders, total, err := h.listCandidateOrdersAllPages(r, candidateID, currentBizType, pipelineNames, paymentSummaryCache, previewCache)
		if err != nil {
			return nil, 0, err
		}
		totalOrders += total
		allOrders = append(allOrders, orders...)
	}

	sort.SliceStable(allOrders, func(i, j int) bool {
		left := parseOrderTime(allOrders[i].CreatedAt)
		right := parseOrderTime(allOrders[j].CreatedAt)
		if left.Equal(right) {
			return allOrders[i].OrderID > allOrders[j].OrderID
		}
		return left.After(right)
	})

	start := (page - 1) * pageSize
	if start >= len(allOrders) {
		return []OrderItem{}, totalOrders, nil
	}
	end := start + pageSize
	if end > len(allOrders) {
		end = len(allOrders)
	}
	return allOrders[start:end], totalOrders, nil
}

func (h *Handler) listCandidateOrdersAllPages(
	r *http.Request,
	candidateID string,
	bizType string,
	pipelineNames map[string]string,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) ([]OrderItem, int, error) {
	const chunkSize = defaultCandidateOrderPageMax
	items := make([]OrderItem, 0)
	total := 0
	offset := 0
	for {
		pageItems, pageTotal, err := h.listCandidateOrdersByType(r, candidateID, bizType, offset/chunkSize+1, chunkSize, pipelineNames, paymentSummaryCache, previewCache)
		if err != nil {
			return nil, 0, err
		}
		if offset == 0 {
			total = pageTotal
		}
		if len(pageItems) == 0 {
			break
		}
		items = append(items, pageItems...)
		offset += len(pageItems)
		if offset >= pageTotal {
			break
		}
	}
	return items, total, nil
}

func (h *Handler) listCandidateOrdersByType(
	r *http.Request,
	candidateID string,
	bizType string,
	page int,
	pageSize int,
	pipelineNames map[string]string,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) ([]OrderItem, int, error) {
	offset := (page - 1) * pageSize
	switch bizType {
	case orderBizPipelinePayment:
		resp, err := h.Mall.ListPipelineOrders(r.Context(), &mallpb.ListPipelineOrdersRequest{
			CandidateUlid: candidateID,
			Limit:         int32(pageSize),
			Offset:        int32(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		return h.buildPipelineOrderItems(r, resp.GetItems(), pipelineNames, paymentSummaryCache, previewCache), int(resp.GetTotal()), nil
	case orderBizStagePayment:
		resp, err := h.Mall.ListStageOrders(r.Context(), &mallpb.ListStageOrdersRequest{
			CandidateUlid: candidateID,
			Limit:         int32(pageSize),
			Offset:        int32(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		return h.buildStageOrderItems(r, resp.GetItems(), pipelineNames, paymentSummaryCache, previewCache), int(resp.GetTotal()), nil
	case orderBizCourseRetakePayment:
		resp, err := h.Mall.ListCourseRetakeOrders(r.Context(), &mallpb.ListCourseRetakeOrdersRequest{
			CandidateUlid: candidateID,
			Limit:         int32(pageSize),
			Offset:        int32(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		return h.buildCourseRetakeOrderItems(r, resp.GetItems(), paymentSummaryCache, previewCache), int(resp.GetTotal()), nil
	case orderBizPipelineUnlock:
		resp, err := h.Mall.ListPipelineUnlockOrders(r.Context(), &mallpb.ListPipelineUnlockOrdersRequest{
			CandidateUlid: candidateID,
			Limit:         int32(pageSize),
			Offset:        int32(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		return h.buildPipelineUnlockOrderItems(r, resp.GetItems(), pipelineNames, paymentSummaryCache, previewCache), int(resp.GetTotal()), nil
	case orderBizCredentialApply:
		resp, err := h.Mall.ListCredentialApplicationOrders(r.Context(), &mallpb.ListCredentialApplicationOrdersRequest{
			CandidateUlid: candidateID,
			Limit:         int32(pageSize),
			Offset:        int32(offset),
		})
		if err != nil {
			return nil, 0, err
		}
		return h.buildCredentialApplicationOrderItems(r, resp.GetItems(), paymentSummaryCache, previewCache), int(resp.GetTotal()), nil
	default:
		return []OrderItem{}, 0, nil
	}
}

func (h *Handler) buildPipelineOrderItems(
	r *http.Request,
	items []*mallpb.PipelineOrderSummary,
	pipelineNames map[string]string,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) []OrderItem {
	out := make([]OrderItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		pipelineULID := strings.TrimSpace(item.GetPipelineCcUlid())
		bizRefULID := strings.TrimSpace(item.GetPipelineOrderUlid())
		payOrderUlid := strings.TrimSpace(item.GetPipelinePayOrderUlid())
		name := h.pipelineName(r, pipelineULID, pipelineNames)
		if name == "" {
			name = orderProductName(orderBizPipelinePayment, bizRefULID)
		}
		out = append(out, h.buildOrderItem(r, orderBuildInput{
			OrderID:        bizRefULID,
			ProductName:    name,
			BizType:        orderBizPipelinePayment,
			BizRefUlid:     bizRefULID,
			PayOrderUlid:   payOrderUlid,
			RawStatus:      item.GetOrderStatus(),
			PipelineID:     pipelineULID,
			CreatedAt:      item.GetCreatedAt(),
			PaymentMethod:  item.GetPaymentMode(),
			PreviewBizType: orderBizPipelinePayment,
		}, paymentSummaryCache, previewCache))
	}
	return out
}

func (h *Handler) buildStageOrderItems(
	r *http.Request,
	items []*mallpb.StageOrderSummary,
	pipelineNames map[string]string,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) []OrderItem {
	out := make([]OrderItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		pipelineULID := strings.TrimSpace(item.GetPipelineCcUlid())
		bizRefULID := strings.TrimSpace(item.GetStageOrderUlid())
		stageULID := strings.TrimSpace(item.GetStageCcUlid())
		name := orderProductName(orderBizStagePayment, stageULID)
		if pipelineName := h.pipelineName(r, pipelineULID, pipelineNames); pipelineName != "" {
			name = pipelineName + " - Stage Order"
		}
		out = append(out, h.buildOrderItem(r, orderBuildInput{
			OrderID:        bizRefULID,
			ProductName:    name,
			BizType:        orderBizStagePayment,
			BizRefUlid:     bizRefULID,
			PayOrderUlid:   strings.TrimSpace(item.GetStagePayOrderUlid()),
			RawStatus:      item.GetOrderStatus(),
			PipelineID:     pipelineULID,
			CreatedAt:      item.GetCreatedAt(),
			PreviewBizType: orderBizStagePayment,
		}, paymentSummaryCache, previewCache))
	}
	return out
}

func (h *Handler) buildCourseRetakeOrderItems(
	r *http.Request,
	items []*mallpb.CourseRetakeOrderSummary,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) []OrderItem {
	out := make([]OrderItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		bizRefULID := strings.TrimSpace(item.GetCourseRetakeOrderUlid())
		name := orderProductName(orderBizCourseRetakePayment, strings.TrimSpace(item.GetCourseUnitCcUlid()))
		if item.GetRetriedCount() > 0 {
			name += " #" + strconv.FormatUint(uint64(item.GetRetriedCount()), 10)
		}
		out = append(out, h.buildOrderItem(r, orderBuildInput{
			OrderID:        bizRefULID,
			ProductName:    name,
			BizType:        orderBizCourseRetakePayment,
			BizRefUlid:     bizRefULID,
			PayOrderUlid:   strings.TrimSpace(item.GetPayOrderUlid()),
			RawStatus:      item.GetOrderStatus(),
			CreatedAt:      item.GetCreatedAt(),
			PreviewBizType: orderBizCourseRetakePayment,
		}, paymentSummaryCache, previewCache))
	}
	return out
}

func (h *Handler) buildPipelineUnlockOrderItems(
	r *http.Request,
	items []*mallpb.PipelineUnlockOrderSummary,
	pipelineNames map[string]string,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) []OrderItem {
	out := make([]OrderItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		pipelineULID := strings.TrimSpace(item.GetPipelineCcUlid())
		bizRefULID := strings.TrimSpace(item.GetPipelineUnlockOrderUlid())
		name := orderProductName(orderBizPipelineUnlock, bizRefULID)
		if pipelineName := h.pipelineName(r, pipelineULID, pipelineNames); pipelineName != "" {
			name = pipelineName + " - Pipeline Unlock"
		}
		out = append(out, h.buildOrderItem(r, orderBuildInput{
			OrderID:        bizRefULID,
			ProductName:    name,
			BizType:        orderBizPipelineUnlock,
			BizRefUlid:     bizRefULID,
			PayOrderUlid:   strings.TrimSpace(item.GetPayOrderUlid()),
			RawStatus:      item.GetOrderStatus(),
			PipelineID:     pipelineULID,
			CreatedAt:      item.GetCreatedAt(),
			PreviewBizType: orderBizPipelineUnlock,
		}, paymentSummaryCache, previewCache))
	}
	return out
}

func (h *Handler) buildCredentialApplicationOrderItems(
	r *http.Request,
	items []*mallpb.CredentialApplicationOrderSummary,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) []OrderItem {
	out := make([]OrderItem, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		bizRefULID := strings.TrimSpace(item.GetApplicationOrderUlid())
		out = append(out, h.buildOrderItem(r, orderBuildInput{
			OrderID:        bizRefULID,
			ProductName:    orderProductName(orderBizCredentialApply, bizRefULID),
			BizType:        orderBizCredentialApply,
			BizRefUlid:     bizRefULID,
			PayOrderUlid:   strings.TrimSpace(item.GetPayOrderUlid()),
			RawStatus:      item.GetOrderStatus(),
			CreatedAt:      item.GetCreatedAt(),
			PreviewBizType: orderBizCredentialApply,
		}, paymentSummaryCache, previewCache))
	}
	return out
}

type orderBuildInput struct {
	OrderID        string
	ProductName    string
	BizType        string
	BizRefUlid     string
	PayOrderUlid   string
	RawStatus      string
	PipelineID     string
	CreatedAt      string
	PaymentMethod  string
	PreviewBizType string
}

func (h *Handler) buildOrderItem(
	r *http.Request,
	input orderBuildInput,
	paymentSummaryCache map[string]paymentSummary,
	previewCache map[string]paymentPreview,
) OrderItem {
	status := candidateOrderStatus(input.RawStatus)
	amount := 0.0
	currency := "USD"
	actualOrderID := strings.TrimSpace(input.OrderID)

	if input.PayOrderUlid != "" {
		if summary := h.orderPaymentSummary(r, input.PayOrderUlid, paymentSummaryCache); summary.found {
			amount = summary.amount
			currency = summary.currency
			if summary.orderID != "" {
				actualOrderID = summary.orderID
			}
		}
	}
	if amount <= 0 && input.PreviewBizType != "" && input.BizRefUlid != "" {
		if preview := h.previewPayment(r, input.PreviewBizType, input.BizRefUlid, previewCache); preview.found {
			amount = preview.amount
			currency = preview.currency
		}
	}

	return OrderItem{
		OrderID:              actualOrderID,
		ProductName:          input.ProductName,
		BizType:              input.BizType,
		BizRefUlid:           input.BizRefUlid,
		Status:               status,
		RawStatus:            strings.ToUpper(strings.TrimSpace(input.RawStatus)),
		PipelineID:           input.PipelineID,
		CreatedAt:            formatOrderCreatedAt(input.CreatedAt),
		PaymentMethod:        input.PaymentMethod,
		Amount:               amount,
		Currency:             currency,
		PayOrderUlid:         input.PayOrderUlid,
		PipelinePayOrderUlid: input.PayOrderUlid,
		CanViewInvoice:       input.PayOrderUlid != "",
	}
}

type paymentSummary struct {
	found    bool
	orderID  string
	amount   float64
	currency string
}

type paymentPreview struct {
	found    bool
	amount   float64
	currency string
}

func (h *Handler) orderPaymentSummary(r *http.Request, payOrderULID string, cache map[string]paymentSummary) paymentSummary {
	payOrderULID = strings.TrimSpace(payOrderULID)
	if payOrderULID == "" {
		return paymentSummary{}
	}
	if cached, ok := cache[payOrderULID]; ok {
		return cached
	}
	out := paymentSummary{currency: "USD"}
	resp, err := h.Mall.GetOrderSummary(r.Context(), &mallpb.GetOrderSummaryRequest{OrderUlid: payOrderULID})
	if err == nil && resp != nil && resp.GetFound() && resp.GetSummary() != nil {
		summary := resp.GetSummary()
		out.found = true
		out.orderID = strings.TrimSpace(summary.GetOrderUlid())
		out.amount = float64(summary.GetAmountMinor()) / 100.0
		if strings.TrimSpace(summary.GetCurrencyCode()) != "" {
			out.currency = strings.ToUpper(strings.TrimSpace(summary.GetCurrencyCode()))
		}
	}
	cache[payOrderULID] = out
	return out
}

func (h *Handler) previewPayment(r *http.Request, bizType string, bizRefULID string, cache map[string]paymentPreview) paymentPreview {
	key := strings.ToUpper(strings.TrimSpace(bizType)) + ":" + strings.TrimSpace(bizRefULID)
	if key == ":" {
		return paymentPreview{}
	}
	if cached, ok := cache[key]; ok {
		return cached
	}
	out := paymentPreview{currency: "USD"}
	resp, err := h.Mall.PreviewPayment(r.Context(), &mallpb.PreviewPaymentRequest{
		BizType:    strings.ToUpper(strings.TrimSpace(bizType)),
		BizRefUlid: strings.TrimSpace(bizRefULID),
	})
	if err == nil && resp != nil {
		out.found = true
		out.amount = float64(resp.GetTotal()) / 100.0
		if strings.TrimSpace(resp.GetCurrency()) != "" {
			out.currency = strings.ToUpper(strings.TrimSpace(resp.GetCurrency()))
		}
	}
	cache[key] = out
	return out
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
		Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineULID},
	}); err == nil && resp != nil && strings.TrimSpace(resp.GetName()) != "" {
		name = resp.GetName()
	}
	cache[pipelineULID] = name
	return name
}

func normalizeOrderBizType(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
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
	orderStatus := strings.ToUpper(strings.TrimSpace(raw))
	if strings.Contains(orderStatus, "COMPLETED") || strings.Contains(orderStatus, "SUCCESS") || strings.Contains(orderStatus, "PAID") {
		return "completed"
	}
	if strings.Contains(orderStatus, "CANCEL") || strings.Contains(orderStatus, "FAILED") {
		return "cancelled"
	}
	if orderStatus == "" {
		return "pending"
	}
	return "processing"
}

func formatOrderCreatedAt(createdAt string) string {
	createdAt = strings.TrimSpace(createdAt)
	if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
		return t.Format("2006-01-02 15:04")
	}
	return createdAt
}

func parseOrderTime(createdAt string) time.Time {
	createdAt = strings.TrimSpace(createdAt)
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04", "2006-01-02 15:04:05"} {
		if t, err := time.Parse(layout, createdAt); err == nil {
			return t
		}
	}
	return time.Time{}
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
	default:
		return strings.TrimSpace(bizType)
	}
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
