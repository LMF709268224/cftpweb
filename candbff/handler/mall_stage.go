package handler

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
)

// CreateStageOrder POST /api/mall/pipelines/{pipelineId}/stages/{stageId}/purchase
func (h *Handler) CreateStageOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	stageID := strings.TrimSpace(chi.URLParam(r, "stageId"))

	if candidateID == "" || pipelineID == "" || stageID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "missing required parameters")
		return
	}

	// 1. Fetch the active pipeline order to get pipeline_order_ulid and bundle_order_ulid
	ctx := r.Context()
	ordersResp, err := h.Mall.ListPipelineOrders(ctx, &mallpb.ListPipelineOrdersRequest{
		Filters: &mallpb.PipelineOrderFilters{
			CandidateUlid:  candidateID,
			PipelineCcUlid: pipelineID,
			OrderStatus:    "PAID", // Assuming the pipeline order is PAID
		},
		PageSize: 1,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	if len(ordersResp.Items) == 0 {
		WriteError(w, http.StatusNotFound, ErrNotFound, "no active pipeline order found for candidate")
		return
	}

	pipelineOrder := ordersResp.Items[0]

	// 2. Call CreateStageOrder
	req := &mallpb.CreateStageOrderRequest{
		CandidateUlid:     candidateID,
		PipelineCcUlid:    pipelineID,
		PipelineOrderUlid: pipelineOrder.GetPipelineOrderUlid(),
		StageCcUlid:       stageID,
		BundleOrderUlid:   pipelineOrder.GetBundleOrderUlid(),
	}

	resp, err := h.Mall.CreateStageOrder(ctx, req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
