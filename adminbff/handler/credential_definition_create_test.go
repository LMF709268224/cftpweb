package handler

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"google.golang.org/grpc"
)

type credentialDefinitionCreateClientStub struct {
	gcredspb.CredentialServiceClient
	createRequest *gcredspb.CreateCredentialDefinitionRequest
}

func (s *credentialDefinitionCreateClientStub) CreateCredentialDefinition(_ context.Context, request *gcredspb.CreateCredentialDefinitionRequest, _ ...grpc.CallOption) (*gcredspb.CredentialDefinitionSummary, error) {
	s.createRequest = request
	return &gcredspb.CredentialDefinitionSummary{CredDefUlid: request.GetCredDefUlid(), Name: request.GetName()}, nil
}

func TestCreateCredentialDefinitionValidatesFileConstraints(t *testing.T) {
	tests := []struct {
		name    string
		files   string
		message string
	}{
		{"empty", `[]`, "file_constraints must not be empty"},
		{"missing name", `[{"name":"","type":2}]`, "file_constraints[0].name is required"},
		{"unspecified", `[{"name":"Evidence","type":0}]`, "file_constraints[0].type 0 is not a valid type"},
		{"unknown", `[{"name":"Evidence","type":3}]`, "file_constraints[0].type 3 is not a valid type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &credentialDefinitionCreateClientStub{}
			recorder := httptest.NewRecorder()
			body := fmt.Sprintf(`{"name":"Example","category":"Exemption","description":"Example","respath":"/gcc/credential/example","file_constraints":%s}`, tt.files)
			request := httptest.NewRequest(http.MethodPost, "/api/credentials/definitions", bytes.NewBufferString(body))
			(&Handler{Creds: client}).CreateCredentialDefinition(recorder, request)
			if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), tt.message) {
				t.Fatalf("status = %d, body = %s; want error %q", recorder.Code, recorder.Body.String(), tt.message)
			}
			if client.createRequest != nil {
				t.Fatal("invalid request reached credential service")
			}
		})
	}
}

func TestCreateCredentialDefinitionForwardsSupportedFileTypes(t *testing.T) {
	client := &credentialDefinitionCreateClientStub{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/credentials/definitions", bytes.NewBufferString(`{
		"name":"Example","category":"Exemption","description":"Example","respath":"/gcc/credential/example",
		"file_constraints":[{"name":"Image","type":1},{"name":"PDF","type":2},{"name":"Video","type":4},{"name":"Text","type":8}]
	}`))
	(&Handler{Creds: client}).CreateCredentialDefinition(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	got := client.createRequest
	if got == nil || len(got.GetFileConstraints()) != 4 || got.GetCredDefUlid() == "" {
		t.Fatalf("create request = %+v", got)
	}
	for i, want := range []gcredspb.CredentialFileType{
		gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_IMAGE,
		gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_PDF,
		gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_VIDEO,
		gcredspb.CredentialFileType_CREDENTIAL_FILE_TYPE_TEXT,
	} {
		if got.GetFileConstraints()[i].GetType() != want {
			t.Errorf("file constraint %d type = %v, want %v", i, got.GetFileConstraints()[i].GetType(), want)
		}
	}
}
