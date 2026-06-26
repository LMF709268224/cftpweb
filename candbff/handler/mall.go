package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	gprog "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"

	"github.com/go-chi/chi/v5"
	"github.com/oklog/ulid/v2"
)

type bundleEligibilityBlocker struct {
	BlockerType string   `json:"blocker_type,omitempty"`
	Description string   `json:"description,omitempty"`
	Details     []string `json:"details,omitempty"`
}

type bundleEligibilitySummary struct {
	Eligible    bool                       `json:"eligible"`
	CanUnlock   bool                       `json:"can_unlock"`
	CanPurchase bool                       `json:"can_purchase"`
	Blockers    []bundleEligibilityBlocker `json:"blockers,omitempty"`
}

type bundleEnrichmentState struct {
	candidateID              string
	membershipHistory        []*gmbrpb.UserMembership
	activeMembershipByGpath  map[string]*gmbrpb.UserMembership
	loadedMembershipsByGpath map[string]bool
}

type bundleActiveOrderSummary struct {
	Action     string `json:"action"`
	OrderID    string `json:"order_id"`
	Status     string `json:"status,omitempty"`
	PayOrderID string `json:"pay_order_id,omitempty"`
	Message    string `json:"message,omitempty"`
}

type bundlePaymentPreviewSummary struct {
	Subtotal      int64  `json:"subtotal"`
	DiscountTotal int64  `json:"discount_total"`
	TaxTotal      int64  `json:"tax_total"`
	Total         int64  `json:"total"`
	Currency      string `json:"currency,omitempty"`
}

type bundlePurchaseState struct {
	Eligibility      bundleEligibilitySummary     `json:"eligibility"`
	ActiveOrder      *bundleActiveOrderSummary    `json:"active_order,omitempty"`
	PaymentPreview   *bundlePaymentPreviewSummary `json:"payment_preview,omitempty"`
	ExemptionOptions *PipelineExemptionOptionsRsp `json:"exemption_options,omitempty"`
}

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
			Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipeline.GetPipelineUlid()},
		})
		if err != nil {
			slog.Warn("Failed to get pipeline detail for mall list", "error", err, "pipeline_id", pipeline.GetPipelineUlid())
		} else {
			pipelineForOutput = detailResp
		}
		if pipelineForOutput == nil {
			pipelineForOutput = pipelineSummaryToConfig(pipeline)
		}

		finalEligibilityResp, err := h.Gcc.GetPipelineFinalEligibility(r.Context(), &gccpb.GetPipelineFinalEligibilityRequest{
			PipelineUlid: pipeline.GetPipelineUlid(),
		})
		if err != nil {
			slog.Error("Failed to get pipeline final eligibility", "error", err, "pipeline_id", pipeline.GetPipelineUlid())
			continue
		}

		config := toPipelineConfig(pipelineForOutput, finalEligibilityResp.GetCerts())
		if count, ok := h.pipelinePurchaseCount(r, config.PipelineUlid); ok {
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
		Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipelineID},
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
			if p.GetPipelineCcUlid() == gccResp.GetPipelineUlid() {
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
		Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipelineID},
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
		if p.GetPipelineCcUlid() != gccResp.GetPipelineUlid() {
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
		stageID := strings.TrimSpace(config.Stages[index].StageUlid)
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
			unitID := strings.TrimSpace(config.Stages[stageIndex].Units[unitIndex].UnitUlid)
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
			config.Stages[stageIndex].Units[unitIndex].CourseUnitUlid = unit.GetCourseUnitUlid()
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
		CandidateUlid: CandidateID(r),
		CourseUlid:    courseID,
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

	summaryResp, err := h.Lms.GetCourseSummary(r.Context(), &lmspb.GetCourseSummaryCandidateRequest{
		CandidateUlid: CandidateID(r),
		CourseUlid:    courseID,
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

	viewResp, err := h.Lms.CreateViewURL(r.Context(), &lmspb.CreateViewURLCandidateRequest{
		CandidateUlid: CandidateID(r),
		ObjectKey:     objectKey,
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
	// 	Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipelineID},
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
		PipelineUlid: pipelineID,
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
	candidateID := CandidateID(r)
	page := parsePositiveIntQuery(r, "page", 1)
	pageSize := parsePositiveIntQuery(r, "page_size", parsePositiveIntQuery(r, "limit", 20))
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize
	if r.URL.Query().Get("offset") != "" {
		offset = parseNonNegativeIntQuery(r, "offset", offset)
	}
	status := strings.TrimSpace(r.URL.Query().Get("status"))

	resp, err := h.Mall.ListBundles(r.Context(), &mallpb.ListBundlesRequest{
		Limit:  int32(pageSize),
		Offset: int32(offset),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	bundles := resp.GetBundles()
	if status != "" {
		filtered := make([]*mallpb.BundleInfo, 0, len(bundles))
		for _, b := range bundles {
			if strings.EqualFold(strings.TrimSpace(b.GetStatus()), status) {
				filtered = append(filtered, b)
			}
		}
		bundles = filtered
	}

	state := h.newBundleEnrichmentState(r.Context(), candidateID)
	enrichedList := make([]map[string]interface{}, 0, len(bundles))
	for _, b := range bundles {
		enrichedList = append(enrichedList, h.enrichBundle(r.Context(), b, state))
	}
	total := int(resp.GetTotal())
	if status != "" {
		total = len(bundles)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"bundles":     enrichedList,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"offset":      offset,
		"total_pages": totalPages(total, pageSize),
	})
}

// GetBundleDetail GET /api/mall/bundles/{bundleId}
func (h *Handler) GetBundleDetail(w http.ResponseWriter, r *http.Request) {
	bundleId := strings.TrimSpace(chi.URLParam(r, "bundleId"))
	if !requireRequestField(w, bundleId, "bundle_id") {
		return
	}
	resp, err := h.Mall.GetBundle(r.Context(), &mallpb.GetBundleRequest{
		Query: &mallpb.GetBundleRequest_BundleUlid{BundleUlid: bundleId},
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, h.enrichBundle(r.Context(), resp.GetBundle(), h.newBundleEnrichmentState(r.Context(), CandidateID(r))))
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
				hasPipelineType := isPipelineBundleItem(item)
				for _, key := range []string{"pipeline_id", "pipeline_cc_ulid"} {
					if idVal := mapString(item, key); looksLikeULID(idVal) {
						return idVal
					}
				}
				if !hasPipelineType {
					continue
				}
				for _, key := range []string{"ref_ulid", "item_id", "id"} {
					if idVal := mapString(item, key); looksLikeULID(idVal) {
						return idVal
					}
				}
			}
		}

		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(itemsJSON), &obj); err == nil {
			if pps, ok := obj["pipelines"].([]interface{}); ok {
				for _, p := range pps {
					if idVal, ok := p.(string); ok && looksLikeULID(idVal) {
						return idVal
					}
					if item, ok := p.(map[string]interface{}); ok {
						for _, key := range []string{"ref_ulid", "item_id", "id", "pipeline_id", "pipeline_cc_ulid"} {
							if idVal := mapString(item, key); looksLikeULID(idVal) {
								return idVal
							}
						}
					}
				}
			}
			for _, key := range []string{"pipeline_id", "pipeline_cc_ulid", "pipeline"} {
				if idVal := mapString(obj, key); looksLikeULID(idVal) {
					return idVal
				}
			}
		}
	}

	gpath := bundle.GetBundleGpath()
	if strings.HasPrefix(gpath, "/pipeline/") {
		parts := strings.Split(gpath, "/")
		if len(parts) > 2 && looksLikeULID(parts[2]) {
			return parts[2]
		}
	}
	return ""
}

func looksLikeULID(value string) bool {
	return len(strings.TrimSpace(value)) == 26
}

func mapString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	value, ok := m[key]
	if !ok {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return strings.TrimSpace(v.String())
	default:
		return ""
	}
}

func normalizeBundleItemType(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "-", "_")
	return value
}

func isPipelineBundleItem(item map[string]interface{}) bool {
	for _, key := range []string{"item_type", "type", "itemType", "kind"} {
		itemType := normalizeBundleItemType(mapString(item, key))
		if itemType == "pipeline" || itemType == "pipeline_cc" || strings.Contains(itemType, "pipeline") {
			return true
		}
	}
	return false
}

func isMembershipBundleItem(item map[string]interface{}) bool {
	for _, key := range []string{"item_type", "type", "itemType", "kind"} {
		itemType := normalizeBundleItemType(mapString(item, key))
		if itemType == "membership" || strings.Contains(itemType, "membership") {
			return true
		}
	}
	return false
}

func extractMembershipID(bundle *mallpb.BundleInfo) string {
	if bundle == nil {
		return ""
	}
	itemsJSON := bundle.GetItemsJson()
	if itemsJSON != "" {
		var list []map[string]interface{}
		if err := json.Unmarshal([]byte(itemsJSON), &list); err == nil {
			for _, item := range list {
				for _, key := range []string{"membership_id", "membership_ulid"} {
					if idVal := mapString(item, key); looksLikeULID(idVal) {
						return idVal
					}
				}
				if !isMembershipBundleItem(item) {
					continue
				}
				for _, key := range []string{"ref_ulid", "item_id", "id"} {
					if idVal := mapString(item, key); looksLikeULID(idVal) {
						return idVal
					}
				}
			}
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(itemsJSON), &obj); err == nil {
			if memberships, ok := obj["memberships"].([]interface{}); ok {
				for _, membership := range memberships {
					if idVal, ok := membership.(string); ok && looksLikeULID(idVal) {
						return idVal
					}
					if item, ok := membership.(map[string]interface{}); ok {
						for _, key := range []string{"ref_ulid", "item_id", "id", "membership_id", "membership_ulid"} {
							if idVal := mapString(item, key); looksLikeULID(idVal) {
								return idVal
							}
						}
					}
				}
			}
			for _, key := range []string{"membership_id", "membership_ulid", "membership"} {
				if idVal := mapString(obj, key); looksLikeULID(idVal) {
					return idVal
				}
			}
		}
	}
	return ""
}

func appendUniqueString(list []string, value string) []string {
	value = normalizeBundleItemType(value)
	if value == "" {
		return list
	}
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

func bundleItemTypes(bundle *mallpb.BundleInfo) []string {
	if bundle == nil {
		return []string{}
	}
	out := []string{}
	itemsJSON := bundle.GetItemsJson()
	if itemsJSON != "" {
		var list []map[string]interface{}
		if err := json.Unmarshal([]byte(itemsJSON), &list); err == nil {
			for _, item := range list {
				for _, key := range []string{"item_type", "type", "itemType", "kind"} {
					out = appendUniqueString(out, mapString(item, key))
				}
			}
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(itemsJSON), &obj); err == nil {
			for _, key := range []string{"item_type", "type", "itemType", "kind"} {
				out = appendUniqueString(out, mapString(obj, key))
			}
			if _, ok := obj["pipelines"]; ok {
				out = appendUniqueString(out, "pipeline")
			}
			if _, ok := obj["memberships"]; ok {
				out = appendUniqueString(out, "membership")
			}
		}
	}
	gpath := strings.ToLower(strings.TrimSpace(bundle.GetBundleGpath()))
	if strings.Contains(gpath, "/pipeline/") {
		out = appendUniqueString(out, "pipeline")
	}
	if strings.Contains(gpath, "/mbr") || strings.Contains(gpath, "membership") {
		out = appendUniqueString(out, "membership")
	}
	return out
}

func defaultBundleEligibility() bundleEligibilitySummary {
	return bundleEligibilitySummary{
		Eligible:    true,
		CanPurchase: true,
		CanUnlock:   false,
		Blockers:    []bundleEligibilityBlocker{},
	}
}

func unavailableBundleEligibility(description string) bundleEligibilitySummary {
	return bundleEligibilitySummary{
		Eligible:    false,
		CanPurchase: false,
		CanUnlock:   false,
		Blockers: []bundleEligibilityBlocker{
			{
				BlockerType: "ELIGIBILITY_UNAVAILABLE",
				Description: description,
			},
		},
	}
}

func eligibilityFromPipeline(resp *mallpb.CheckPipelineEligibilityResponse) bundleEligibilitySummary {
	if resp == nil {
		return unavailableBundleEligibility("eligibility unavailable")
	}
	out := bundleEligibilitySummary{
		Eligible:    resp.GetEligible(),
		CanUnlock:   resp.GetCanUnlock(),
		CanPurchase: resp.GetCanPurchase(),
		Blockers:    make([]bundleEligibilityBlocker, 0, len(resp.GetBlockers())),
	}
	for _, blocker := range resp.GetBlockers() {
		if blocker == nil {
			continue
		}
		out.Blockers = append(out.Blockers, bundleEligibilityBlocker{
			BlockerType: blocker.GetBlockerType(),
			Description: blocker.GetDescription(),
			Details:     append([]string{}, blocker.GetDetails()...),
		})
	}
	return out
}

func isActiveMembershipStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == "active" || status == "membership_status_active"
}

func (h *Handler) newBundleEnrichmentState(ctx context.Context, candidateID string) *bundleEnrichmentState {
	state := &bundleEnrichmentState{
		candidateID:              strings.TrimSpace(candidateID),
		activeMembershipByGpath:  map[string]*gmbrpb.UserMembership{},
		loadedMembershipsByGpath: map[string]bool{},
	}
	if state.candidateID == "" || h.Gmbr == nil {
		return state
	}
	resp, err := h.Gmbr.ListUserMemberships(ctx, &gmbrpb.ListUserMembershipsRequest{
		CandidateUlid: state.candidateID,
		Page:          1,
		PageSize:      100,
	})
	if err != nil {
		slog.Warn("Failed to preload membership history for bundle list", "error", err, "candidate_id", state.candidateID)
		return state
	}
	state.membershipHistory = resp.GetUserMemberships()
	return state
}

func findMatchingMembershipRecord(records []*gmbrpb.UserMembership, membershipID string, membershipGpath string) *gmbrpb.UserMembership {
	membershipID = strings.TrimSpace(membershipID)
	membershipGpath = strings.TrimSpace(membershipGpath)
	for _, record := range records {
		if record == nil || !isActiveMembershipStatus(record.GetStatus()) {
			continue
		}
		if membershipGpath != "" && strings.TrimSpace(record.GetMembershipGpath()) == membershipGpath {
			return record
		}
		if membershipID != "" && strings.TrimSpace(record.GetMembershipUlid()) == membershipID {
			return record
		}
	}
	return nil
}

func (h *Handler) resolveActiveMembership(ctx context.Context, state *bundleEnrichmentState, membershipID string, membershipGpath string) *gmbrpb.UserMembership {
	if state == nil || state.candidateID == "" || h.Gmbr == nil {
		return nil
	}
	record := findMatchingMembershipRecord(state.membershipHistory, membershipID, membershipGpath)
	if record == nil {
		return nil
	}
	if membershipGpath == "" {
		return record
	}
	if state.loadedMembershipsByGpath[membershipGpath] {
		return state.activeMembershipByGpath[membershipGpath]
	}
	state.loadedMembershipsByGpath[membershipGpath] = true
	resp, err := h.Gmbr.GetActiveMembership(ctx, &gmbrpb.GetActiveMembershipRequest{
		CandidateUlid:   state.candidateID,
		MembershipGpath: membershipGpath,
	})
	if err != nil {
		slog.Warn("Failed to get active membership for bundle list", "error", err, "candidate_id", state.candidateID, "membership_gpath", membershipGpath)
		state.activeMembershipByGpath[membershipGpath] = record
		return record
	}
	state.activeMembershipByGpath[membershipGpath] = resp.GetMembership()
	if resp.GetMembership() != nil {
		return resp.GetMembership()
	}
	return record
}

func (h *Handler) bundleThumbnailURL(ctx context.Context, bundleID string) string {
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return ""
	}
	resp, err := h.Mall.GetBundleThumbnailURL(ctx, &mallpb.GetBundleThumbnailURLRequest{
		BundleUlid: bundleID,
	})
	if err != nil {
		slog.Warn("Failed to get bundle thumbnail url during bundle list enrichment", "error", err, "bundle_id", bundleID)
		return ""
	}
	return strings.TrimSpace(resp.GetPublicUrl())
}

func isOpenMallOrderStatus(status string) bool {
	status = strings.ToUpper(strings.TrimSpace(status))
	if status == "" {
		return true
	}
	return !strings.Contains(status, "COMPLETED") &&
		!strings.Contains(status, "CANCEL") &&
		!strings.Contains(status, "FAILED")
}

func bundleOrderID(order interface{}) string {
	switch v := order.(type) {
	case *mallpb.BundleOrderSummary:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(v.GetBundleOrderUlid())
	case *mallpb.BundleOrderDetail:
		if v == nil {
			return ""
		}
		return bundleOrderID(v.GetSummary())
	default:
		return ""
	}
}

func bundleOrderStatus(order interface{}) string {
	switch v := order.(type) {
	case *mallpb.BundleOrderSummary:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(v.GetOrderStatus())
	case *mallpb.BundleOrderDetail:
		if v == nil {
			return ""
		}
		return bundleOrderStatus(v.GetSummary())
	default:
		return ""
	}
}

func bundleOrderPayID(order interface{}) string {
	switch v := order.(type) {
	case *mallpb.BundleOrderSummary:
		if v == nil {
			return ""
		}
		return strings.TrimSpace(v.GetBundlePayOrderUlid())
	case *mallpb.BundleOrderDetail:
		if v == nil {
			return ""
		}
		return bundleOrderPayID(v.GetSummary())
	default:
		return ""
	}
}

func toBundlePaymentPreviewSummary(resp *mallpb.PreviewPaymentResponse) *bundlePaymentPreviewSummary {
	if resp == nil {
		return nil
	}
	return &bundlePaymentPreviewSummary{
		Subtotal:      resp.GetSubtotal(),
		DiscountTotal: resp.GetDiscountTotal(),
		TaxTotal:      resp.GetTaxTotal(),
		Total:         resp.GetTotal(),
		Currency:      resp.GetCurrency(),
	}
}

func (h *Handler) previewPaymentSummary(ctx context.Context, bizType string, bizRefULID string) *bundlePaymentPreviewSummary {
	bizType = strings.TrimSpace(bizType)
	bizRefULID = strings.TrimSpace(bizRefULID)
	if bizType == "" || bizRefULID == "" {
		return nil
	}
	resp, err := h.Mall.PreviewPayment(ctx, &mallpb.PreviewPaymentRequest{
		BizType:    bizType,
		BizRefUlid: bizRefULID,
	})
	if err != nil {
		slog.Warn("Failed to preview bundle payment during enrichment", "error", err, "biz_type", bizType, "biz_ref_ulid", bizRefULID)
		return nil
	}
	return toBundlePaymentPreviewSummary(resp)
}

func (h *Handler) activeBundleOrder(ctx context.Context, candidateID string, bundleID string) (*bundleActiveOrderSummary, *bundlePaymentPreviewSummary) {
	candidateID = strings.TrimSpace(candidateID)
	bundleID = strings.TrimSpace(bundleID)
	if candidateID == "" || bundleID == "" {
		return nil, nil
	}
	resp, err := h.Mall.ListBundleOrders(ctx, &mallpb.ListBundleOrdersRequest{
		CandidateUlid: candidateID,
		BundleUlid:    bundleID,
		Limit:         20,
	})
	if err != nil {
		slog.Warn("Failed to load active bundle order during enrichment", "error", err, "candidate_id", candidateID, "bundle_id", bundleID)
		return nil, nil
	}
	for _, item := range resp.GetItems() {
		if item == nil || !isOpenMallOrderStatus(item.GetOrderStatus()) {
			continue
		}
		var order interface{} = item
		detailResp, err := h.Mall.GetBundleOrderDetail(ctx, &mallpb.GetBundleOrderDetailRequest{
			BundleOrderUlid: item.GetBundleOrderUlid(),
		})
		if err != nil {
			slog.Warn("Failed to load active bundle order detail during enrichment", "error", err, "bundle_order_ulid", item.GetBundleOrderUlid())
		} else if detailResp.GetFound() && detailResp.GetDetail() != nil {
			order = detailResp.GetDetail()
		}

		orderID := bundleOrderID(order)
		if orderID == "" {
			continue
		}
		active := &bundleActiveOrderSummary{
			Action:     "purchase",
			OrderID:    orderID,
			Status:     bundleOrderStatus(order),
			PayOrderID: bundleOrderPayID(order),
			Message:    "in-progress purchase order exists",
		}
		return active, h.previewPaymentSummary(ctx, orderBizBundlePurchase, orderID)
	}
	return nil, nil
}

func (h *Handler) pipelineExemptionOptions(ctx context.Context, candidateID string, pipelineID string) *PipelineExemptionOptionsRsp {
	candidateID = strings.TrimSpace(candidateID)
	pipelineID = strings.TrimSpace(pipelineID)
	if candidateID == "" || pipelineID == "" {
		return nil
	}

	pipeline, err := h.Gcc.GetPipeline(ctx, &gccpb.GetPipelineRequest{
		Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipelineID},
	})
	if err != nil {
		slog.Warn("Failed to load pipeline for exemption options during enrichment", "error", err, "pipeline_id", pipelineID)
		return nil
	}

	out := PipelineExemptionOptionsRsp{
		Stages: make([]PipelineExemptionStage, 0, len(pipeline.GetStages())),
	}
	defCache := map[string]*gcredspb.CredentialDefinition{}
	checkCache := map[string]*gcredspb.CheckCandidateQualificationResponse{}

	for stageIndex, stage := range pipeline.GetStages() {
		stageOut := PipelineExemptionStage{
			Index:     int32(stageIndex),
			StageUlid: stage.GetStageUlid(),
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
				UnitUlid:       unit.GetUnitUlid(),
				UnitName:       unit.GetName(),
				AllowExemption: true,
				ExemptionQuals: make([]PipelineExemptionQual, 0, len(qualIDs)),
			}
			for _, qualID := range qualIDs {
				def := h.cachedCredentialDefinition(ctx, defCache, qualID)
				check := h.cachedCandidateQualification(ctx, checkCache, candidateID, qualID)
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

	return &out
}

func (h *Handler) cachedCredentialDefinition(ctx context.Context, cache map[string]*gcredspb.CredentialDefinition, qualID string) *gcredspb.CredentialDefinition {
	if def, ok := cache[qualID]; ok {
		return def
	}
	def, err := h.Creds.GetCredentialDefinitionDetail(ctx, &gcredspb.GetCredentialDefinitionDetailRequest{
		CredDefUlid: qualID,
	})
	if err != nil {
		slog.Warn("Failed to load credential definition for exemption option", "error", err, "qual_id", qualID)
		cache[qualID] = nil
		return nil
	}
	cache[qualID] = def
	return def
}

func (h *Handler) cachedCandidateQualification(ctx context.Context, cache map[string]*gcredspb.CheckCandidateQualificationResponse, candidateID string, qualID string) *gcredspb.CheckCandidateQualificationResponse {
	if check, ok := cache[qualID]; ok {
		return check
	}
	check, err := h.Creds.CheckCandidateQualification(ctx, &gcredspb.CheckCandidateQualificationRequest{
		CandidateUlid: candidateID,
		CredDefUlid:   qualID,
	})
	if err != nil {
		slog.Warn("Failed to check candidate qualification for exemption option", "error", err, "candidate_id", candidateID, "qual_id", qualID)
		cache[qualID] = nil
		return nil
	}
	cache[qualID] = check
	return check
}

func (h *Handler) bundlePurchaseState(ctx context.Context, state *bundleEnrichmentState, bundleID string, pipelineID string, eligibility bundleEligibilitySummary) bundlePurchaseState {
	out := bundlePurchaseState{
		Eligibility: eligibility,
	}
	if state == nil || strings.TrimSpace(state.candidateID) == "" {
		return out
	}
	if activeOrder, preview := h.activeBundleOrder(ctx, state.candidateID, bundleID); activeOrder != nil {
		out.ActiveOrder = activeOrder
		out.PaymentPreview = preview
	}
	if pipelineID != "" && (eligibility.CanPurchase || out.ActiveOrder != nil) {
		out.ExemptionOptions = h.pipelineExemptionOptions(ctx, state.candidateID, pipelineID)
	}
	return out
}

func (h *Handler) enrichBundle(ctx context.Context, b *mallpb.BundleInfo, state *bundleEnrichmentState) map[string]interface{} {
	pipelineID := h.extractPipelineID(b)
	membershipID := extractMembershipID(b)
	itemTypes := bundleItemTypes(b)
	eligibility := defaultBundleEligibility()
	membershipGpath := ""
	m := map[string]interface{}{
		"bundle_id":            b.GetBundleUlid(),
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
		"display_amount_min":   b.GetDisplayAmountMin(),
		"display_amount_max":   b.GetDisplayAmountMax(),
		"display_currency":     b.GetDisplayCurrency(),
		"stages":               []interface{}{},
		"final_quals":          []interface{}{},
		"category_tips":        "",
		"pipeline_id":          pipelineID,
		"membership_id":        membershipID,
		"membership_gpath":     "",
		"bundle_item_types":    itemTypes,
		"is_pipeline_bundle":   pipelineID != "",
		"is_membership_bundle": membershipID != "",
		"thumbnail_url":        h.bundleThumbnailURL(ctx, b.GetBundleUlid()),
	}

	if pipelineID != "" {
		pipeline, err := h.Gcc.GetPipeline(ctx, &gccpb.GetPipelineRequest{
			Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: pipelineID},
		})
		if err == nil && pipeline != nil {
			m["stages"] = toStages(pipeline.GetStages())
			m["final_quals"] = toUnlockQuals(pipeline.GetCertsQuals())
			m["category_tips"] = pipeline.GetCategoryTips()
		}
		if state != nil && state.candidateID != "" {
			eligibilityResp, err := h.Mall.CheckPipelineEligibility(ctx, &mallpb.CheckPipelineEligibilityRequest{
				CandidateUlid:  state.candidateID,
				PipelineCcUlid: pipelineID,
			})
			if err != nil {
				slog.Warn("Failed to enrich pipeline bundle eligibility", "error", err, "candidate_id", state.candidateID, "pipeline_id", pipelineID, "bundle_id", b.GetBundleUlid())
				eligibility = unavailableBundleEligibility("eligibility unavailable")
			} else {
				eligibility = eligibilityFromPipeline(eligibilityResp)
			}
		}
	}
	if membershipID != "" && h.Gmbr != nil {
		membership, err := h.Gmbr.GetMembership(ctx, &gmbrpb.GetMembershipRequest{
			MembershipUlid: membershipID,
		})
		if err == nil && membership != nil {
			membershipGpath = membership.GetMembershipGpath()
			m["membership_gpath"] = membershipGpath
			m["membership_name"] = membership.GetName()
			m["membership_status"] = membership.GetStatus()
			m["membership_tier_level"] = membership.GetTierLevel()
			if m["category_tips"] == "" {
				m["category_tips"] = membership.GetName()
			}
		} else if err != nil {
			slog.Warn("Failed to enrich membership bundle", "error", err, "membership_id", membershipID, "bundle_id", b.GetBundleUlid())
		}
		if activeMembership := h.resolveActiveMembership(ctx, state, membershipID, membershipGpath); activeMembership != nil {
			m["active_membership"] = activeMembership
			eligibility = bundleEligibilitySummary{
				Eligible:    false,
				CanUnlock:   false,
				CanPurchase: false,
				Blockers: []bundleEligibilityBlocker{
					{
						BlockerType: "ALREADY_PURCHASED",
						Description: "active membership already exists",
					},
				},
			}
		}
	}
	purchaseState := h.bundlePurchaseState(ctx, state, b.GetBundleUlid(), pipelineID, eligibility)
	m["eligibility"] = purchaseState.Eligibility
	m["purchase_state"] = purchaseState
	if purchaseState.ActiveOrder != nil {
		m["active_order"] = purchaseState.ActiveOrder
	}
	if purchaseState.PaymentPreview != nil {
		m["payment_preview"] = purchaseState.PaymentPreview
	}
	if purchaseState.ExemptionOptions != nil {
		m["exemption_options"] = purchaseState.ExemptionOptions
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
		BundleUlid: bundleId,
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
	bundleOrderID := strings.TrimSpace(req.BundleOrderUlid)
	if bundleOrderID == "" {
		bundleOrderID = ulid.Make().String()
	}

	resp, err := h.Mall.CreateBundleOrder(r.Context(), &mallpb.CreateBundleOrderRequest{
		CandidateUlid:          candidateID,
		BundleCcUlid:           bundleId,
		PaymentMode:            req.PaymentMode,
		SelectedExemptionsJson: req.SelectedExemptionsJson,
		BundleOrderUlid:        bundleOrderID,
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
	finalQuals := certQuals
	if finalQuals == nil {
		finalQuals = p.GetCertsQuals()
	}

	return PipelineConfig{
		PipelineUlid:          p.GetPipelineUlid(),
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
		FinalQuals:            toUnlockQuals(finalQuals),
	}
}

func pipelineSummaryToConfig(p *gccpb.PipelineSummary) *gccpb.PipelineConfig {
	if p == nil {
		return nil
	}
	return &gccpb.PipelineConfig{
		PipelineUlid:       p.GetPipelineUlid(),
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
			QualId:   qual.GetQualUlid(),
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
			StageUlid: stage.GetStageUlid(),
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
			UnitUlid:                 unit.GetUnitUlid(),
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
			GlmsCourseUlid:           unit.GetGlmsCourseUlid(),
			Program:                  unit.GetProgram(),
			ExamUlid:                 unit.GetExamUlid(),
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
	issuesCertificate := pipelineIssuesCertificate(config)
	if out.PipelineStatus == gprogpb.PipelineStatus_PIPELINE_STATUS_WAIT_FINAL_ELIG {
		fillFinalEligibilityNextStep(&out, issuesCertificate)
		return out
	}
	if out.PipelineStatus == gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED ||
		out.PipelineStatus == gprogpb.PipelineStatus_PIPELINE_STATUS_ISSUING_CERT {
		fillCompletedPipelineNextStep(&out, issuesCertificate)
		return out
	}
	if config == nil {
		return out
	}

	if runtime == nil || runtime.GetPipeline() == nil {
		firstUnit := firstConfigUnit(config)
		if firstUnit == nil {
			fillCompletedPipelineNextStep(&out, issuesCertificate)
			return out
		}
		fillNextStepFromUnit(&out, nil, firstUnit, "")
		if strings.TrimSpace(firstUnit.GetGlmsCourseUlid()) != "" {
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
					out.StageUlid = stage.GetStage().GetStageUlid()
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
			fillCompletedPipelineNextStep(&out, issuesCertificate)
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
		out.PipelineStatus = runtime.GetPipeline().GetStatus()
		if out.PipelineStatus == gprogpb.PipelineStatus_PIPELINE_STATUS_WAIT_FINAL_ELIG {
			fillFinalEligibilityNextStep(&out, issuesCertificate)
			return out
		}
	}

	pickStageUlidx := -1
	for idx, stage := range stageDetails {
		if stage == nil || stage.GetStage() == nil {
			continue
		}
		if currentStageUlid != "" && stage.GetStage().GetStageUlid() == currentStageUlid {
			pickStageUlidx = idx
			break
		}
	}
	if pickStageUlidx < 0 {
		pickStageUlidx = 0
	}

	pickStage := stageDetails[pickStageUlidx]
	pickUnit := pickNextRuntimeUnit(pickStage)
	if pickUnit == nil {
		for idx := pickStageUlidx + 1; idx < len(stageDetails); idx++ {
			pickUnit = pickNextRuntimeUnit(stageDetails[idx])
			if pickUnit != nil {
				pickStage = stageDetails[idx]
				break
			}
		}
	}

	if pickUnit == nil {
		if runtime.GetPipeline().GetStatus() == gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED {
			fillCompletedPipelineNextStep(&out, issuesCertificate)
		} else if runtime.GetPipeline().GetStatus() == gprogpb.PipelineStatus_PIPELINE_STATUS_WAIT_FINAL_ELIG {
			fillFinalEligibilityNextStep(&out, issuesCertificate)
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
			fillCompletedPipelineNextStep(&out, issuesCertificate)
		} else if runtime.GetPipeline().GetStatus() == gprogpb.PipelineStatus_PIPELINE_STATUS_WAIT_FINAL_ELIG {
			fillFinalEligibilityNextStep(&out, issuesCertificate)
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

func pipelineIssuesCertificate(config *gccpb.PipelineConfig) bool {
	if config == nil {
		return false
	}
	for _, qual := range config.GetCertsQuals() {
		if strings.TrimSpace(qual.GetQualUlid()) != "" {
			return true
		}
	}
	return false
}

func fillCompletedPipelineNextStep(out *PipelineNextStep, issuesCertificate bool) {
	if out == nil {
		return
	}
	if issuesCertificate {
		out.Action = "view_certificate"
		out.Message = "view the certificate"
		return
	}
	out.Action = "completed"
	out.Message = "pipeline completed"
}

func fillFinalEligibilityNextStep(out *PipelineNextStep, issuesCertificate bool) {
	out.PipelineStatus = gprogpb.PipelineStatus_PIPELINE_STATUS_WAIT_FINAL_ELIG
	if issuesCertificate {
		out.Action = "final_qualification"
		out.Message = "submit final qualification materials"
		return
	}
	out.Action = "completed"
	out.Message = "pipeline completed"
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
	out.CourseUnitUlid = unit.GetUnitUlid()
	out.CourseUlid = unit.GetGlmsCourseUlid()
	out.AllowRetake = unit.GetExamUlid() != ""
	out.AllowExemption = unit.GetAllowExemption()
	out.Program = unit.GetProgram()
	out.ExamUlid = unit.GetExamUlid()
	out.FormCode = unit.GetFormCode()
	if stage != nil && stage.GetStage() != nil {
		out.StageUlid = stage.GetStage().GetStageUlid()
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
		if strings.TrimSpace(stage.GetStageUlid()) == stageID {
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
			if unit != nil && strings.TrimSpace(unit.GetUnitUlid()) != "" {
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
			if unit.GetUnitUlid() == unitID {
				return unit
			}
		}
	}
	return nil
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
		BundleUlid:     bundleId,
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
