package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/oklog/ulid/v2"

	gmailpb "github.com/afnandelfin620-star/cftptest/cftp/gmail"
)

// ── Mails ──

type SendMailInput struct {
	ToEmail    string `json:"to_email"`
	ToName     string `json:"to_name"`
	Subject    string `json:"subject"`
	TemplateId string `json:"template_id"`
	Payload    string `json:"payload"`
	IsHtml     bool   `json:"is_html"`
}

func (h *Handler) SendMail(w http.ResponseWriter, r *http.Request) {
	var input SendMailInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	mailID := ulid.Make().String()

	subject := input.Subject
	if input.TemplateId != "" && subject == "" {
		subject = "Template Mail" // Bypass protobuf validation requiring a non-empty subject
	}

	resp, err := h.Gmail.CreateMail(r.Context(), &gmailpb.CreateMailRequest{
		MailId:       mailID,
		BusinessUnit: "adminserver",
		ToEmail:      input.ToEmail,
		ToName:       input.ToName,
		Subject:      subject,
		Priority:     0, // 默认优先级
		TemplateId:   input.TemplateId,
		Payload:      input.Payload,
		IsHtml:       input.IsHtml,
	})

	if err != nil {
		slog.Error("SendMail failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMail(w http.ResponseWriter, r *http.Request) {
	mailId := r.URL.Query().Get("mail_id")
	if mailId == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "mail_id is required")
		return
	}

	resp, err := h.Gmail.GetMail(r.Context(), &gmailpb.GetMailRequest{
		MailId: mailId,
	})

	if err != nil {
		slog.Error("GetMail failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ListSentMails GET /api/mails/sent
func (h *Handler) ListSentMails(w http.ResponseWriter, r *http.Request) {
	page := 1
	pageSize := 50
	
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := r.URL.Query().Get("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}

	var statusPtr *string
	if status := r.URL.Query().Get("status"); status != "" {
		statusPtr = &status
	}

	resp, err := h.Gmail.ListMails(r.Context(), &gmailpb.ListMailsRequest{
		Page:     uint32(page),
		PageSize: uint32(pageSize),
		Status:   statusPtr,
	})
	if err != nil {
		slog.Error("ListMails failed", "error", err)
		WriteError(w, http.StatusInternalServerError, ErrInternal, "failed to list mails")
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMailStatus(w http.ResponseWriter, r *http.Request) {
	mailId := r.URL.Query().Get("mail_id")
	if mailId == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "mail_id is required")
		return
	}

	resp, err := h.Gmail.GetMailStatus(r.Context(), &gmailpb.GetMailStatusRequest{
		MailId: mailId,
	})

	if err != nil {
		slog.Error("GetMailStatus failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CancelMail(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MailId string `json:"mail_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	resp, err := h.Gmail.CancelMail(r.Context(), &gmailpb.CancelMailRequest{
		MailId: req.MailId,
	})

	if err != nil {
		slog.Error("CancelMail failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

// ── Mail Templates ──

var tplVarRegex = regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)\s*}}`)

func (h *Handler) CreateMailTemplate(w http.ResponseWriter, r *http.Request) {
	var req gmailpb.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	req.BusinessUnit = "adminserver"
	req.TemplateId = ulid.Make().String()

	// Fix Go template syntax by injecting dot for variables like {{name}} -> {{.name}}
	req.SubjectTemplate = tplVarRegex.ReplaceAllString(req.SubjectTemplate, "{{.$1}}")
	req.TemplateBody = tplVarRegex.ReplaceAllString(req.TemplateBody, "{{.$1}}")

	resp, err := h.Gmail.CreateTemplate(r.Context(), &req)
	if err != nil {
		slog.Error("CreateMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateMailTemplate(w http.ResponseWriter, r *http.Request) {
	var req gmailpb.UpdateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	req.BusinessUnit = "adminserver"

	// Fix Go template syntax by injecting dot for variables like {{name}} -> {{.name}}
	req.SubjectTemplate = tplVarRegex.ReplaceAllString(req.SubjectTemplate, "{{.$1}}")
	req.TemplateBody = tplVarRegex.ReplaceAllString(req.TemplateBody, "{{.$1}}")

	resp, err := h.Gmail.UpdateTemplate(r.Context(), &req)
	if err != nil {
		slog.Error("UpdateMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteMailTemplate(w http.ResponseWriter, r *http.Request) {
	// The grpc gmail service does not have a DeleteTemplate method
	// so we return an error. The frontend will catch this and inform the user.

	WriteError(w, http.StatusNotImplemented, ErrNotImplemented, "not implemented in gmail microservice")
}

func (h *Handler) ListMailTemplates(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gmail.GetTemplateList(r.Context(), &gmailpb.GetTemplateListRequest{
		BusinessUnit: "adminserver",
	})

	if err != nil {
		slog.Error("ListMailTemplates failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMailTemplate(w http.ResponseWriter, r *http.Request) {
	templateId := r.URL.Query().Get("template_id")
	if templateId == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "template_id is required")
		return
	}

	resp, err := h.Gmail.GetTemplate(r.Context(), &gmailpb.GetTemplateRequest{
		TemplateId: templateId,
	})

	if err != nil {
		slog.Error("GetMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, resp)
}
