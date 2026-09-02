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

func TestIsBuiltInMessageTemplate(t *testing.T) {
	tests := []struct {
		name string
		path string
		list []*gmsgpb.BuiltInPathInfo
		want bool
	}{
		{
			name: "matching built-in path",
			path: "/msg/course/completed/gprog",
			list: []*gmsgpb.BuiltInPathInfo{{Path: "/msg/course/completed/gprog"}},
			want: true,
		},
		{
			name: "custom path",
			path: "/msg/custom/course-completed",
			list: []*gmsgpb.BuiltInPathInfo{{Path: "/msg/course/completed/gprog"}},
			want: false,
		},
		{name: "empty list", path: "/msg/course/completed/gprog", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBuiltInMessageTemplate(tt.path, tt.list); got != tt.want {
				t.Fatalf("isBuiltInMessageTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

type messageTemplateClientStub struct {
	gmsgpb.MessageServiceClient
	updateRequest *gmsgpb.UpdateTemplateRequest
}

func (s *messageTemplateClientStub) GetAllBuiltInPaths(
	_ context.Context,
	_ *gmsgpb.GetAllBuiltInPathsRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.GetAllBuiltInPathsResponse, error) {
	return &gmsgpb.GetAllBuiltInPathsResponse{
		Paths: []*gmsgpb.BuiltInPathInfo{{Path: "/msg/course/completed/gprog"}},
	}, nil
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
		"path":"/msg/course/completed/gprog",
		"title_tpl":"Course completed",
		"content_tpl":"Open {{CandidatePortalBaseURL}}/profile/pipelines",
		"parameter_schema":"{}"
	}`))

	h.UpdateTemplate(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.updateRequest == nil {
		t.Fatal("UpdateTemplate() did not call the message service")
	}
	if client.updateRequest.GetParameterSchema() != "" {
		t.Fatalf("parameter_schema = %q, want empty", client.updateRequest.GetParameterSchema())
	}
}
