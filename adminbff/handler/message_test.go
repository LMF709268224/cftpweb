package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type messageReadClientStub struct {
	gmsgpb.MessageServiceClient
	listRequest  *gmsgpb.ListMessagesAdminRequest
	countRequest *gmsgpb.GetMessageCountAdminRequest
}

func (s *messageReadClientStub) ListMessagesAdmin(
	_ context.Context,
	req *gmsgpb.ListMessagesAdminRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.ListMessagesAdminResponse, error) {
	s.listRequest = req
	return &gmsgpb.ListMessagesAdminResponse{
		NextCursor: "next-page",
		HasMore:    true,
	}, nil
}

func (s *messageReadClientStub) GetMessageCountAdmin(
	_ context.Context,
	req *gmsgpb.GetMessageCountAdminRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.GetMessageCountAdminResponse, error) {
	s.countRequest = req
	return &gmsgpb.GetMessageCountAdminResponse{Count: 4}, nil
}

func TestListSentMessagesReturnsReadOnlyMessagePage(t *testing.T) {
	client := &messageReadClientStub{}
	h := &Handler{Gmsg: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/messages?status=1&page_size=25&cursor=current-page", nil)

	h.ListSentMessages(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil || client.countRequest == nil {
		t.Fatal("ListSentMessages() did not call both read-only message queries")
	}
	if client.listRequest.GetPageSize() != 25 || client.listRequest.GetCursor() != "current-page" {
		t.Fatalf("list request = %+v", client.listRequest)
	}
	if client.listRequest.GetFilters().GetStatus() != gmsgpb.MessageStatus(1) || client.countRequest.GetFilters().GetStatus() != gmsgpb.MessageStatus(1) {
		t.Fatalf("message filters = %v / %v", client.listRequest.GetFilters(), client.countRequest.GetFilters())
	}

	var payload struct {
		Data struct {
			Total      uint32 `json:"total"`
			NextCursor string `json:"next_cursor"`
			HasMore    bool   `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 4 || payload.Data.NextCursor != "next-page" || !payload.Data.HasMore {
		t.Fatalf("message page = %+v", payload.Data)
	}
}

type messageTemplateClientStub struct {
	gmsgpb.MessageServiceClient
	builtInPathRequest *gmsgpb.GetBuiltInPathRequest
	builtInPathError   error
	updateRequest      *gmsgpb.UpdateTemplateRequest
}

func (s *messageTemplateClientStub) GetBuiltInPath(
	_ context.Context,
	req *gmsgpb.GetBuiltInPathRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.GetBuiltInPathResponse, error) {
	s.builtInPathRequest = req
	if s.builtInPathError != nil {
		return nil, s.builtInPathError
	}
	return &gmsgpb.GetBuiltInPathResponse{Info: &gmsgpb.BuiltInPathInfo{Path: req.GetPath()}}, nil
}

func (s *messageTemplateClientStub) UpdateTemplate(
	_ context.Context,
	req *gmsgpb.UpdateTemplateRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.Template, error) {
	s.updateRequest = req
	return &gmsgpb.Template{}, nil
}

func TestUpdateTemplateClearsParameterSchemaForBuiltInTemplate(t *testing.T) {
	client := &messageTemplateClientStub{}
	h := &Handler{Gmsg: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/messages/templates", strings.NewReader(`{
		"path":"/msg/bundle/payment-paid/gmall",
		"title_tpl":"Bundle purchase completed",
		"content_tpl":"Bundle purchase completed",
		"parameter_schema":"{}"
	}`))

	h.UpdateTemplate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.updateRequest == nil {
		t.Fatal("UpdateTemplate() did not call the message service")
	}
	if client.builtInPathRequest == nil || client.builtInPathRequest.GetPath() != "/msg/bundle/payment-paid/gmall" {
		t.Fatalf("GetBuiltInPath() request = %+v", client.builtInPathRequest)
	}
	if client.updateRequest.GetParameterSchema() != "" {
		t.Fatalf("parameter_schema = %q, want empty", client.updateRequest.GetParameterSchema())
	}
}

func TestUpdateTemplatePreservesParameterSchemaForCustomTemplate(t *testing.T) {
	client := &messageTemplateClientStub{builtInPathError: status.Error(codes.NotFound, "not a built-in path")}
	h := &Handler{Gmsg: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/messages/templates", strings.NewReader(`{
		"path":"/msg/custom/course-completed",
		"title_tpl":"Course completed",
		"content_tpl":"Course completed",
		"parameter_schema":"{\"type\":\"object\"}"
	}`))

	h.UpdateTemplate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.updateRequest == nil {
		t.Fatal("UpdateTemplate() did not call the message service")
	}
	if client.updateRequest.GetParameterSchema() != `{"type":"object"}` {
		t.Fatalf("parameter_schema = %q", client.updateRequest.GetParameterSchema())
	}
}
