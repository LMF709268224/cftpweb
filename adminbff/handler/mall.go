package handler

import (
	"net/http"
	"strings"

	mallpb "github.com/LMF709268224/cftpproto/gmall"
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
	candidateULID := strings.TrimSpace(r.URL.Query().Get("candidate_ulid"))
	bizType := strings.TrimSpace(r.URL.Query().Get("biz_type"))
	bizRefULID := strings.TrimSpace(r.URL.Query().Get("biz_ref_ulid"))
	orderStatus := strings.TrimSpace(r.URL.Query().Get("order_status"))
	paymentStatus := strings.TrimSpace(r.URL.Query().Get("payment_status"))

	req := &mallpb.ListOrdersRequest{
		CandidateUlid: candidateULID,
		BizType:       bizType,
		BizRefUlid:    bizRefULID,
		OrderStatus:   orderStatus,
		PaymentStatus: paymentStatus,
		Limit:         int32(parseUint32Query(r, "limit")),
		Offset:        int32(parseUint32Query(r, "offset")),
	}

	resp, err := h.Mall.ListOrders(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
