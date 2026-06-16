package handler

import (
	"net/http"
	"strings"
	"time"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
)

// GetOrder  GET /api/orders/{orderId}
func (h *Handler) GetOrder(w http.ResponseWriter, r *http.Request) {
	// orderID := chi.URLParam(r, "orderId")
	// candidateID := CandidateID(r)

	// // TODO: 需 gmall 增加 GetOrder gRPC 接口
	// resp, err := h.Mall.GetPipelineOrderStatus(r.Context(), &mallpb.GetPipelineOrderStatusRequest{
	// 	PipelineOrderUlid: orderID,
	// })
	// if err != nil {
	// 	HandleGrpcError(w, err)
	// 	return
	// }

	// WriteJSON(w, http.StatusOK, GetOrderRsp{
	// 	OrderID: orderID,
	// 	Status:  "TODO_FROM_MALL", // 暂时占位
	// })
}

// ListOrders GET /api/orders
func (h *Handler) ListOrders(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	bizType := r.URL.Query().Get("biz_type")

	if bizType != "" && bizType != "PIPELINE_PAYMENT" {
		// Currently only PIPELINE_PAYMENT is implemented to fetch names and details.
		// Return empty for other types until their respective detail fetchers are added.
		WriteJSON(w, http.StatusOK, OrderListRsp{
			TotalOrders: 0,
			Completed:   0,
			TotalAmount: 0.0,
			Orders:      []OrderItem{},
		})
		return
	}

	mallResp, err := h.Mall.ListPipelineOrders(r.Context(), &mallpb.ListPipelineOrdersRequest{
		CandidateUlid: candidateID,
		Limit:         100,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	var orders []OrderItem
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
		actualOrderUlid := item.GetPipelineOrderUlid() // default fallback
		// 获取真实的支付订单来拿正确的金额 (Total Amount) 和真实的 Order ID (用于发票)
		listOrdersResp, err := h.Mall.ListOrders(r.Context(), &mallpb.ListOrdersRequest{
			BizType:    "PIPELINE_PAYMENT",
			BizRefUlid: item.GetPipelineOrderUlid(),
			Limit:      1,
		})
		if err == nil && listOrdersResp != nil && len(listOrdersResp.GetItems()) > 0 {
			realOrder := listOrdersResp.GetItems()[0]
			amount = float64(realOrder.GetAmountMinor()) / 100.0
			if strings.TrimSpace(realOrder.GetCurrencyCode()) != "" {
				currency = strings.ToUpper(strings.TrimSpace(realOrder.GetCurrencyCode()))
			}
			actualOrderUlid = realOrder.GetOrderUlid()
		} else {
			// 如果由于某种原因真实支付订单还未生成（比如刚创建还在 pending），回退使用 PreviewPayment
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
			OrderID:       actualOrderUlid,
			ProductName:   name,
			Status:        status,
			RawStatus:     item.GetOrderStatus(),
			PipelineID:    pipelineUlid,
			CreatedAt:     createdAt,
			PaymentMethod: item.GetPaymentMode(),
			Amount:        amount,
			Currency:      currency,
		})
	}

	if orders == nil {
		orders = []OrderItem{}
	}

	WriteJSON(w, http.StatusOK, OrderListRsp{
		TotalOrders: len(orders),
		Completed:   completed,
		TotalAmount: totalAmount,
		Orders:      orders,
	})
}
