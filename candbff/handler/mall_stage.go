package handler

import (
	"context"
	"encoding/json"
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

type selectStageExemptionsReq struct {
	StageCcUlid         string   `json:"stage_cc_ulid"`
	ExemptedUnitCcUlids []string `json:"exempted_unit_cc_ulids"`
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

// SelectStageExemptions POST /api/mall/stage-orders/{stageOrderId}/exemptions
func (h *Handler) SelectStageExemptions(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	stageOrderID := strings.TrimSpace(chi.URLParam(r, "stageOrderId"))

	var input selectStageExemptionsReq
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}
	input.StageCcUlid = strings.TrimSpace(input.StageCcUlid)
	input.ExemptedUnitCcUlids = compactStageExemptionUnitIDs(input.ExemptedUnitCcUlids)

	if !requireRequestFields(
		w,
		candidateID, "candidate_id",
		stageOrderID, "stage_order_ulid",
		input.StageCcUlid, "stage_cc_ulid",
	) {
		return
	}

	order, err := h.stageCancelableOrder(r.Context(), stageOrderID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if order == nil || order.Candidate != candidateID {
		WriteError(w, http.StatusNotFound, ErrNotFound, "stage order not found or access denied")
		return
	}
	if strings.ToUpper(strings.TrimSpace(order.Status)) != "WAIT_EXEMPTION_SELECTION" {
		WriteError(w, http.StatusConflict, ErrPrecondition, "stage order is not waiting for exemption selection")
		return
	}

	exemptionsJSON, err := json.Marshal(map[string]any{
		"stage_cc_ulid":          input.StageCcUlid,
		"exempted_unit_cc_ulids": input.ExemptedUnitCcUlids,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to encode stage exemptions")
		return
	}

	resp, err := h.Mall.SelectStageExemptions(r.Context(), &mallpb.SelectStageExemptionsRequest{
		StageOrderUlid: stageOrderID,
		ExemptionsJson: string(exemptionsJSON),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func compactStageExemptionUnitIDs(values []string) []string {
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
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
	const pageSize uint32 = 100
	filters := &mallpb.PipelineOrderFilters{
		CandidateUlid:  candidateID,
		PipelineCcUlid: pipelineCcUlid,
		OrderStatus:    "COMPLETED",
		PaymentMode:    "BY_STAGE",
	}
	cursor := ""
	guard := newCursorScanGuard()
	for {
		ordersResp, err := h.Mall.ListPipelineOrders(ctx, &mallpb.ListPipelineOrdersRequest{
			Filters:  filters,
			Cursor:   cursor,
			PageSize: pageSize,
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

		nextCursor, done, guardErr := guard.next(cursor, ordersResp.GetHasMore(), ordersResp.GetNextCursor())
		if guardErr != nil {
			return nil, gstatus.Error(codes.Internal, guardErr.Error())
		}
		if done {
			return nil, nil
		}
		cursor = nextCursor
	}
}
