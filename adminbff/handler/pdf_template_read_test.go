package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"google.golang.org/grpc"
)

type pdfTemplateReadClientStub struct {
	gcredspb.CredentialServiceClient
	listCalled     bool
	summaryRequest *gcredspb.GetPdfTemplateRequest
	detailRequest  *gcredspb.GetPdfTemplateRequest
}

func (s *pdfTemplateReadClientStub) ListPdfTemplates(_ context.Context, _ *gcredspb.ListPdfTemplatesRequest, _ ...grpc.CallOption) (*gcredspb.ListPdfTemplatesResponse, error) {
	s.listCalled = true
	return &gcredspb.ListPdfTemplatesResponse{Templates: []*gcredspb.PdfTemplateSummary{{
		TemplateUlid: "template-1",
		Name:         "Regression Certificate",
		Description:  "Read-only regression template",
		Version:      2,
		CreatedAt:    "2026-08-11T00:00:00Z",
	}}}, nil
}

func (s *pdfTemplateReadClientStub) GetPdfTemplate(_ context.Context, req *gcredspb.GetPdfTemplateRequest, _ ...grpc.CallOption) (*gcredspb.PdfTemplateSummary, error) {
	s.summaryRequest = req
	return &gcredspb.PdfTemplateSummary{TemplateUlid: "template-1", Name: "Regression Certificate", Version: 2}, nil
}

func (s *pdfTemplateReadClientStub) GetPdfTemplateDetail(_ context.Context, req *gcredspb.GetPdfTemplateRequest, _ ...grpc.CallOption) (*gcredspb.PdfTemplateDetail, error) {
	s.detailRequest = req
	return &gcredspb.PdfTemplateDetail{
		TemplateUlid:    "template-1",
		Name:            "Regression Certificate",
		Description:     "Read-only regression template",
		HtmlTemplate:    "<main>{{.candidate_name}}</main>",
		ParameterSchema: `{"type":"object"}`,
		Version:         2,
	}, nil
}

func TestListPdfTemplatesReturnsReadOnlySummaries(t *testing.T) {
	client := &pdfTemplateReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()

	h.ListPdfTemplates(recorder, httptest.NewRequest(http.MethodGet, "/api/pdf-templates", nil))

	if recorder.Code != http.StatusOK || !client.listCalled {
		t.Fatalf("status = %d, list called = %v; body=%s", recorder.Code, client.listCalled, recorder.Body.String())
	}
	var payload struct {
		Data struct {
			Templates []struct {
				TemplateULID string `json:"template_ulid"`
				Name         string `json:"name"`
				Version      uint32 `json:"version"`
			} `json:"templates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Templates) != 1 || payload.Data.Templates[0].TemplateULID != "template-1" || payload.Data.Templates[0].Name != "Regression Certificate" || payload.Data.Templates[0].Version != 2 {
		t.Fatalf("PDF templates = %+v", payload.Data.Templates)
	}
}

func TestGetPdfTemplateDetailMergesReadOnlySummaryAndDetail(t *testing.T) {
	client := &pdfTemplateReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()

	h.GetPdfTemplateDetail(recorder, httptest.NewRequest(http.MethodGet, "/api/pdf-templates/detail?template_id=template-1", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.summaryRequest.GetTemplateUlid() != "template-1" || client.detailRequest.GetTemplateUlid() != "template-1" {
		t.Fatalf("detail requests = %+v / %+v", client.summaryRequest, client.detailRequest)
	}
	var payload struct {
		Data struct {
			TemplateULID    string                 `json:"template_ulid"`
			Name            string                 `json:"name"`
			HtmlTemplate    string                 `json:"html_template"`
			ParameterSchema string                 `json:"parameter_schema"`
			Summary         map[string]interface{} `json:"summary"`
			Detail          map[string]interface{} `json:"detail"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.TemplateULID != "template-1" || payload.Data.Name != "Regression Certificate" || payload.Data.HtmlTemplate == "" || payload.Data.ParameterSchema == "" || payload.Data.Summary["template_ulid"] != "template-1" || payload.Data.Detail["template_ulid"] != "template-1" {
		t.Fatalf("PDF template detail = %+v", payload.Data)
	}
}
