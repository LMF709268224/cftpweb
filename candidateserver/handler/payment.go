package handler

import (
	"net/http"
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
	// candidateID := CandidateID(r)

	var payments []GetPaymentStatusRsp

	// for _, id := range orderIDs {
	// 	resp, err := h.Pay.GetPaymentStatus(r.Context(), &paypb.GetPaymentStatusRequest{OrderId: id})
	// 	if err != nil {
	// 		slog.Error("Failed to get payment status", "error", err, "order_id", id)
	// 		continue
	// 	}
	// 	payments = append(payments, GetPaymentStatusRsp{
	// 		OrderID:           resp.OrderId,
	// 		RequestStatus:     resp.RequestStatus,
	// 		SessionStatus:     resp.SessionStatus,
	// 		PaymentStatus:     resp.PaymentStatus,
	// 		TotalAmount:       resp.TotalAmount,
	// 		Currency:          resp.Currency,
	// 		StripeCheckoutURL: resp.StripeCheckoutUrl,
	// 	})
	// }

	if payments == nil {
		payments = []GetPaymentStatusRsp{}
	}

	WriteJSON(w, http.StatusOK, payments)
}
