package handler

import (
	"net/http"
	"strconv"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
)

// ListPdfTemplates GET /api/pdf-templates
func (h *Handler) ListPdfTemplates(w http.ResponseWriter, r *http.Request) {
	req := &gcredspb.ListPdfTemplatesRequest{}

	res, err := h.Creds.ListPdfTemplates(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type CreatePdfTemplateReq struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	HtmlTemplate string `json:"html_template"`
}

// CreatePdfTemplate POST /api/pdf-templates
func (h *Handler) CreatePdfTemplate(w http.ResponseWriter, r *http.Request) {
	var body CreatePdfTemplateReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	req := &gcredspb.CreatePdfTemplateRequest{
		Name:         body.Name,
		Description:  body.Description,
		HtmlTemplate: body.HtmlTemplate,
	}

	res, err := h.Creds.CreatePdfTemplate(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type UpdatePdfTemplateReq struct {
	TemplateId   string `json:"template_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	HtmlTemplate string `json:"html_template"`
}

// UpdatePdfTemplate PUT /api/pdf-templates
func (h *Handler) UpdatePdfTemplate(w http.ResponseWriter, r *http.Request) {
	var body UpdatePdfTemplateReq
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	req := &gcredspb.UpdatePdfTemplateRequest{
		TemplateUlid: body.TemplateId,
		Name:         body.Name,
		Description:  body.Description,
		HtmlTemplate: body.HtmlTemplate,
	}

	res, err := h.Creds.UpdatePdfTemplate(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// ListPdfRequests GET /api/pdf-requests
func (h *Handler) ListPdfRequests(w http.ResponseWriter, r *http.Request) {
	// 榛樿鍙傛暟
	page := uint32(1)
	pageSize := uint32(20)

	// query params
	qPageNumber := r.URL.Query().Get("page")
	qPageSize := r.URL.Query().Get("page_size")

	if qPageNumber != "" {
		if val, err := strconv.Atoi(qPageNumber); err == nil {
			page = uint32(val)
		}
	}
	if qPageSize != "" {
		if val, err := strconv.Atoi(qPageSize); err == nil {
			pageSize = uint32(val)
		}
	}

	req := &gcredspb.ListPdfRequestsRequest{
		Page:     uint32(page),
		PageSize: uint32(pageSize),
	}

	res, err := h.Creds.ListPdfRequests(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}
