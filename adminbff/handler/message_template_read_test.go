package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"google.golang.org/grpc"
)

type messageTemplateReadClientStub struct {
	gmsgpb.MessageServiceClient
	listRequest   *gmsgpb.ListTemplatesRequest
	countRequest  *gmsgpb.GetTemplateCountRequest
	detailRequest *gmsgpb.GetTemplateDetailRequest
}

func (s *messageTemplateReadClientStub) ListTemplates(_ context.Context, req *gmsgpb.ListTemplatesRequest, _ ...grpc.CallOption) (*gmsgpb.ListTemplatesResponse, error) {
	s.listRequest = req
	return &gmsgpb.ListTemplatesResponse{
		Templates:  []*gmsgpb.TemplateSummary{{Path: "system/regression", Description: "Regression notice", Version: 3}},
		HasMore:    true,
		NextCursor: "next-page",
	}, nil
}

func (s *messageTemplateReadClientStub) GetTemplateCount(_ context.Context, req *gmsgpb.GetTemplateCountRequest, _ ...grpc.CallOption) (*gmsgpb.GetTemplateCountResponse, error) {
	s.countRequest = req
	return &gmsgpb.GetTemplateCountResponse{Count: 1}, nil
}

func (s *messageTemplateReadClientStub) GetTemplateDetail(_ context.Context, req *gmsgpb.GetTemplateDetailRequest, _ ...grpc.CallOption) (*gmsgpb.Template, error) {
	s.detailRequest = req
	return &gmsgpb.Template{
		Path:            "system/regression",
		TitleTpl:        "Regression {{.name}}",
		ContentTpl:      "Read-only notification",
		Description:     "Regression notice",
		ParameterSchema: `{"type":"object"}`,
		Version:         3,
	}, nil
}

func TestListMessageTemplatesReturnsReadOnlyPage(t *testing.T) {
	client := &messageTemplateReadClientStub{}
	h := &Handler{Gmsg: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/messages/templates?keyword=regression&cursor=current-page&page_size=10", nil)

	h.ListTemplates(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest.GetFilters().GetKeyword() != "regression" || client.listRequest.GetCursor() != "current-page" || client.listRequest.GetPageSize() != 10 || client.countRequest.GetFilters().GetKeyword() != "regression" {
		t.Fatalf("template requests = %+v / %+v", client.listRequest, client.countRequest)
	}
	var payload struct {
		Data struct {
			Total      uint32 `json:"total"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
			Templates  []struct {
				Path    string `json:"path"`
				Version uint32 `json:"version"`
			} `json:"templates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 1 || !payload.Data.HasMore || payload.Data.NextCursor != "next-page" || len(payload.Data.Templates) != 1 || payload.Data.Templates[0].Path != "system/regression" || payload.Data.Templates[0].Version != 3 {
		t.Fatalf("message template page = %+v", payload.Data)
	}
}

func TestGetMessageTemplateReturnsReadOnlyDetail(t *testing.T) {
	client := &messageTemplateReadClientStub{}
	h := &Handler{Gmsg: client}
	recorder := httptest.NewRecorder()

	h.GetMessageTemplate(recorder, httptest.NewRequest(http.MethodGet, "/api/messages/templates/detail?path=system%2Fregression", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest.GetPath() != "system/regression" {
		t.Fatalf("detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			Path            string `json:"path"`
			TitleTemplate   string `json:"title_tpl"`
			ContentTemplate string `json:"content_tpl"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Path != "system/regression" || payload.Data.TitleTemplate == "" || payload.Data.ContentTemplate != "Read-only notification" {
		t.Fatalf("message template detail = %+v", payload.Data)
	}
}
