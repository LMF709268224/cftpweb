package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
)

// ListPipelines GET /api/pipelines
func (h *Handler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	req := &gccpb.ListPipelinesAdminRequest{
		CategoryTips: r.URL.Query().Get("category_tips"),
		OnlyCurrent:  r.URL.Query().Get("only_current") == "true",
		Limit:        int32(parseUint32Query(r, "limit")),
		Offset:       int32(parseUint32Query(r, "offset")),
	}

	resp, err := h.Gcc.ListPipelinesAdmin(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// CreatePipelineDraft POST /api/pipelines
func (h *Handler) CreatePipelineDraft(w http.ResponseWriter, r *http.Request) {
	var input struct {
		CategoryTips       string `json:"category_tips"`
		Name               string `json:"name"`
		PipelineId         string `json:"pipeline_id"`
		PipelineGuid       string `json:"pipeline_guid"`
		Respath            string `json:"respath"`
		ThumbnailObjectKey string `json:"thumbnail_object_key"`
		ThumbnailFileHash  string `json:"thumbnail_file_hash"`
		FromPipelineId     string `json:"from_pipeline_id"`
		FromPipelineGuid   string `json:"from_pipeline_guid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	fromPipelineID := strings.TrimSpace(input.FromPipelineId)
	if fromPipelineID == "" {
		fromPipelineID = strings.TrimSpace(r.URL.Query().Get("from_pipeline_id"))
	}
	fromPipelineGUID := strings.TrimSpace(input.FromPipelineGuid)
	if fromPipelineGUID == "" {
		fromPipelineGUID = strings.TrimSpace(r.URL.Query().Get("from_pipeline_guid"))
	}

	if fromPipelineID == "" && fromPipelineGUID != "" {
		resp, err := h.Gcc.ListPipelinesAdmin(r.Context(), &gccpb.ListPipelinesAdminRequest{
			Limit:  500,
			Offset: 0,
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		candidates := make([]*gccpb.PipelineSummary, 0)
		for _, pipeline := range resp.GetPipelines() {
			if pipeline.GetPipelineGuid() == fromPipelineGUID {
				candidates = append(candidates, pipeline)
			}
		}
		sort.SliceStable(candidates, func(i, j int) bool {
			return candidates[i].GetVersion() > candidates[j].GetVersion()
		})
		if len(candidates) > 0 {
			fromPipelineID = candidates[0].GetPipelineId()
		}
		if fromPipelineID == "" {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "from_pipeline_guid was not found")
			return
		}
	}

	if fromPipelineID != "" {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			name = "Pipeline Copy"
		}
		req := &gccpb.DuplicatePipelineDraftRequest{
			FromPipelineId: fromPipelineID,
			PipelineId:     newLmsID(),
			Name:           name,
		}
		if !requireRequestFields(w, req.FromPipelineId, "from_pipeline_id", req.PipelineId, "pipeline_id", req.Name, "name") {
			return
		}
		resp, err := h.Gcc.DuplicatePipelineDraft(r.Context(), req)
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		WriteJSON(w, http.StatusOK, resp)
		return
	}

	req := gccpb.CreatePipelineDraftRequest{
		CategoryTips:       strings.TrimSpace(input.CategoryTips),
		Name:               strings.TrimSpace(input.Name),
		PipelineId:         strings.TrimSpace(input.PipelineId),
		PipelineGuid:       strings.TrimSpace(input.PipelineGuid),
		Respath:            strings.TrimSpace(input.Respath),
		ThumbnailObjectKey: strings.TrimSpace(input.ThumbnailObjectKey),
		ThumbnailFileHash:  strings.TrimSpace(input.ThumbnailFileHash),
	}
	if req.PipelineId == "" {
		req.PipelineId = newLmsID()
	}
	if strings.TrimSpace(req.PipelineGuid) == "" {
		req.PipelineGuid = newLmsID()
	}
	if !requireRequestFields(w, req.CategoryTips, "category_tips", req.Name, "name", req.Respath, "respath") {
		return
	}

	resp, err := h.Gcc.CreatePipelineDraft(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// UpdatePipelineStructure PUT /api/pipelines/{pipeline_id}/structure
func (h *Handler) UpdatePipelineStructure(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "pipeline_id")
	if !ok {
		return
	}
	var req gccpb.UpdatePipelineStructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PipelineId = id
	if len(req.Stages) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "stages is required")
		return
	}
	if !requirePairedFields(w, req.UnlockStripeProductId, "unlock_stripe_product_id", req.UnlockStripePriceId, "unlock_stripe_price_id") {
		return
	}
	for i, stage := range req.Stages {
		if stage.StageId == "" {
			stage.StageId = newLmsID()
		}
		if !requireRequestField(w, stage.Name, "stages["+strconv.Itoa(i)+"].name") {
			return
		}
		if len(stage.Units) == 0 {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "stages["+strconv.Itoa(i)+"].units is required")
			return
		}
		for j, unit := range stage.Units {
			if unit.UnitId == "" {
				unit.UnitId = newLmsID()
			}
			if !requireRequestField(w, unit.GlmsCourseId, "stages["+strconv.Itoa(i)+"].units["+strconv.Itoa(j)+"].glms_course_id") {
				return
			}
			prefix := "stages[" + strconv.Itoa(i) + "].units[" + strconv.Itoa(j) + "]"
			if !requirePairedFields(w, unit.StripeProductId, prefix+".stripe_product_id", unit.StripePriceId, prefix+".stripe_price_id") {
				return
			}
			if !requirePairedFields(w, unit.ExemptionStripeProductId, prefix+".exemption_stripe_product_id", unit.ExemptionStripePriceId, prefix+".exemption_stripe_price_id") {
				return
			}
			if !requirePairedFields(w, unit.RetakeStripeProductId, prefix+".retake_stripe_product_id", unit.RetakeStripePriceId, prefix+".retake_stripe_price_id") {
				return
			}
		}
	}

	resp, err := h.Gcc.UpdatePipelineStructure(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// PublishPipeline POST /api/pipelines/{pipeline_id}/publish
func (h *Handler) PublishPipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "pipeline_id")
	if !ok {
		return
	}
	var req gccpb.PublishPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PipelineId = id

	resp, err := h.Gcc.PublishPipeline(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// DeprecatePipeline POST /api/pipelines/{pipeline_id}/deprecate
func (h *Handler) DeprecatePipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "pipeline_id")
	if !ok {
		return
	}

	resp, err := h.Gcc.DeprecatePipeline(r.Context(), &gccpb.DeprecatePipelineRequest{
		PipelineId: id,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// DeletePipeline DELETE /api/pipelines/{pipeline_id}
func (h *Handler) DeletePipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "pipeline_id")
	if !ok {
		return
	}

	resp, err := h.Gcc.DeletePipeline(r.Context(), &gccpb.DeletePipelineRequest{
		PipelineId: id,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// UpdatePipelineMetadata PUT /api/pipelines/{pipeline_id}/metadata
func (h *Handler) UpdatePipelineMetadata(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "pipeline_id")
	if !ok {
		return
	}
	var req gccpb.UpdateMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.TargetId = id
	if req.NewName != nil && !requireRequestField(w, *req.NewName, "new_name") {
		return
	}

	resp, err := h.Gcc.UpdatePipelineMetadata(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// GetPipeline GET /api/pipelines/{pipeline_id}
func (h *Handler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "pipeline_id")
	if !ok {
		return
	}
	req := &gccpb.GetPipelineRequest{
		Query: &gccpb.GetPipelineRequest_PipelineId{PipelineId: id},
	}

	resp, err := h.Gcc.GetPipeline(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ListCatalogs GET /api/catalogs
func (h *Handler) ListCatalogs(w http.ResponseWriter, r *http.Request) {
	// TODO: 待微服务团队补充 GCC catalog 管理接口后接入；当前 GCC proto 已移除 ListCatalogs/CreateCatalog/UpdateCatalog。
	WriteJSON(w, http.StatusOK, map[string]any{"catalogs": []any{}})
}

// CreateCatalog POST /api/catalogs
func (h *Handler) CreateCatalog(w http.ResponseWriter, r *http.Request) {
	// TODO: 待微服务团队补充 GCC catalog 管理接口后接入；当前 GCC proto 已移除 ListCatalogs/CreateCatalog/UpdateCatalog。
	WriteError(w, http.StatusNotImplemented, ErrInvalidRequest, "catalog management API is not available in current GCC proto")
}

// UpdateCatalog PUT /api/catalogs/{catalog_id}
func (h *Handler) UpdateCatalog(w http.ResponseWriter, r *http.Request) {
	// TODO: 待微服务团队补充 GCC catalog 管理接口后接入；当前 GCC proto 已移除 ListCatalogs/CreateCatalog/UpdateCatalog。
	WriteError(w, http.StatusNotImplemented, ErrInvalidRequest, "catalog management API is not available in current GCC proto")
}

func requirePairedFields(w http.ResponseWriter, left string, leftName string, right string, rightName string) bool {
	hasLeft := strings.TrimSpace(left) != ""
	hasRight := strings.TrimSpace(right) != ""
	if hasLeft == hasRight {
		return true
	}
	if hasLeft {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, rightName+" is required when "+leftName+" is provided")
		return false
	}
	WriteError(w, http.StatusBadRequest, ErrInvalidRequest, leftName+" is required when "+rightName+" is provided")
	return false
}
