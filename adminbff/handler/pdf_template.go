package handler

import (
	"net/http"
	"strings"

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

// GetPdfTemplateDetail GET /api/pdf-templates/detail?template_id=...
func (h *Handler) GetPdfTemplateDetail(w http.ResponseWriter, r *http.Request) {
	templateID := firstNonEmpty(r.URL.Query().Get("template_id"), r.URL.Query().Get("template_ulid"))
	if !requireRequestField(w, templateID, "template_id") {
		return
	}

	req := &gcredspb.GetPdfTemplateRequest{
		TemplateUlid: templateID,
	}
	summary, summaryErr := h.Creds.GetPdfTemplate(r.Context(), req)
	detail, err := h.Creds.GetPdfTemplateDetail(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	payload := jsonPayloadObject(detail)
	payload["detail"] = detail
	if summaryErr == nil && summary != nil {
		payload["summary"] = summary
		for key, value := range jsonPayloadObject(summary) {
			if _, exists := payload[key]; !exists {
				payload[key] = value
			}
		}
	}

	WriteJSON(w, http.StatusOK, payload)
}

type CreatePdfTemplateReq struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	HtmlTemplate    string `json:"html_template"`
	ParameterSchema string `json:"parameter_schema"`
}

// CreatePdfTemplate POST /api/pdf-templates
func (h *Handler) CreatePdfTemplate(w http.ResponseWriter, r *http.Request) {
	var body CreatePdfTemplateReq
	if err := ReadLargeJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	req := &gcredspb.CreatePdfTemplateRequest{
		TemplateUlid:    newLmsID(),
		Name:            body.Name,
		Description:     body.Description,
		HtmlTemplate:    body.HtmlTemplate,
		ParameterSchema: body.ParameterSchema,
	}

	res, err := h.Creds.CreatePdfTemplate(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

type UpdatePdfTemplateReq struct {
	TemplateId      string `json:"template_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	HtmlTemplate    string `json:"html_template"`
	ParameterSchema string `json:"parameter_schema"`
}

// UpdatePdfTemplate PUT /api/pdf-templates
func (h *Handler) UpdatePdfTemplate(w http.ResponseWriter, r *http.Request) {
	var body UpdatePdfTemplateReq
	if err := ReadLargeJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "Invalid request body")
		return
	}

	req := &gcredspb.UpdatePdfTemplateRequest{
		TemplateUlid:    body.TemplateId,
		Name:            body.Name,
		Description:     body.Description,
		HtmlTemplate:    body.HtmlTemplate,
		ParameterSchema: body.ParameterSchema,
	}

	res, err := h.Creds.UpdatePdfTemplate(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}

// GetPdfRequest GET /api/pdf-requests/{request_ulid}
func (h *Handler) GetPdfRequest(w http.ResponseWriter, r *http.Request) {
	requestULID, ok := requiredURLParam(w, r, "request_ulid")
	if !ok {
		return
	}
	res, err := h.Creds.GetPdfRequest(r.Context(), &gcredspb.GetPdfRequestRequest{RequestUlid: requestULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

// GetPdfRequestDetail GET /api/pdf-requests/{request_ulid}/detail
func (h *Handler) GetPdfRequestDetail(w http.ResponseWriter, r *http.Request) {
	requestULID, ok := requiredURLParam(w, r, "request_ulid")
	if !ok {
		return
	}
	res, err := h.Creds.GetPdfRequestDetail(r.Context(), &gcredspb.GetPdfRequestRequest{RequestUlid: requestULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

// CreateProgPdfRequest POST /api/prog/pipelines/{pipeline_ulid}/pdf-requests
func (h *Handler) CreateProgPdfRequest(w http.ResponseWriter, r *http.Request) {
	pipelineULID, ok := requiredURLParam(w, r, "pipeline_ulid")
	if !ok {
		return
	}
	var req gcredspb.CreatePdfRequestRequest
	if err := ReadLargeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if strings.TrimSpace(req.RequestUlid) == "" {
		req.RequestUlid = newLmsID()
	}
	if strings.TrimSpace(req.BusinessUnit) == "" {
		req.BusinessUnit = "gprog"
	}
	if strings.TrimSpace(req.ExtRefId) == "" {
		req.ExtRefId = pipelineULID
	}
	if !requireRequestFields(w,
		req.RequestUlid, "request_ulid",
		req.BusinessUnit, "business_unit",
		req.CandidateUlid, "candidate_ulid",
		req.CredDefUlid, "cred_def_ulid",
		req.DegreeNo, "degree_no",
		req.TemplateUlid, "template_ulid",
		req.TemplateParams, "template_params",
	) {
		return
	}

	res, err := h.Creds.CreatePdfRequest(r.Context(), &req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

// ListPdfRequests GET /api/pdf-requests
func (h *Handler) ListPdfRequests(w http.ResponseWriter, r *http.Request) {
	page := parseCursorPage(r, 20)

	req := &gcredspb.ListPdfRequestsRequest{
		Cursor:   page.Cursor,
		PageSize: page.PageSize,
	}

	res, err := h.Creds.ListPdfRequests(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, res)
}
