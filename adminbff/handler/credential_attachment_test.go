package handler

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type credentialAttachmentClientStub struct {
	gcredspb.CredentialServiceClient
	uploadRequest *gcredspb.AdminRequestDefinitionAttachmentUploadUrlRequest
	updateRequest *gcredspb.AdminUpdateCredentialDefinitionAttachmentsRequest
}

func (s *credentialAttachmentClientStub) AdminRequestDefinitionAttachmentUploadUrl(_ context.Context, request *gcredspb.AdminRequestDefinitionAttachmentUploadUrlRequest, _ ...grpc.CallOption) (*gcredspb.AdminRequestDefinitionAttachmentUploadUrlResponse, error) {
	s.uploadRequest = request
	return &gcredspb.AdminRequestDefinitionAttachmentUploadUrlResponse{
		UploadUrl:     "https://uploads.example.test/credential-template",
		FileKey:       "credential-definitions/definition-route/template.docx",
		SignedHeaders: map[string]string{"content-type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
	}, nil
}

func (s *credentialAttachmentClientStub) AdminUpdateCredentialDefinitionAttachments(_ context.Context, request *gcredspb.AdminUpdateCredentialDefinitionAttachmentsRequest, _ ...grpc.CallOption) (*gcredspb.CredentialDefinitionSummary, error) {
	s.updateRequest = request
	return &gcredspb.CredentialDefinitionSummary{CredDefUlid: request.GetCredDefUlid(), Name: "Work Experience"}, nil
}

func credentialAttachmentRequest(method, target, body, definitionID string) *http.Request {
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("cred_def_ulid", definitionID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
}

func TestRequestCredentialDefinitionAttachmentUploadURLUsesRouteDefinition(t *testing.T) {
	client := &credentialAttachmentClientStub{}
	recorder := httptest.NewRecorder()
	request := credentialAttachmentRequest(http.MethodPost, "/api/credentials/definitions/definition-route/attachments/upload-url", `{
		"file_name":" work-experience.docx ",
		"file_ext":" .docx ",
		"content_type":" application/vnd.openxmlformats-officedocument.wordprocessingml.document ",
		"file_hash":" sha256-template "
	}`, "definition-route")

	(&Handler{Creds: client}).RequestCredentialDefinitionAttachmentUploadURL(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	got := client.uploadRequest
	if got == nil || got.GetCredDefUlid() != "definition-route" || got.GetFileName() != "work-experience.docx" || got.GetFileExt() != "docx" || got.GetFileHash() != "sha256-template" {
		t.Fatalf("upload request = %+v", got)
	}
}

func TestUpdateCredentialDefinitionAttachmentsForwardsCompleteList(t *testing.T) {
	client := &credentialAttachmentClientStub{}
	recorder := httptest.NewRecorder()
	request := credentialAttachmentRequest(http.MethodPut, "/api/credentials/definitions/definition-route/attachments", `{
		"cred_def_ulid":"definition-body-must-not-win",
		"attachments":[{
			"name":" Work experience template ",
			"description":" Complete and sign ",
			"file_name":" work-experience.docx ",
			"file_type":8,
			"file_ext":" .docx ",
			"file_size":2048,
			"file_hash":" sha256-template ",
			"file_key":" credential-definitions/definition-route/template.docx "
		}]
	}`, "definition-route")

	(&Handler{Creds: client}).UpdateCredentialDefinitionAttachments(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	got := client.updateRequest
	if got == nil || got.GetCredDefUlid() != "definition-route" || len(got.GetAttachments()) != 1 {
		t.Fatalf("update request = %+v", got)
	}
	attachment := got.GetAttachments()[0]
	if attachment.GetName() != "Work experience template" || attachment.GetFileExt() != "docx" || attachment.GetFileType() != gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_TEXT || attachment.GetFileKey() != "credential-definitions/definition-route/template.docx" {
		t.Fatalf("attachment = %+v", attachment)
	}
}
