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
		// Try to get amount from PreviewPayment
		previewResp, err := h.Mall.PreviewPayment(r.Context(), &mallpb.PreviewPaymentRequest{
			BizType:    "PIPELINE_PAYMENT",
			BizRefUlid: item.GetPipelineOrderUlid(),
		})
		if err == nil && previewResp != nil {
			amount = float64(previewResp.GetTotal()) / 100.0
		} else if status == "completed" {
			// Actually for completed orders PreviewPayment might fail or we may need another way.
			// Currently we leave it as 0 if we can't get it.
		}

		if status == "completed" {
			totalAmount += amount
		}

		createdAt := item.GetCreatedAt()
		if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
			createdAt = t.Format("2006-01-02 15:04")
		}

		orders = append(orders, OrderItem{
			OrderID:       item.GetPipelineOrderUlid(),
			ProductName:   name,
			Status:        status,
			CreatedAt:     createdAt,
			PaymentMethod: item.GetPaymentMode(),
			Amount:        amount,
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
