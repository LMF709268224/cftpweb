package handler

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"

	gmailpb "github.com/afnandelfin620-star/cftptest/cftp/gmail"
)

type SendMailInput struct {
	ToEmail      string `json:"to_email"`
	ToName       string `json:"to_name"`
	Subject      string `json:"subject"`
	TemplateId   string `json:"template_id"`
	TemplatePath string `json:"template_path"`
	Payload      string `json:"payload"`
	HtmlBody     string `json:"html_body"`
	PlainBody    string `json:"plain_body"`
	IsHtml       bool   `json:"is_html"`
}

type mailTemplateInput struct {
	Path            string `json:"path"`
	TemplateId      string `json:"template_id"`
	BusinessUnit    string `json:"business_unit"`
	Name            string `json:"name"`
	SubjectTemplate string `json:"subject_template"`
	HtmlBody        string `json:"html_body"`
	PlainBody       string `json:"plain_body"`
	TemplateBody    string `json:"template_body"`
	Description     string `json:"description"`
}

var tplVarRegex = regexp.MustCompile(`{{\s*([a-zA-Z0-9_]+)\s*}}`)

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) normalizedMailTemplatePath(input mailTemplateInput) string {
	path := firstNonEmpty(input.Path, input.TemplateId)
	if path != "" {
		return path
	}
	return fmt.Sprintf("mail_%d", time.Now().Unix())
}

func normalizeTemplateSyntax(value string) string {
	return tplVarRegex.ReplaceAllString(value, "{{.$1}}")
}

func (h *Handler) SendMail(w http.ResponseWriter, r *http.Request) {
	var input SendMailInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	templatePath := firstNonEmpty(input.TemplatePath, input.TemplateId)
	subject := input.Subject
	if templatePath != "" && subject == "" {
		renderResp, err := h.Gmail.RenderTemplate(r.Context(), &gmailpb.RenderTemplateRequest{
			TemplatePath:       templatePath,
			TemplateParamsJson: input.Payload,
			SubjectParamsJson:  input.Payload,
		})
		if err == nil && renderResp != nil && renderResp.GetSubject() != "" {
			subject = renderResp.GetSubject()
		} else {
			subject = "Template Mail"
		}
	}
	if !requireRequestFields(w, input.ToEmail, "to_email", subject, "subject") {
		return
	}

	mailID := ulid.Make().String()
	var resp *gmailpb.CreateMailResponse
	var err error
	if templatePath != "" {
		resp, err = h.Gmail.CreateMail(r.Context(), &gmailpb.CreateMailRequest{
			MailId:       mailID,
			BusinessUnit: "adminserver",
			ToEmail:      input.ToEmail,
			ToName:       input.ToName,
			Subject:      subject,
			Priority:     0,
			TemplatePath: templatePath,
			Payload:      input.Payload,
		})
	} else {
		htmlBody := input.HtmlBody
		plainBody := input.PlainBody
		if htmlBody == "" && input.IsHtml {
			htmlBody = input.Payload
		}
		if plainBody == "" && !input.IsHtml {
			plainBody = input.Payload
		}
		if !requireRequestField(w, firstNonEmpty(htmlBody, plainBody), "html_body or plain_body") {
			return
		}
		resp, err = h.Gmail.CreateMailRaw(r.Context(), &gmailpb.CreateMailRawRequest{
			MailId:       mailID,
			BusinessUnit: "adminserver",
			ToEmail:      input.ToEmail,
			ToName:       input.ToName,
			Subject:      subject,
			Priority:     0,
			HtmlBody:     htmlBody,
			PlainBody:    plainBody,
		})
	}

	if err != nil {
		slog.Error("SendMail failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMail(w http.ResponseWriter, r *http.Request) {
	mailID := r.URL.Query().Get("mail_id")
	if !requireRequestField(w, mailID, "mail_id") {
		return
	}
	resp, err := h.Gmail.GetMail(r.Context(), &gmailpb.GetMailRequest{MailId: mailID})
	if err != nil {
		slog.Error("GetMail failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

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
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMailStatus(w http.ResponseWriter, r *http.Request) {
	mailID := r.URL.Query().Get("mail_id")
	if !requireRequestField(w, mailID, "mail_id") {
		return
	}
	resp, err := h.Gmail.GetMailStatus(r.Context(), &gmailpb.GetMailStatusRequest{MailId: mailID})
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
	if !requireRequestField(w, req.MailId, "mail_id") {
		return
	}
	resp, err := h.Gmail.CancelMail(r.Context(), &gmailpb.CancelMailRequest{MailId: req.MailId})
	if err != nil {
		slog.Error("CancelMail failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) CreateMailTemplate(w http.ResponseWriter, r *http.Request) {
	var input mailTemplateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	if input.Path == "" && input.TemplateId == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "template path is required")
		return
	}

	htmlBody := firstNonEmpty(input.HtmlBody, input.TemplateBody)
	plainBody := firstNonEmpty(input.PlainBody, input.TemplateBody)
	if !requireRequestFields(w, input.Name, "name", input.SubjectTemplate, "subject_template", htmlBody, "html_body") {
		return
	}

	if input.Name == "" {
		input.Name = "mail_" + time.Now().Format("20060102150405")
	}
	if input.Description == "" {
		input.Description = "-"
	}

	resp, err := h.Gmail.CreateTemplate(r.Context(), &gmailpb.CreateTemplateRequest{
		Path:            firstNonEmpty(input.Path, input.TemplateId),
		BusinessUnit:    firstNonEmpty(input.BusinessUnit, "adminserver"),
		Name:            input.Name,
		SubjectTemplate: normalizeTemplateSyntax(input.SubjectTemplate),
		HtmlBody:        normalizeTemplateSyntax(htmlBody),
		PlainBody:       normalizeTemplateSyntax(plainBody),
		Description:     input.Description,
	})
	if err != nil {
		slog.Error("CreateMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) UpdateMailTemplate(w http.ResponseWriter, r *http.Request) {
	var input mailTemplateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}

	path := firstNonEmpty(input.Path, input.TemplateId)
	htmlBody := firstNonEmpty(input.HtmlBody, input.TemplateBody)
	plainBody := firstNonEmpty(input.PlainBody, input.TemplateBody)
	if !requireRequestFields(w, path, "path", input.Name, "name", input.SubjectTemplate, "subject_template", htmlBody, "html_body") {
		return
	}

	resp, err := h.Gmail.UpdateTemplate(r.Context(), &gmailpb.UpdateTemplateRequest{
		Path:            path,
		BusinessUnit:    firstNonEmpty(input.BusinessUnit, "adminserver"),
		Name:            input.Name,
		SubjectTemplate: normalizeTemplateSyntax(input.SubjectTemplate),
		HtmlBody:        normalizeTemplateSyntax(htmlBody),
		PlainBody:       normalizeTemplateSyntax(plainBody),
		Description:     input.Description,
	})
	if err != nil {
		slog.Error("UpdateMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) DeleteMailTemplate(w http.ResponseWriter, r *http.Request) {
	// TODO(microservice-missing-api): gmail does not provide DeleteTemplate yet.
	WriteError(w, http.StatusNotImplemented, ErrNotImplemented, "not implemented in gmail microservice")
}

func (h *Handler) ListMailTemplates(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gmail.GetTemplateList(r.Context(), &gmailpb.GetTemplateListRequest{BusinessUnit: "adminserver"})
	if err != nil {
		slog.Error("ListMailTemplates failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMailTemplate(w http.ResponseWriter, r *http.Request) {
	path := firstNonEmpty(r.URL.Query().Get("path"), r.URL.Query().Get("template_id"))
	if !requireRequestField(w, path, "path") {
		return
	}
	resp, err := h.Gmail.GetTemplate(r.Context(), &gmailpb.GetTemplateRequest{Path: path})
	if err != nil {
		slog.Error("GetMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RenderMailTemplate(w http.ResponseWriter, r *http.Request) {
	var req gmailpb.RenderTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return
	}
	if !requireRequestField(w, req.TemplatePath, "template_path") {
		return
	}
	resp, err := h.Gmail.RenderTemplate(r.Context(), &req)
	if err != nil {
		slog.Error("RenderMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) HasMailTemplate(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if !requireRequestField(w, path, "path") {
		return
	}
	resp, err := h.Gmail.HasTemplate(r.Context(), &gmailpb.HasTemplateRequest{Path: path})
	if err != nil {
		slog.Error("HasMailTemplate failed", "error", err)
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
