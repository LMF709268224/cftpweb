package handler

import (
	"net/http"
	"strings"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	"github.com/go-chi/chi/v5"
)

func (h *Handler) GetPipelineTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "pipeline_id"))
	if !requireRequestField(w, targetID, "pipeline_id") {
		return
	}
	resp, err := h.Gcc.GetPipelineTranslations(r.Context(), &gccpb.GetPipelineTranslationsRequest{
		PipelineId: targetID,
		Locale:     strings.TrimSpace(r.URL.Query().Get("locale")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetPipelineTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "pipeline_id"))
	var req gccpb.SetPipelineTranslationsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.PipelineId = targetID
	if !requireRequestField(w, req.PipelineId, "pipeline_id") {
		return
	}
	if _, err := h.Gcc.SetPipelineTranslations(r.Context(), &req); err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"translations": req.Translations})
}

func (h *Handler) GetStageTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "stage_id"))
	if !requireRequestField(w, targetID, "stage_id") {
		return
	}
	resp, err := h.Gcc.GetStageTranslations(r.Context(), &gccpb.GetStageTranslationsRequest{
		StageId: targetID,
		Locale:  strings.TrimSpace(r.URL.Query().Get("locale")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetStageTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "stage_id"))
	var req gccpb.SetStageTranslationsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.StageId = targetID
	if !requireRequestField(w, req.StageId, "stage_id") {
		return
	}
	if _, err := h.Gcc.SetStageTranslations(r.Context(), &req); err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"translations": req.Translations})
}

func (h *Handler) GetUnitTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "unit_id"))
	if !requireRequestField(w, targetID, "unit_id") {
		return
	}
	resp, err := h.Gcc.GetUnitTranslations(r.Context(), &gccpb.GetUnitTranslationsRequest{
		UnitId: targetID,
		Locale: strings.TrimSpace(r.URL.Query().Get("locale")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetUnitTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "unit_id"))
	var req gccpb.SetUnitTranslationsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.UnitId = targetID
	if !requireRequestField(w, req.UnitId, "unit_id") {
		return
	}
	if _, err := h.Gcc.SetUnitTranslations(r.Context(), &req); err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"translations": req.Translations})
}

func (h *Handler) GetCredDefTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "cred_def_ulid"))
	if !requireRequestField(w, targetID, "cred_def_ulid") {
		return
	}
	resp, err := h.Creds.GetCredDefTranslations(r.Context(), &gcredspb.GetCredDefTranslationsRequest{
		CredDefId: targetID,
		Locale:    strings.TrimSpace(r.URL.Query().Get("locale")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetCredDefTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "cred_def_ulid"))
	var req gcredspb.SetCredDefTranslationsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.CredDefId = targetID
	if !requireRequestField(w, req.CredDefId, "cred_def_ulid") {
		return
	}
	if _, err := h.Creds.SetCredDefTranslations(r.Context(), &req); err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"translations": req.Translations})
}

func (h *Handler) GetPdfTemplateTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "template_id"))
	if !requireRequestField(w, targetID, "template_id") {
		return
	}
	resp, err := h.Creds.GetPdfTemplateTranslations(r.Context(), &gcredspb.GetPdfTemplateTranslationsRequest{
		TemplateId: targetID,
		Locale:     strings.TrimSpace(r.URL.Query().Get("locale")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetPdfTemplateTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "template_id"))
	var req gcredspb.SetPdfTemplateTranslationsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.TemplateId = targetID
	if !requireRequestField(w, req.TemplateId, "template_id") {
		return
	}
	if _, err := h.Creds.SetPdfTemplateTranslations(r.Context(), &req); err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"translations": req.Translations})
}

func (h *Handler) GetBundleTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	if !requireRequestField(w, targetID, "bundle_ulid") {
		return
	}
	resp, err := h.Mall.GetBundleTranslations(r.Context(), &mallpb.GetBundleTranslationsRequest{
		BundleId: targetID,
		Locale:   strings.TrimSpace(r.URL.Query().Get("locale")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetBundleTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "bundle_ulid"))
	var req mallpb.SetBundleTranslationsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.BundleId = targetID
	if !requireRequestField(w, req.BundleId, "bundle_ulid") {
		return
	}
	if _, err := h.Mall.SetBundleTranslations(r.Context(), &req); err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"translations": req.Translations})
}

func (h *Handler) GetMembershipTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "membership_ulid"))
	if !requireRequestField(w, targetID, "membership_ulid") {
		return
	}
	resp, err := h.Gmbr.GetMembershipTranslations(r.Context(), &gmbrpb.GetMembershipTranslationsRequest{
		MembershipId: targetID,
		Locale:       strings.TrimSpace(r.URL.Query().Get("locale")),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) SetMembershipTranslations(w http.ResponseWriter, r *http.Request) {
	targetID := strings.TrimSpace(chi.URLParam(r, "membership_ulid"))
	var req gmbrpb.SetMembershipTranslationsRequest
	if err := ReadJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body: "+err.Error())
		return
	}
	req.MembershipId = targetID
	if !requireRequestField(w, req.MembershipId, "membership_ulid") {
		return
	}
	if _, err := h.Gmbr.SetMembershipTranslations(r.Context(), &req); err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"translations": req.Translations})
}
