package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"

	"google.golang.org/protobuf/types/known/emptypb"
)

// ListPipelines GET /api/pipelines
func (h *Handler) ListPipelines(w http.ResponseWriter, r *http.Request) {
	var req gccpb.ListPipelinesRequest
	// 可以从 query params 中解析条件
	req.CategoryId = r.URL.Query().Get("category_id")
	req.OnlyCurrent = r.URL.Query().Get("only_current") == "true"

	resp, err := h.Gcc.ListPipelines(r.Context(), &req)
	if err != nil {
		slog.Error("ListPipelines failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// CreatePipelineDraft POST /api/pipelines
func (h *Handler) CreatePipelineDraft(w http.ResponseWriter, r *http.Request) {
	var req gccpb.CreatePipelineDraftRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PipelineId = newLmsID()
	if req.FromPipelineGuid == "" && req.PipelineGuid == "" {
		req.PipelineGuid = newLmsID()
	}
	if !requireRequestFields(w, req.CategoryId, "category_id", req.Name, "name") {
		return
	}

	resp, err := h.Gcc.CreatePipelineDraft(r.Context(), &req)
	if err != nil {
		slog.Error("CreatePipelineDraft failed", "error", err)
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
	for i, stage := range req.Stages {
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
		}
	}

	resp, err := h.Gcc.UpdatePipelineStructure(r.Context(), &req)
	if err != nil {
		slog.Error("UpdatePipelineStructure failed", "error", err)
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
		slog.Error("PublishPipeline failed", "error", err)
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
	var req gccpb.GetPipelineRequest
	req.Query = &gccpb.GetPipelineRequest_PipelineId{PipelineId: id}

	resp, err := h.Gcc.GetPipeline(r.Context(), &req)
	if err != nil {
		slog.Error("GetPipeline failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ListCatalogs GET /api/catalogs
func (h *Handler) ListCatalogs(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gcc.ListCatalogs(r.Context(), &emptypb.Empty{})
	if err != nil {
		slog.Error("ListCatalogs failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// CreateCatalog POST /api/catalogs
func (h *Handler) CreateCatalog(w http.ResponseWriter, r *http.Request) {
	var req gccpb.CreateCatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if !requireRequestField(w, req.Name, "name") {
		return
	}

	resp, err := h.Gcc.CreateCatalog(r.Context(), &req)
	if err != nil {
		slog.Error("CreateCatalog failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// UpdateCatalog PUT /api/catalogs/{catalog_id}
func (h *Handler) UpdateCatalog(w http.ResponseWriter, r *http.Request) {
	id, ok := requiredURLParam(w, r, "catalog_id")
	if !ok {
		return
	}
	var req gccpb.UpdateCatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CatalogId = id
	if !requireRequestField(w, req.Name, "name") {
		return
	}

	resp, err := h.Gcc.UpdateCatalog(r.Context(), &req)
	if err != nil {
		slog.Error("UpdateCatalog failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
