package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"

	"github.com/go-chi/chi/v5"
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
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to list pipelines")
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

	resp, err := h.Gcc.CreatePipelineDraft(r.Context(), &req)
	if err != nil {
		slog.Error("CreatePipelineDraft failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create pipeline draft")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// UpdatePipelineStructure PUT /api/pipelines/{pipeline_id}/structure
func (h *Handler) UpdatePipelineStructure(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "pipeline_id")
	var req gccpb.UpdatePipelineStructureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PipelineId = id

	resp, err := h.Gcc.UpdatePipelineStructure(r.Context(), &req)
	if err != nil {
		slog.Error("UpdatePipelineStructure failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to update structure")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// PublishPipeline POST /api/pipelines/{pipeline_id}/publish
func (h *Handler) PublishPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "pipeline_id")
	var req gccpb.PublishPipelineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.PipelineId = id

	resp, err := h.Gcc.PublishPipeline(r.Context(), &req)
	if err != nil {
		slog.Error("PublishPipeline failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to publish pipeline")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// GetPipeline GET /api/pipelines/{pipeline_id}
func (h *Handler) GetPipeline(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "pipeline_id")
	var req gccpb.GetPipelineRequest
	req.Query = &gccpb.GetPipelineRequest_PipelineId{PipelineId: id}

	resp, err := h.Gcc.GetPipeline(r.Context(), &req)
	if err != nil {
		slog.Error("GetPipeline failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to get pipeline")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ListCatalogs GET /api/catalogs
func (h *Handler) ListCatalogs(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gcc.ListCatalogs(r.Context(), &emptypb.Empty{})
	if err != nil {
		slog.Error("ListCatalogs failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to list catalogs")
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

	resp, err := h.Gcc.CreateCatalog(r.Context(), &req)
	if err != nil {
		slog.Error("CreateCatalog failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to create catalog")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// UpdateCatalog PUT /api/catalogs/{catalog_id}
func (h *Handler) UpdateCatalog(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "catalog_id")
	var req gccpb.UpdateCatalogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	req.CatalogId = id

	resp, err := h.Gcc.UpdateCatalog(r.Context(), &req)
	if err != nil {
		slog.Error("UpdateCatalog failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to update catalog")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
