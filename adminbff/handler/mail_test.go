package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gmailpb "github.com/afnandelfin620-star/cftptest/cftp/gmail"
	"google.golang.org/grpc"
)

func TestIsBuiltInMailTemplate(t *testing.T) {
	tests := []struct {
		name string
		path string
		list []*gmailpb.BuiltInPathInfo
		want bool
	}{
		{
			name: "matching built-in path",
			path: "/mail/gprog/certificate-issued",
			list: []*gmailpb.BuiltInPathInfo{{Path: "/mail/gprog/certificate-issued"}},
			want: true,
		},
		{
			name: "custom path",
			path: "/mail/custom/certificate-issued",
			list: []*gmailpb.BuiltInPathInfo{{Path: "/mail/gprog/certificate-issued"}},
			want: false,
		},
		{name: "empty list", path: "/mail/gprog/certificate-issued", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isBuiltInMailTemplate(tt.path, tt.list); got != tt.want {
				t.Fatalf("isBuiltInMailTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mailTemplateClientStub struct {
	gmailpb.MailServiceClient
	builtInPaths  []*gmailpb.BuiltInPathInfo
	updateRequest *gmailpb.UpdateTemplateRequest
}

func (s *mailTemplateClientStub) GetAllBuiltInPaths(
	_ context.Context,
	_ *gmailpb.GetAllBuiltInPathsRequest,
	_ ...grpc.CallOption,
) (*gmailpb.GetAllBuiltInPathsResponse, error) {
	return &gmailpb.GetAllBuiltInPathsResponse{Paths: s.builtInPaths}, nil
}

func (s *mailTemplateClientStub) UpdateTemplate(
	_ context.Context,
	req *gmailpb.UpdateTemplateRequest,
	_ ...grpc.CallOption,
) (*gmailpb.UpdateTemplateResponse, error) {
	s.updateRequest = req
	return &gmailpb.UpdateTemplateResponse{Success: true}, nil
}

func TestUpdateMailTemplateParameterSchema(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		builtInPaths    []*gmailpb.BuiltInPathInfo
		parameterSchema string
		wantSchema      string
	}{
		{
			name:            "built-in template clears schema",
			path:            "/mail/gprog/certificate-issued",
			builtInPaths:    []*gmailpb.BuiltInPathInfo{{Path: "/mail/gprog/certificate-issued"}},
			parameterSchema: "{\"type\":\"object\"}",
			wantSchema:      "",
		},
		{
			name:            "custom template keeps schema",
			path:            "/mail/custom/certificate-issued",
			builtInPaths:    []*gmailpb.BuiltInPathInfo{{Path: "/mail/gprog/certificate-issued"}},
			parameterSchema: "{\"type\":\"object\"}",
			wantSchema:      "{\"type\":\"object\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &mailTemplateClientStub{builtInPaths: tt.builtInPaths}
			h := &Handler{Gmail: client}
			recorder := httptest.NewRecorder()
			body, err := json.Marshal(mailTemplateInput{
				Path:            tt.path,
				Name:            "certificate-issued",
				SubjectTemplate: "Certificate issued",
				HtmlBody:        "<a href=\"{{.CandidatePortalBaseURL}}/certificates\">View certificate</a>",
				ParameterSchema: tt.parameterSchema,
			})
			if err != nil {
				t.Fatalf("marshal request: %v", err)
			}
			request := httptest.NewRequest(http.MethodPut, "/api/mails/templates", bytes.NewReader(body))

			h.UpdateMailTemplate(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			if client.updateRequest == nil {
				t.Fatal("UpdateMailTemplate() did not call the mail service")
			}
			if got := client.updateRequest.GetParameterSchema(); got != tt.wantSchema {
				t.Fatalf("parameter_schema = %q, want %q", got, tt.wantSchema)
			}
		})
	}
}
