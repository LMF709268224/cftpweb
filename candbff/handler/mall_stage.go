package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc/codes"
	gstatus "google.golang.org/grpc/status"
)

type createStageOrderReq struct {
	PipelineUlid string `json:"pipeline_ulid"`
	StageUlid    string `json:"stage_ulid"`
}

// CreateStageOrder POST /api/mall/pipelines/{pipelineId}/stages/{stageId}/purchase
func (h *Handler) CreateStageOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	stageID := strings.TrimSpace(chi.URLParam(r, "stageId"))

	var input createStageOrderReq
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}
	input.PipelineUlid = strings.TrimSpace(input.PipelineUlid)
	input.StageUlid = strings.TrimSpace(input.StageUlid)

	if !requireRequestFields(
		w,
		candidateID, "candidate_id",
		pipelineID, "pipeline_cc_ulid",
		stageID, "stage_cc_ulid",
		input.PipelineUlid, "pipeline_ulid",
		input.StageUlid, "stage_ulid",
	) {
		return
	}

	ctx := r.Context()
	if err := h.validateStagePurchaseRuntime(ctx, candidateID, pipelineID, input.PipelineUlid, stageID, input.StageUlid); err != nil {
		HandleGrpcError(w, err)
		return
	}

	pipelineOrder, err := h.completedByStagePipelineOrder(ctx, candidateID, pipelineID, input.PipelineUlid)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if pipelineOrder == nil {
		WriteError(w, http.StatusNotFound, ErrNotFound, "completed by-stage pipeline order not found for runtime pipeline")
		return
	}

	resp, err := h.Mall.CreateStageOrder(ctx, &mallpb.CreateStageOrderRequest{
		CandidateUlid:     candidateID,
		PipelineCcUlid:    pipelineID,
		PipelineOrderUlid: pipelineOrder.GetPipelineOrderUlid(),
		StageUlid:         input.StageUlid,
		StageCcUlid:       stageID,
		BundleOrderUlid:   pipelineOrder.GetBundleOrderUlid(),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) validateStagePurchaseRuntime(
	ctx context.Context,
	candidateID string,
	pipelineCcUlid string,
	pipelineUlid string,
	stageCcUlid string,
	stageUlid string,
) error {
	resp, err := h.Gprog.GetPipelineDetail(ctx, &gprogpb.GetPipelineDetailReq{
		PipelineUlid: pipelineUlid,
	})
	if err != nil {
		return err
	}
	pipeline := resp.GetPipeline()
	if pipeline == nil ||
		strings.TrimSpace(pipeline.GetCandidateUlid()) != candidateID ||
		strings.TrimSpace(pipeline.GetPipelineCcUlid()) != pipelineCcUlid {
		return gstatus.Error(codes.PermissionDenied, "pipeline runtime does not belong to candidate")
	}
	for _, stage := range resp.GetStages() {
		if stage == nil || stage.GetStage() == nil {
			continue
		}
		summary := stage.GetStage()
		if strings.TrimSpace(summary.GetStageUlid()) == stageUlid &&
			strings.TrimSpace(summary.GetStageCcUlid()) == stageCcUlid {
			if summary.GetStatus() != gprogpb.StageStatus_STAGE_STATUS_WAIT_CANDIDATE {
				return gstatus.Error(codes.FailedPrecondition, "stage is not waiting for candidate action")
			}
			return nil
		}
	}
	return gstatus.Error(codes.InvalidArgument, "stage runtime does not match stage configuration")
}

func (h *Handler) completedByStagePipelineOrder(
	ctx context.Context,
	candidateID string,
	pipelineCcUlid string,
	pipelineUlid string,
) (*mallpb.PipelineOrderSummary, error) {
	ordersResp, err := h.Mall.ListPipelineOrders(ctx, &mallpb.ListPipelineOrdersRequest{
		Filters: &mallpb.PipelineOrderFilters{
			CandidateUlid:  candidateID,
			PipelineCcUlid: pipelineCcUlid,
			OrderStatus:    "COMPLETED",
			PaymentMode:    "BY_STAGE",
		},
		PageSize: 20,
	})
	if err != nil {
		return nil, err
	}

	for _, order := range ordersResp.GetItems() {
		if order == nil || strings.TrimSpace(order.GetPipelineOrderUlid()) == "" {
			continue
		}
		detailResp, detailErr := h.Mall.GetPipelineOrderDetail(ctx, &mallpb.GetPipelineOrderDetailRequest{
			PipelineOrderUlid: order.GetPipelineOrderUlid(),
		})
		if detailErr != nil {
			return nil, detailErr
		}
		if !detailResp.GetFound() || detailResp.GetDetail() == nil {
			continue
		}
		if strings.TrimSpace(detailResp.GetDetail().GetInstantiatedPipelineUlid()) != pipelineUlid {
			continue
		}
		summary := detailResp.GetDetail().GetSummary()
		if summary == nil {
			continue
		}
		return summary, nil
	}
	return nil, nil
}
