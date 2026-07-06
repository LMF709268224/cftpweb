package handler

import (
	"encoding/json"
	"net/http"
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
		PipelineUlid       string `json:"pipeline_id"`
		Respath            string `json:"respath"`
		ThumbnailObjectKey string `json:"thumbnail_object_key"`
		ThumbnailFileHash  string `json:"thumbnail_file_hash"`
		FromPipelineUlid   string `json:"from_pipeline_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	fromPipelineID := strings.TrimSpace(input.FromPipelineUlid)
	if fromPipelineID == "" {
		fromPipelineID = strings.TrimSpace(r.URL.Query().Get("from_pipeline_id"))
	}

	if fromPipelineID != "" {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			name = "Pipeline Copy"
		}
		req := &gccpb.DuplicatePipelineDraftRequest{
			FromPipelineUlid: fromPipelineID,
			PipelineUlid:     newLmsID(),
			Name:             name,
		}
		if !requireRequestFields(w, req.FromPipelineUlid, "from_pipeline_id", req.PipelineUlid, "pipeline_id", req.Name, "name") {
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
		PipelineUlid:       strings.TrimSpace(input.PipelineUlid),
		Respath:            strings.TrimSpace(input.Respath),
		ThumbnailObjectKey: strings.TrimSpace(input.ThumbnailObjectKey),
		ThumbnailFileHash:  strings.TrimSpace(input.ThumbnailFileHash),
	}
	if req.PipelineUlid == "" {
		req.PipelineUlid = newLmsID()
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
	var raw map[string]any
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	normalizePipelineStructureAliases(raw)
	var req gccpb.UpdatePipelineStructureRequest
	normalizedBody, err := json.Marshal(raw)
	if err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if err := json.Unmarshal(normalizedBody, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PipelineUlid = id
	if len(req.Stages) == 0 {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "stages is required")
		return
	}
	for i, stage := range req.Stages {
		if stage.StageUlid == "" {
			stage.StageUlid = newLmsID()
		}
		if !requireRequestField(w, stage.Name, "stages["+strconv.Itoa(i)+"].name") {
			return
		}
		if len(stage.Units) == 0 {
			WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "stages["+strconv.Itoa(i)+"].units is required")
			return
		}
		for j, unit := range stage.Units {
			if unit.UnitUlid == "" {
				unit.UnitUlid = newLmsID()
			}
			if !requireRequestField(w, unit.GlmsCourseUlid, "stages["+strconv.Itoa(i)+"].units["+strconv.Itoa(j)+"].glms_course_ulid") {
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

func normalizePipelineStructureAliases(raw map[string]any) {
	for _, key := range []string{"unlock_quals", "certs_quals", "certs"} {
		normalizeQualificationAliases(raw[key])
	}

	stages, _ := raw["stages"].([]any)
	for _, stageValue := range stages {
		stage, ok := stageValue.(map[string]any)
		if !ok {
			continue
		}
		copyAlias(stage, "stage_ulid", "stage_id")
		units, _ := stage["units"].([]any)
		for _, unitValue := range units {
			unit, ok := unitValue.(map[string]any)
			if !ok {
				continue
			}
			copyAlias(unit, "unit_ulid", "unit_id")
			copyAlias(unit, "glms_course_ulid", "glms_course_id")
			copyAlias(unit, "exam_ulid", "exam_id")
			copyAlias(unit, "cert_qual_ulid", "cert_qual_id")
			copyAlias(unit, "cert_pdf_template_ulid", "cert_pdf_template_id")
		}
	}
}

func normalizeQualificationAliases(value any) {
	items, _ := value.([]any)
	for _, itemValue := range items {
		item, ok := itemValue.(map[string]any)
		if !ok {
			continue
		}
		copyAlias(item, "qual_ulid", "qual_id")
		copyAlias(item, "pdf_template_ulid", "pdf_template_id")
	}
}

func copyAlias(record map[string]any, canonical string, legacy string) {
	if hasJSONValue(record[canonical]) || !hasJSONValue(record[legacy]) {
		return
	}
	record[canonical] = record[legacy]
}

func hasJSONValue(value any) bool {
	if value == nil {
		return false
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) != ""
	}
	return true
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
	req.PipelineUlid = id

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
		PipelineUlid: id,
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
		PipelineUlid: id,
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
	req.TargetUlid = id
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
		Query: &gccpb.GetPipelineRequest_PipelineUlid{PipelineUlid: id},
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
