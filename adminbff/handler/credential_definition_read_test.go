package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type credentialDefinitionReadClientStub struct {
	gcredspb.CredentialServiceClient
	listCalled    bool
	detailRequest *gcredspb.GetCredentialDefinitionDetailRequest
}

func (s *credentialDefinitionReadClientStub) ListCredentialDefinitions(
	_ context.Context,
	_ *gcredspb.ListCredentialDefinitionsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.ListCredentialDefinitionsResponse, error) {
	s.listCalled = true
	return &gcredspb.ListCredentialDefinitionsResponse{
		Definitions: []*gcredspb.CredentialDefinitionSummary{{
			CredDefUlid: "credential-1",
			Name:        "Regression Credential",
			Category:    "Certification",
		}},
	}, nil
}

func (s *credentialDefinitionReadClientStub) GetCredentialDefinitionDetail(
	_ context.Context,
	req *gcredspb.GetCredentialDefinitionDetailRequest,
	_ ...grpc.CallOption,
) (*gcredspb.CredentialDefinition, error) {
	s.detailRequest = req
	return &gcredspb.CredentialDefinition{
		CredDefUlid:       req.GetCredDefUlid(),
		Name:              "Regression Credential",
		Category:          "Certification",
		Description:       "Read-only regression definition",
		Respath:           "/gcc/credential/regression",
		AcquisitionMethod: "Pass the regression assessment",
	}, nil
}

func TestListCredentialDefinitionsReturnsReadOnlyDefinitions(t *testing.T) {
	client := &credentialDefinitionReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()

	h.ListCredentialDefinitions(recorder, httptest.NewRequest(http.MethodGet, "/api/credentials/definitions", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if !client.listCalled {
		t.Fatal("ListCredentialDefinitions() did not call the credential service")
	}
	var payload struct {
		Data struct {
			Definitions []struct {
				ID       string `json:"cred_def_ulid"`
				Name     string `json:"name"`
				Category string `json:"category"`
			} `json:"definitions"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Definitions) != 1 || payload.Data.Definitions[0].ID != "credential-1" || payload.Data.Definitions[0].Name != "Regression Credential" {
		t.Fatalf("credential definitions = %+v", payload.Data.Definitions)
	}
}

func TestGetCredentialDefinitionDetailReturnsReadOnlyDefinition(t *testing.T) {
	client := &credentialDefinitionReadClientStub{}
	h := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/credentials/definitions/credential-1", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("cred_def_ulid", "credential-1")
	request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

	h.GetCredentialDefinitionDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest == nil || client.detailRequest.GetCredDefUlid() != "credential-1" {
		t.Fatalf("detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			ID          string `json:"cred_def_ulid"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Respath     string `json:"respath"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.ID != "credential-1" || payload.Data.Name != "Regression Credential" || payload.Data.Description != "Read-only regression definition" || payload.Data.Respath != "/gcc/credential/regression" {
		t.Fatalf("credential detail = %+v", payload.Data)
	}
}
