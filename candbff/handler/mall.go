package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strings"

	gccpb "github.com/LMF709268224/cftpproto/gcc"
	gcredspb "github.com/LMF709268224/cftpproto/gcreds"
	lmspb "github.com/LMF709268224/cftpproto/glms"
	mallpb "github.com/LMF709268224/cftpproto/gmall"
	gprog "github.com/LMF709268224/cftpproto/gprog"
	gprogpb "github.com/LMF709268224/cftpproto/gprog"

	"github.com/go-chi/chi/v5"
)

// ListPipelines  GET /api/mall/pipelines
func (h *Handler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gcc.ListPipelines(r.Context(), &gccpb.ListPipelinesRequest{
		CandidateUlid: CandidateID(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListPipelinesRsp{
		Pipelines: make([]PipelineConfig, 0, len(resp.GetPipelines())),
	}

	// TODO: wait for GCC catalog management/list API to support grouped browsing.
	for _, pipeline := range resp.GetPipelines() {
		var pipelineForOutput *gccpb.PipelineConfig
		detailResp, err := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
			Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipeline.GetPipelineId()},
		})
		if err != nil {
			slog.Warn("Failed to get pipeline detail for mall list", "error", err, "pipeline_id", pipeline.GetPipelineId())
		} else {
			pipelineForOutput = detailResp
		}
		if pipelineForOutput == nil {
			pipelineForOutput = pipelineSummaryToConfig(pipeline)
		}

		finalEligibilityResp, err := h.Gcc.GetPipelineFinalEligibility(r.Context(), &gccpb.GetPipelineFinalEligibilityRequest{
			PipelineId: pipeline.GetPipelineId(),
		})
		if err != nil {
			slog.Error("Failed to get pipeline final eligibility", "error", err, "pipeline_id", pipeline.GetPipelineId())
			continue
		}

		config := toPipelineConfig(pipelineForOutput, finalEligibilityResp.GetCerts())
		if count, ok := h.pipelinePurchaseCount(r, config.PipelineId); ok {
			config.PurchaseCount = &count
		}
		out.Pipelines = append(out.Pipelines, config)
	}

	WriteJSON(w, http.StatusOK, out)
}

func (h *Handler) pipelinePurchaseCount(r *http.Request, pipelineID string) (int32, bool) {
	pipelineID = strings.TrimSpace(pipelineID)
	if pipelineID == "" {
		return 0, false
	}
	resp, err := h.Gprog.ListPipelines(r.Context(), &gprog.ListPipelinesReq{
		PipelineCcUlid: pipelineID,
		Limit:          1,
	})
	if err != nil {
		slog.Warn("Failed to get pipeline purchase count", "error", err, "pipeline_id", pipelineID)
		return 0, false
	}
	return resp.GetTotal(), true
}

// GetPipelineDetail  GET /api/mall/pipelines/{id}
func (h *Handler) GetPipelineDetail(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	if !requireRequestField(w, pipelineID, "pipeline_id") {
		return
	}
	ctx := r.Context()
	// 1. load static pipeline config
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

	progResp, err := h.Gprog.ListCandidatePipelines(ctx, &gprogpb.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err == nil {
		for _, p := range progResp.GetPipelines() {
			if p.GetPipelineCcUlid() == gccResp.GetPipelineId() {
				out.Instance = toPipelineSummary(p)
				break
			}
		}
	}

	WriteJSON(w, http.StatusOK, out)
}

// GetPipelineRuntime GET /api/mall/pipelines/{id}/runtime
func (h *Handler) GetPipelineRuntime(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	if !requireRequestField(w, pipelineID, "pipeline_id") {
		return
	}
	ctx := r.Context()

	gccResp, err := h.Gcc.GetPipeline(ctx, &gccpb.GetPipelineRequest{
		Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineID},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := PipelineRuntimeRsp{
		Config: toPipelineConfig(gccResp, nil),
	}

	progResp, err := h.Gprog.ListCandidatePipelines(ctx, &gprogpb.ListCandidatePipelinesReq{
		CandidateUlid: candidateID,
	})
	if err != nil {
		WriteJSON(w, http.StatusOK, out)
		return
	}

	for _, p := range progResp.GetPipelines() {
		if p.GetPipelineCcUlid() != gccResp.GetPipelineId() {
			continue
		}
		out.Instance = toPipelineSummary(p)
		out.PipelineStatus = p.GetStatus()
		out.CurrentStageUlid = strings.TrimSpace(p.GetCurrentStageUlid())
		runtimeResp, runtimeErr := h.Gprog.GetPipelineDetail(ctx, &gprog.GetPipelineDetailReq{
			PipelineUlid: p.GetPipelineUlid(),
		})
		if runtimeErr == nil {
			mergeRuntimeStatuses(&out.Config, runtimeResp)
			out.NextStep = buildPipelineNextStep(runtimeResp, gccResp, p)
			if runtimeResp.GetPipeline() != nil {
				out.PipelineStatus = runtimeResp.GetPipeline().GetStatus()
				out.CurrentStageUlid = strings.TrimSpace(runtimeResp.GetPipeline().GetCurrentStageUlid())
			}
			if stageDetails := runtimeResp.GetStages(); len(stageDetails) > 0 {
				for _, stage := range stageDetails {
					if stage == nil || stage.GetStage() == nil {
						continue
					}
					if out.CurrentStageUlid != "" && stage.GetStage().GetStageUlid() == out.CurrentStageUlid {
						out.CurrentStageName = stageConfigNameByID(gccResp, stage.GetStage().GetStageCcUlid())
						out.CurrentStageStatus = stage.GetStage().GetStatus()
						for _, unit := range stage.GetCourseUnits() {
							if unit == nil {
								continue
							}
							if unit.GetStatus() != gprog.CourseUnitStatus_COURSE_UNIT_STATUS_COMPLETED {
								out.CurrentUnitStatus = unit.GetStatus()
								break
							}
						}
						break
					}
				}
			}
		}
		break
	}

	if out.NextStep.Action == "" {
		out.NextStep = buildPipelineNextStep(nil, gccResp, nil)
	}

	WriteJSON(w, http.StatusOK, out)
}

func mergeRuntimeStatuses(config *PipelineConfig, runtime *gprog.GetPipelineDetailRsp) {
	if config == nil || runtime == nil {
		return
	}
	stageIndexes := make(map[string]int, len(config.Stages))
	for index := range config.Stages {
		stageID := strings.TrimSpace(config.Stages[index].StageId)
		if stageID == "" {
			continue
		}
		stageIndexes[stageID] = index
	}

	for _, stageDetail := range runtime.GetStages() {
		if stageDetail == nil || stageDetail.GetStage() == nil {
			continue
		}
		stageCcID := strings.TrimSpace(stageDetail.GetStage().GetStageCcUlid())
		stageIndex, ok := stageIndexes[stageCcID]
		if !ok {
			continue
		}
		config.Stages[stageIndex].RuntimeStatus = stageDetail.GetStage().GetStatus()

		unitIndexes := make(map[string]int, len(config.Stages[stageIndex].Units))
		for unitIndex := range config.Stages[stageIndex].Units {
			unitID := strings.TrimSpace(config.Stages[stageIndex].Units[unitIndex].UnitId)
			if unitID == "" {
				continue
			}
			unitIndexes[unitID] = unitIndex
		}
		for _, unit := range stageDetail.GetCourseUnits() {
			if unit == nil {
				continue
			}
			unitCcID := strings.TrimSpace(unit.GetCourseUnitCcUlid())
			unitIndex, ok := unitIndexes[unitCcID]
			if !ok {
				continue
			}
			config.Stages[stageIndex].Units[unitIndex].RuntimeStatus = unit.GetStatus()
		}
	}
}

// GetMallCourseSummary GET /api/mall/courses/{courseId}
func (h *Handler) GetMallCourseSummary(w http.ResponseWriter, r *http.Request) {
	courseID := strings.TrimSpace(chi.URLParam(r, "courseId"))
	if !requireRequestField(w, courseID, "course_id") {
		return
	}

	resp, err := h.Lms.GetCourseSummary(r.Context(), &lmspb.GetCourseSummaryCandidateRequest{
		CandidateId: CandidateID(r),
		CourseId:    courseID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetMallCourseThumbnailURL GET /api/mall/courses/{courseId}/thumbnail-url
func (h *Handler) GetMallCourseThumbnailURL(w http.ResponseWriter, r *http.Request) {
	courseID := strings.TrimSpace(chi.URLParam(r, "courseId"))
	if !requireRequestField(w, courseID, "course_id") {
		return
	}

	summaryResp, err := h.Lms.GetCourseSummaryAdmin(r.Context(), &lmspb.GetCourseRequest{
		CourseId: courseID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	objectKey := strings.TrimSpace(summaryResp.GetCourse().GetThumbnailObjectKey())
	if objectKey == "" {
		WriteJSON(w, http.StatusOK, GetAccessURLRsp{})
		return
	}

	viewResp, err := h.Lms.CreateViewURLAdmin(r.Context(), &lmspb.CreateViewURLRequest{
		ObjectKey: objectKey,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL:       viewResp.GetViewUrl(),
		ExpiresAt: viewResp.GetExpiresAt(),
	})
}

// GetMallPipelineThumbnailURL GET /api/mall/pipelines/{pipelineId}/thumbnail-url
func (h *Handler) GetMallPipelineThumbnailURL(w http.ResponseWriter, r *http.Request) {
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	if !requireRequestField(w, pipelineID, "pipeline_id") {
		return
	}

	// pipelineResp, err := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
	// 	Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineID},
	// })
	// if err != nil {
	// 	HandleGrpcError(w, err)
	// 	return
	// }

	// objectKey := strings.TrimSpace(pipelineResp.GetThumbnailObjectKey())
	// if objectKey == "" {
	// 	WriteJSON(w, http.StatusOK, GetAccessURLRsp{})
	// 	return
	// }

	viewResp, err := h.Gcc.GetPublicURL(r.Context(), &gccpb.GetPublicURLRequest{
		PipelineId: pipelineID,
	})
	if err != nil {
		slog.Warn("Failed to get pipeline thumbnail url", "error", err, "pipeline_id", pipelineID)
		WriteJSON(w, http.StatusOK, GetAccessURLRsp{})
		return
	}

	WriteJSON(w, http.StatusOK, GetAccessURLRsp{
		URL: viewResp.GetPublicUrl(),
	})
}

// ListBundles GET /api/mall/bundles
func (h *Handler) ListBundles(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Mall.ListBundles(r.Context(), &mallpb.ListBundlesRequest{
		Limit:  100,
		Offset: 0,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	enrichedList := make([]map[string]interface{}, 0, len(resp.GetBundles()))
	for _, b := range resp.GetBundles() {
		enrichedList = append(enrichedList, h.enrichBundle(r.Context(), b))
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"bundles": enrichedList,
		"total":   resp.GetTotal(),
	})
}

// GetBundleDetail GET /api/mall/bundles/{bundleId}
func (h *Handler) GetBundleDetail(w http.ResponseWriter, r *http.Request) {
	bundleId := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if !requireRequestField(w, bundleId, "bundle_id") {
		return
	}
	resp, err := h.Mall.GetBundle(r.Context(), &mallpb.GetBundleRequest{
		Query: &mallpb.GetBundleRequest_BundleId{BundleId: bundleId},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, h.enrichBundle(r.Context(), resp.GetBundle()))
}

func (h *Handler) extractPipelineID(bundle *mallpb.BundleInfo) string {
	if bundle == nil {
		return ""
	}
	itemsJSON := bundle.GetItemsJson()
	if itemsJSON != "" {
		var list []map[string]interface{}
		if err := json.Unmarshal([]byte(itemsJSON), &list); err == nil {
			for _, item := range list {
				for _, key := range []string{"item_id", "id", "pipeline_id", "pipeline_cc_ulid"} {
					if idVal, ok := item[key].(string); ok && len(idVal) == 26 {
						return idVal
					}
				}
			}
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(itemsJSON), &obj); err == nil {
			if pps, ok := obj["pipelines"].([]interface{}); ok {
				for _, p := range pps {
					if idVal, ok := p.(string); ok && len(idVal) == 26 {
						return idVal
					}
				}
			}
			for _, key := range []string{"pipeline_id", "pipeline_cc_ulid", "pipeline"} {
				if idVal, ok := obj[key].(string); ok && len(idVal) == 26 {
					return idVal
				}
			}
		}
	}

	gpath := bundle.GetBundleGpath()
	if strings.HasPrefix(gpath, "/pipeline/") {
		parts := strings.Split(gpath, "/")
		if len(parts) > 2 && len(parts[2]) == 26 {
			return parts[2]
		}
	}
	re := regexp.MustCompile(`[0-9A-HJKMNP-TV-Z]{26}`)
	if match := re.FindString(itemsJSON); match != "" {
		return match
	}
	if match := re.FindString(gpath); match != "" {
		return match
	}
	return ""
}

func (h *Handler) enrichBundle(ctx context.Context, b *mallpb.BundleInfo) map[string]interface{} {
	m := map[string]interface{}{
		"bundle_id":            b.GetBundleId(),
		"bundle_gpath":         b.GetBundleGpath(),
		"version":              b.GetVersion(),
		"name":                 b.GetName(),
		"description":          b.GetDescription(),
		"items_json":           b.GetItemsJson(),
		"pricing_json":         b.GetPricingJson(),
		"thumbnail_object_key": b.GetThumbnailObjectKey(),
		"thumbnail_file_hash":  b.GetThumbnailFileHash(),
		"status":               b.GetStatus(),
		"is_current":           b.GetIsCurrent(),
		"created_at":           b.GetCreatedAt(),
		"updated_at":           b.GetUpdatedAt(),
		"stages":               []interface{}{},
		"final_quals":          []interface{}{},
		"category_tips":        "",
		"pipeline_id":          "",
	}

	pipelineID := h.extractPipelineID(b)
	if pipelineID != "" {
		m["pipeline_id"] = pipelineID
		pipeline, err := h.Gcc.GetPipeline(ctx, &gccpb.GetPipelineRequest{
			Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineID},
		})
		if err == nil && pipeline != nil {
			m["stages"] = toStages(pipeline.GetStages())
			m["final_quals"] = toUnlockQuals(pipeline.GetCertsQuals())
			m["category_tips"] = pipeline.GetCategoryTips()
		}
	}
	return m
}

// GetBundleThumbnailURL GET /api/mall/bundles/{bundleId}/thumbnail-url
func (h *Handler) GetBundleThumbnailURL(w http.ResponseWriter, r *http.Request) {
	bundleId := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if !requireRequestField(w, bundleId, "bundle_id") {
		return
	}
	resp, err := h.Mall.GetBundleThumbnailURL(r.Context(), &mallpb.GetBundleThumbnailURLRequest{
		BundleId: bundleId,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]string{
		"url": resp.GetPublicUrl(),
	})
}

// CreateBundleOrder POST /api/mall/bundles/{bundleId}/purchase
func (h *Handler) CreateBundleOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	bundleId := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if !requireRequestField(w, bundleId, "bundle_id") {
		return
	}

	var req CreateBundleOrderReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}

	resp, err := h.Mall.CreateBundleOrder(r.Context(), &mallpb.CreateBundleOrderRequest{
		CandidateUlid:          candidateID,
		BundleId:               bundleId,
		PaymentMode:            req.PaymentMode,
		SelectedExemptionsJson: req.SelectedExemptionsJson,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, resp)
}

func formatPaymentKey(paymentKey string) string {
	return paymentKey
}

func toPipelineConfig(p *gccpb.PipelineConfig, certQuals []*gccpb.Qualification) PipelineConfig {
	if p == nil {
		return PipelineConfig{}
	}

	return PipelineConfig{
		PipelineId:            p.GetPipelineId(),
		PipelineGuid:          "",
		Version:               p.GetVersion(),
		Name:                  p.GetName(),
		Description:           p.GetDescription(),
		CategoryTips:          p.GetCategoryTips(),
		ThumbnailObjectKey:    p.GetThumbnailObjectKey(),
		ThumbnailFileHash:     p.GetThumbnailFileHash(),
		UnlockStripeProductId: "",
		UnlockStripePriceId:   "",
		UnlockQuals:           toUnlockQuals(p.GetUnlockQuals()),
		CertQuals:             toUnlockQuals(p.GetCertsQuals()),
		Stages:                toStages(p.GetStages()),
		Status:                p.GetStatus(),
		IsCurrent:             p.GetIsCurrent(),
		CreatedAt:             p.GetCreatedAt(),
		FinalQuals:            toUnlockQuals(certQuals),
	}
}

func pipelineSummaryToConfig(p *gccpb.PipelineSummary) *gccpb.PipelineConfig {
	if p == nil {
		return nil
	}
	return &gccpb.PipelineConfig{
		PipelineId:         p.GetPipelineId(),
		Version:            p.GetVersion(),
		Name:               p.GetName(),
		Description:        p.GetDescription(),
		Status:             p.GetStatus(),
		IsCurrent:          p.GetIsCurrent(),
		CreatedAt:          p.GetCreatedAt(),
		CategoryTips:       p.GetCategoryTips(),
		ThumbnailObjectKey: p.GetThumbnailObjectKey(),
		ThumbnailFileHash:  p.GetThumbnailFileHash(),
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
			Name:                     unit.GetName(),
			ExemptionQuals:           unit.GetExemptionQuals(),
			AllowExemption:           unit.GetAllowExemption(),
			AllowRetake:              true,
			StripeProductId:          "",
			StripePriceId:            "",
			ExemptionStripeProductId: "",
			ExemptionStripePriceId:   "",
			RetakeStripeProductId:    "",
			RetakeStripePriceId:      "",
			GlmsCourseId:             unit.GetGlmsCourseId(),
			Program:                  unit.GetProgram(),
			ExamId:                   unit.GetExamId(),
			FormCode:                 unit.GetFormCode(),
		})
	}
	return out
}

func buildPipelineNextStep(runtime *gprogpb.GetPipelineDetailRsp, config *gccpb.PipelineConfig, instance *gprogpb.PipelineSummary) PipelineNextStep {
	out := PipelineNextStep{}
	if instance != nil {
		out.PipelineStatus = instance.GetStatus()
	}
	if out.PipelineStatus == gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED ||
		out.PipelineStatus == gprogpb.PipelineStatus_PIPELINE_STATUS_ISSUING_CERT {
		out.Action = "view_certificate"
		out.Message = "view the certificate"
		return out
	}
	if config == nil {
		return out
	}

	if runtime == nil || runtime.GetPipeline() == nil {
		firstUnit := firstConfigUnit(config)
		if firstUnit == nil {
			out.Action = "view_certificate"
			out.Message = "pipeline has no units"
			return out
		}
		fillNextStepFromUnit(&out, nil, firstUnit, "")
		if strings.TrimSpace(firstUnit.GetGlmsCourseId()) != "" {
			out.Action = "continue_learning"
			out.Message = "continue learning this course"
		} else {
			out.Action = "signup_exam"
			out.Message = "go to exams and sign up"
		}
		return out
	}

	stageDetails := runtime.GetStages()
	if len(stageDetails) > 0 {
		firstUnit := firstConfigUnit(config)
		for _, stage := range stageDetails {
			if stage == nil || stage.GetStage() == nil {
				continue
			}
			if stage.GetStage().GetStatus() == gprog.StageStatus_STAGE_STATUS_WAIT_CANDIDATE {
				if firstUnit != nil {
					fillNextStepFromUnit(&out, stage, firstUnit, stageConfigNameByID(config, stage.GetStage().GetStageCcUlid()))
				} else {
					out.StageId = stage.GetStage().GetStageUlid()
					out.StageName = stageConfigNameByID(config, stage.GetStage().GetStageCcUlid())
				}
				out.Action = "wait_candidate"
				out.Message = "stage is waiting for candidate action"
				out.Status = gprog.CourseUnitStatus_COURSE_UNIT_STATUS_UNSPECIFIED
				return out
			}
		}
	}
	if len(stageDetails) == 0 {
		firstUnit := firstConfigUnit(config)
		if firstUnit == nil {
			out.Action = "view_certificate"
			out.Message = "pipeline has no units"
			return out
		}
		fillNextStepFromUnit(&out, nil, firstUnit, "")
		out.Action = "continue_learning"
		out.Message = "continue learning this course"
		return out
	}

	currentStageUlid := ""
	if runtime.GetPipeline() != nil {
		currentStageUlid = strings.TrimSpace(runtime.GetPipeline().GetCurrentStageUlid())
	}

	pickStageIdx := -1
	for idx, stage := range stageDetails {
		if stage == nil || stage.GetStage() == nil {
			continue
		}
		if currentStageUlid != "" && stage.GetStage().GetStageUlid() == currentStageUlid {
			pickStageIdx = idx
			break
		}
	}
	if pickStageIdx < 0 {
		pickStageIdx = 0
	}

	pickStage := stageDetails[pickStageIdx]
	pickUnit := pickNextRuntimeUnit(pickStage)
	if pickUnit == nil {
		for idx := pickStageIdx + 1; idx < len(stageDetails); idx++ {
			pickUnit = pickNextRuntimeUnit(stageDetails[idx])
			if pickUnit != nil {
				pickStage = stageDetails[idx]
				break
			}
		}
	}

	if pickUnit == nil {
		if runtime.GetPipeline().GetStatus() == gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED {
			out.Action = "view_certificate"
			out.Message = "pipeline completed"
		} else {
			firstUnit := firstConfigUnit(config)
			if firstUnit != nil {
				fillNextStepFromUnit(&out, nil, firstUnit, "")
			}
			out.Action = "signup_exam"
			out.Message = "go to exams and sign up"
		}
		return out
	}

	fillNextStepFromUnit(&out, pickStage, configUnitByID(config, pickUnit.GetCourseUnitCcUlid()), stageConfigNameByID(config, pickStage.GetStage().GetStageCcUlid()))
	out.CourseUnitUlid = pickUnit.GetCourseUnitUlid()
	out.CourseUnitCcUlid = pickUnit.GetCourseUnitCcUlid()
	out.Status = pickUnit.GetStatus()
	switch pickUnit.GetStatus() {
	case gprog.CourseUnitStatus_COURSE_UNIT_STATUS_WAITING_STUDY:
		out.Action = "continue_learning"
		out.Message = "continue learning this course"
	case gprog.CourseUnitStatus_COURSE_UNIT_STATUS_WAITING_SIGNUP_EXAM:
		out.Action = "signup_exam"
		out.Message = "open exam page and sign up"
	case gprog.CourseUnitStatus_COURSE_UNIT_STATUS_EXAM_OPEN:
		out.Action = "schedule_exam"
		out.Message = "schedule the exam"
	case gprog.CourseUnitStatus_COURSE_UNIT_STATUS_EXAM_SCHEDULED:
		out.Action = "view_exam_schedule"
		out.Message = "view the scheduled exam"
	case gprog.CourseUnitStatus_COURSE_UNIT_STATUS_EXAM_FAILED:
		if out.AllowRetake {
			out.Action = "apply_retake"
			out.Message = "apply for a retake"
		} else {
			out.Action = "view_exam_result"
			out.Message = "view the exam result"
		}
	case gprog.CourseUnitStatus_COURSE_UNIT_STATUS_COMPLETED:
		if runtime.GetPipeline().GetStatus() == gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED {
			out.Action = "view_certificate"
			out.Message = "view the certificate"
		} else {
			out.Action = "signup_exam"
			out.Message = "go to exams and sign up"
		}
	default:
		out.Action = "continue_learning"
		out.Message = "continue learning this course"
	}

	return out
}

func pickNextRuntimeUnit(stage *gprogpb.StageDetail) *gprogpb.CourseUnitSummary {
	if stage == nil {
		return nil
	}
	units := stage.GetCourseUnits()
	if len(units) == 0 {
		return nil
	}
	for _, unit := range units {
		if unit == nil {
			continue
		}
		if unit.GetStatus() != gprog.CourseUnitStatus_COURSE_UNIT_STATUS_COMPLETED {
			return unit
		}
	}
	return nil
}

func fillNextStepFromUnit(out *PipelineNextStep, stage *gprogpb.StageDetail, unit *gccpb.UnitConfig, stageName string) {
	if out == nil || unit == nil {
		return
	}
	out.CourseUnitUlid = unit.GetUnitId()
	out.CourseId = unit.GetGlmsCourseId()
	out.AllowRetake = unit.GetExamId() != ""
	out.AllowExemption = unit.GetAllowExemption()
	out.Program = unit.GetProgram()
	out.ExamId = unit.GetExamId()
	out.FormCode = unit.GetFormCode()
	if stage != nil && stage.GetStage() != nil {
		out.StageId = stage.GetStage().GetStageUlid()
		out.StageName = stageName
	}
}

func stageConfigNameByID(config *gccpb.PipelineConfig, stageID string) string {
	if config == nil {
		return ""
	}
	stageID = strings.TrimSpace(stageID)
	if stageID == "" {
		return ""
	}
	for _, stage := range config.GetStages() {
		if stage == nil {
			continue
		}
		if strings.TrimSpace(stage.GetStageId()) == stageID {
			return strings.TrimSpace(stage.GetName())
		}
	}
	return ""
}

func firstConfigUnit(config *gccpb.PipelineConfig) *gccpb.UnitConfig {
	if config == nil {
		return nil
	}
	for _, stage := range config.GetStages() {
		if stage == nil {
			continue
		}
		for _, unit := range stage.GetUnits() {
			if unit != nil && strings.TrimSpace(unit.GetUnitId()) != "" {
				return unit
			}
		}
	}
	return nil
}

func configUnitByID(config *gccpb.PipelineConfig, unitID string) *gccpb.UnitConfig {
	if config == nil {
		return nil
	}
	for _, stage := range config.GetStages() {
		if stage == nil {
			continue
		}
		for _, unit := range stage.GetUnits() {
			if unit == nil {
				continue
			}
			if unit.GetUnitId() == unitID {
				return unit
			}
		}
	}
	return nil
}

// CheckPipelineEligibility GET /api/mall/pipelines/{pipelineId}/eligibility
func (h *Handler) CheckPipelineEligibility(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	if !requireRequestField(w, pipelineID, "pipeline_id") {
		return
	}

	resp, err := h.Mall.CheckPipelineEligibility(r.Context(), &mallpb.CheckPipelineEligibilityRequest{
		CandidateUlid:  candidateID,
		PipelineCcUlid: pipelineID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// GetPipelineExemptionOptions GET /api/mall/pipelines/{pipelineId}/exemptions
func (h *Handler) GetPipelineExemptionOptions(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	if !requireRequestField(w, pipelineID, "pipeline_id") {
		return
	}

	pipeline, err := h.Gcc.GetPipeline(r.Context(), &gccpb.GetPipelineRequest{
		Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: pipelineID},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := PipelineExemptionOptionsRsp{
		Stages: make([]PipelineExemptionStage, 0, len(pipeline.GetStages())),
	}
	defCache := map[string]*gcredspb.CredentialDefinition{}
	checkCache := map[string]*gcredspb.CheckCandidateQualificationResponse{}

	for stageIndex, stage := range pipeline.GetStages() {
		stageOut := PipelineExemptionStage{
			Index:     int32(stageIndex),
			StageId:   stage.GetStageId(),
			StageName: stage.GetName(),
			SortOrder: stage.GetSortOrder(),
			Units:     []PipelineExemptionUnit{},
		}
		for _, unit := range stage.GetUnits() {
			qualIDs := compactStrings(unit.GetExemptionQuals())
			if !unit.GetAllowExemption() || len(qualIDs) == 0 {
				continue
			}
			unitOut := PipelineExemptionUnit{
				UnitId:         unit.GetUnitId(),
				UnitName:       unit.GetName(),
				AllowExemption: true,
				ExemptionQuals: make([]PipelineExemptionQual, 0, len(qualIDs)),
			}
			for _, qualID := range qualIDs {
				def := getCachedCredentialDefinition(r, h, defCache, qualID)
				check := getCachedCandidateQualification(r, h, checkCache, candidateID, qualID)
				qualOut := PipelineExemptionQual{
					QualId: qualID,
				}
				if def != nil {
					qualOut.Name = def.GetName()
					qualOut.Description = def.GetDescription()
					qualOut.Category = def.GetCategory()
				}
				if qualOut.Name == "" {
					qualOut.Name = qualID
				}
				if check != nil {
					qualOut.Eligible = check.GetEligible()
					qualOut.CredentialStatus = check.GetCredentialStatus().String()
					qualOut.Message = check.GetMessage()
				} else {
					qualOut.Message = "qualification status unavailable"
				}
				if qualOut.Eligible {
					unitOut.Qualified = true
				}
				unitOut.ExemptionQuals = append(unitOut.ExemptionQuals, qualOut)
			}
			if unitOut.Qualified {
				unitOut.Message = "eligible for exemption"
			} else {
				unitOut.Message = "missing active qualification for exemption"
			}
			stageOut.Units = append(stageOut.Units, unitOut)
		}
		if len(stageOut.Units) > 0 {
			out.Stages = append(out.Stages, stageOut)
		}
	}

	WriteJSON(w, http.StatusOK, out)
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	return out
}

func getCachedCredentialDefinition(r *http.Request, h *Handler, cache map[string]*gcredspb.CredentialDefinition, qualID string) *gcredspb.CredentialDefinition {
	if def, ok := cache[qualID]; ok {
		return def
	}
	def, err := h.Creds.GetCredentialDefinitionDetail(r.Context(), &gcredspb.GetCredentialDefinitionDetailRequest{
		CredDefId: qualID,
	})
	if err != nil {
		slog.Warn("Failed to load credential definition for exemption option", "error", err, "qual_id", qualID)
		cache[qualID] = nil
		return nil
	}
	cache[qualID] = def
	return def
}

func getCachedCandidateQualification(r *http.Request, h *Handler, cache map[string]*gcredspb.CheckCandidateQualificationResponse, candidateID string, qualID string) *gcredspb.CheckCandidateQualificationResponse {
	if check, ok := cache[qualID]; ok {
		return check
	}
	check, err := h.Creds.CheckCandidateQualification(r.Context(), &gcredspb.CheckCandidateQualificationRequest{
		CandidateId: candidateID,
		CredDefId:   qualID,
	})
	if err != nil {
		slog.Warn("Failed to check candidate qualification for exemption option", "error", err, "candidate_id", candidateID, "qual_id", qualID)
		cache[qualID] = nil
		return nil
	}
	cache[qualID] = check
	return check
}

// UnlockPipeline POST /api/mall/pipelines/{pipelineId}/unlock
func (h *Handler) UnlockPipeline(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	pipelineID := strings.TrimSpace(chi.URLParam(r, "pipelineId"))
	if !requireRequestField(w, pipelineID, "pipeline_id") {
		return
	}

	resp, err := h.Mall.CreatePipelineUnlockOrder(r.Context(), &mallpb.CreatePipelineUnlockOrderRequest{
		CandidateUlid:  candidateID,
		PipelineCcUlid: pipelineID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	resp.PaymentKey = formatPaymentKey(resp.GetPaymentKey())
	WriteJSON(w, http.StatusCreated, resp)
}

// GetActiveBundleOrder GET /api/mall/bundles/{bundleId}/active-order
func (h *Handler) GetActiveBundleOrder(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	bundleId := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if !requireRequestField(w, bundleId, "bundle_id") {
		return
	}

	resp, err := h.Mall.ListBundleOrders(r.Context(), &mallpb.ListBundleOrdersRequest{
		CandidateUlid: candidateID,
		BundleId:      bundleId,
		Limit:         20,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	for _, item := range resp.GetItems() {
		status := strings.ToUpper(strings.TrimSpace(item.GetOrderStatus()))
		if strings.Contains(status, "COMPLETED") || strings.Contains(status, "CANCEL") || strings.Contains(status, "FAILED") {
			continue
		}
		detailResp, err := h.Mall.GetBundleOrderDetail(r.Context(), &mallpb.GetBundleOrderDetailRequest{
			BundleOrderUlid: item.GetBundleOrderUlid(),
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		if detailResp.GetFound() {
			WriteJSON(w, http.StatusOK, detailResp.GetDetail())
			return
		}
		WriteJSON(w, http.StatusOK, item)
		return
	}

	WriteError(w, http.StatusNotFound, ErrNotFound, "未找到未完成订单")
}

// UnlockPipelineInBundle POST /api/mall/bundles/{bundleId}/unlock
func (h *Handler) UnlockPipelineInBundle(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	bundleId := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if !requireRequestField(w, bundleId, "bundle_id") {
		return
	}

	var req UnlockPipelineInBundleReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.PipelineCcUlid = strings.TrimSpace(req.PipelineCcUlid)
	if req.PipelineCcUlid == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "field 'pipeline_cc_ulid' is required")
		return
	}

	resp, err := h.Mall.CreatePipelineUnlockOrder(r.Context(), &mallpb.CreatePipelineUnlockOrderRequest{
		CandidateUlid:  candidateID,
		PipelineCcUlid: req.PipelineCcUlid,
		BundleId:       bundleId,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// PreviewPayment POST /api/mall/payments/preview
func (h *Handler) PreviewPayment(w http.ResponseWriter, r *http.Request) {
	var req PreviewPaymentReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.BizType = strings.TrimSpace(req.BizType)
	req.BizRefUlid = strings.TrimSpace(req.BizRefUlid)
	if !requireRequestField(w, req.BizType, "biz_type") || !requireRequestField(w, req.BizRefUlid, "biz_ref_ulid") {
		return
	}

	bizType := req.BizType
	if bizType == "PIPELINE_PAYMENT" {
		bizType = "BUNDLE_PURCHASE"
	}

	resp, err := h.Mall.PreviewPayment(r.Context(), &mallpb.PreviewPaymentRequest{
		BizType:     bizType,
		BizRefUlid:  req.BizRefUlid,
		CouponCodes: req.CouponCodes,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

// InitiatePayment POST /api/mall/payments/initiate
func (h *Handler) InitiatePayment(w http.ResponseWriter, r *http.Request) {
	var req InitiatePaymentReq
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.BizType = strings.TrimSpace(req.BizType)
	req.BizRefUlid = strings.TrimSpace(req.BizRefUlid)
	if !requireRequestField(w, req.BizType, "biz_type") || !requireRequestField(w, req.BizRefUlid, "biz_ref_ulid") {
		return
	}

	bizType := req.BizType
	if bizType == "PIPELINE_PAYMENT" {
		bizType = "BUNDLE_PURCHASE"
	}

	resp, err := h.Mall.InitiatePayment(r.Context(), &mallpb.InitiatePaymentRequest{
		BizType:     bizType,
		BizRefUlid:  req.BizRefUlid,
		SuccessUrl:  strings.TrimSpace(req.SuccessUrl),
		CancelUrl:   strings.TrimSpace(req.CancelUrl),
		CouponCodes: req.CouponCodes,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	resp.PaymentKey = formatPaymentKey(resp.GetPaymentKey())
	WriteJSON(w, http.StatusOK, resp)
}
