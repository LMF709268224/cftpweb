package handler

import (
	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// ListPipelines  GET /api/mall/pipelines
func (h *Handler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	catalogsResp, err := h.Gcc.ListCatalogs(r.Context(), &emptypb.Empty{})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListPipelinesRsp{
		Pipelines: make([]PipelineConfig, 0),
	}

	for _, catalog := range catalogsResp.GetCatalogs() {
		resp, err := h.Gcc.ListPipelines(r.Context(), &gccpb.ListPipelinesRequest{
			CategoryId:  catalog.GetCatalogId(),
			OnlyCurrent: true,
		})
		if err != nil {
			slog.Error("Failed to list pipelines", "error", err, "catalog_id", catalog.GetCatalogId())
			continue
		}

		for _, pipeline := range resp.GetPipelines() {
			// 获取结业资格
			finalEligibilityResp, err := h.Gcc.GetPipelineFinalEligibility(r.Context(), &gccpb.GetPipelineFinalEligibilityRequest{
				PipelineId: pipeline.GetPipelineId(),
			})
			if err != nil {
				slog.Error("Failed to get pipeline final eligibility", "error", err, "pipeline_id", pipeline.GetPipelineId())
				continue
			}

			out.Pipelines = append(out.Pipelines, toPipelineConfig(pipeline, finalEligibilityResp.GetCertQuals()))
		}

		//TODO 获取单个管线，看看是否已购买
	}

	WriteJSON(w, http.StatusOK, out)
}

// GetPipelineDetail  GET /api/mall/pipelines/{id}
func (h *Handler) GetPipelineDetail(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := chi.URLParam(r, "pipelineId")
	ctx := r.Context()

	// 1. 获取静态配置
	gccResp, err := h.Gcc.GetPipeline(ctx, &gccpb.GetPipelineRequest{
		Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineID},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := PipelineDetailRsp{
		Config: toPipelineConfig(gccResp, nil),
	}

	// 2. 尝试获取考生的实例进度（如果已购买）
	progResp, err := h.Gprog.ListCandidatePipelines(ctx, &gprogpb.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err == nil {
		for _, p := range progResp.GetPipelines() {
			// 重要：使用查出来的配置 ID (PipelineId) 进行匹配
			if p.GetPipelineCcUlid() == gccResp.GetPipelineId() {
				out.Instance = toPipelineSummary(p)
				break
			}
		}
	}

	WriteJSON(w, http.StatusOK, out)
}

// PurchasePipeline  POST /api/mall/pipelines/{pipelineId}/purchase
func (h *Handler) PurchasePipeline(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := chi.URLParam(r, "pipelineId")

	var req PurchasePipelineReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}

	// 1. Verify Pipeline Configuration exists
	_, err := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
		Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineID},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	// 2. Create Order in gmall
	mallResp, err := h.Mall.CreatePipelineOrder(r.Context(), &mallpb.CreatePipelineOrderRequest{
		CandidateUlid:                   candidateID,
		PipelineCcUlid:                  pipelineID,
		PaymentMode:                     req.PaymentMode,
		CandidateSelectedExemptionsJson: req.CandidateSelectedExemptionsJson,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	// 3. Return response with PaymentUrl
	WriteJSON(w, http.StatusCreated, PurchasePipelineRsp{
		PipelineOrderUlid:    mallResp.GetPipelineOrderUlid(),
		OrderStatus:          mallResp.GetOrderStatus(),
		ReviewOrderUlid:      mallResp.GetReviewOrderUlid(),
		PipelinePayOrderUlid: mallResp.GetPipelinePayOrderUlid(),
		PaymentUrl:           mallResp.GetPaymentUrl(),
	})
}

func toPipelineConfig(p *gccpb.PipelineConfig, certQuals []*gccpb.Qualification) PipelineConfig {
	if p == nil {
		return PipelineConfig{}
	}

	return PipelineConfig{
		PipelineId:             p.GetPipelineId(),
		PipelineGuid:           p.GetPipelineGuid(),
		Version:                p.GetVersion(),
		Name:                   p.GetName(),
		UnlockStripeProductId:  p.GetUnlockStripeProductId(),
		UnlockStripePriceId:    p.GetUnlockStripePriceId(),
		PackageStripeProductId: p.GetPackageStripeProductId(),
		PackageStripePriceId:   p.GetPackageStripePriceId(),
		UnlockQuals:            toUnlockQuals(p.GetUnlockQuals()),
		CertQuals:              toUnlockQuals(p.GetCertQuals()),
		Stages:                 toStages(p.GetStages()),
		Status:                 p.GetStatus(),
		IsCurrent:              p.GetIsCurrent(),
		CreatedAt:              p.GetCreatedAt(),
		FinalQuals:             toUnlockQuals(certQuals),
	}
}

func toUnlockQuals(quals []*gccpb.Qualification) []Qualification {
	if quals == nil {
		return nil
	}

	out := make([]Qualification, 0, len(quals))
	for _, qual := range quals {
		out = append(out, Qualification{
			QualId:   qual.GetQualId(),
			NameHint: qual.GetNameHint(),
		})
	}
	return out
}

func toStages(stages []*gccpb.StageConfig) []StageConfig {
	if stages == nil {
		return nil
	}

	out := make([]StageConfig, 0, len(stages))
	for _, stage := range stages {
		out = append(out, StageConfig{
			StageId:   stage.GetStageId(),
			Name:      stage.GetName(),
			SortOrder: stage.GetSortOrder(),
			Units:     toUnits(stage.GetUnits()),
		})
	}
	return out
}

func toUnits(units []*gccpb.UnitConfig) []UnitConfig {
	if units == nil {
		return nil
	}

	out := make([]UnitConfig, 0, len(units))
	for _, unit := range units {
		out = append(out, UnitConfig{
			UnitId:                   unit.GetUnitId(),
			ExemptionQuals:           unit.GetExemptionQuals(),
			AllowRetake:              unit.GetAllowRetake(),
			StripeProductId:          unit.GetStripeProductId(),
			StripePriceId:            unit.GetStripePriceId(),
			ExemptionStripeProductId: unit.GetExemptionStripeProductId(),
			ExemptionStripePriceId:   unit.GetExemptionStripePriceId(),
			RetakeStripeProductId:    unit.GetRetakeStripeProductId(),
			RetakeStripePriceId:      unit.GetRetakeStripePriceId(),
			GlmsCourseId:             unit.GetGlmsCourseId(),
		})
	}
	return out
}
