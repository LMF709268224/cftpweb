package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type certificateRegressionClient struct {
	gcredspb.CredentialServiceClient

	listRequest       *gcredspb.ListCandidateCredentialsRequest
	definitionRequest *gcredspb.GetCredentialDefinitionDetailRequest
	detailRequest     *gcredspb.GetCredentialDetailRequest
	listErr           error
}

func (c *certificateRegressionClient) ListCandidateCredentials(
	_ context.Context,
	request *gcredspb.ListCandidateCredentialsRequest,
	_ ...grpc.CallOption,
) (*gcredspb.ListCandidateCredentialsResponse, error) {
	c.listRequest = request
	if c.listErr != nil {
		return nil, c.listErr
	}
	return &gcredspb.ListCandidateCredentialsResponse{
		Credentials: []*gcredspb.CredentialSummary{
			{
				CredUlid:      "credential-1",
				CredGuid:      "credential-guid-1",
				CandidateUlid: "candidate-1",
				CredDefUlid:   "definition-1",
				Version:       2,
				Status:        gcredspb.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
				ValidUntil:    "2027-08-05T00:00:00Z",
				Source:        "application",
			},
		},
	}, nil
}

func (c *certificateRegressionClient) GetCredentialDefinitionDetail(
	_ context.Context,
	request *gcredspb.GetCredentialDefinitionDetailRequest,
	_ ...grpc.CallOption,
) (*gcredspb.CredentialDefinition, error) {
	c.definitionRequest = request
	return &gcredspb.CredentialDefinition{
		CredDefUlid: request.GetCredDefUlid(),
		Name:        "CFtP",
		Description: "CFtP certificate",
	}, nil
}

func (c *certificateRegressionClient) GetCredentialDetail(
	_ context.Context,
	request *gcredspb.GetCredentialDetailRequest,
	_ ...grpc.CallOption,
) (*gcredspb.Credential, error) {
	c.detailRequest = request
	return &gcredspb.Credential{
		CredUlid: request.GetCredUlid(),
		Files: []*gcredspb.FileInfo{
			nil,
			{
				FileHash:  "hash-1",
				FileName:  "certificate.pdf",
				FileExt:   "pdf",
				FileSize:  1024,
				FileUsage: "certificate",
				ViewUrl:   "https://files.example.test/certificate.pdf",
			},
		},
	}, nil
}

func TestListCertificatesUsesCandidateScopeAndEnrichesDetails(t *testing.T) {
	client := &certificateRegressionClient{}
	handler := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/certificates",
		"",
		"candidate-1",
		nil,
	)

	handler.ListCertificates(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest.GetCandidateUlid() != "candidate-1" ||
		client.listRequest.GetPageSize() != 100 {
		t.Fatalf("list request = %#v", client.listRequest)
	}
	if client.definitionRequest.GetCredDefUlid() != "definition-1" {
		t.Fatalf("definition request = %#v", client.definitionRequest)
	}
	if client.detailRequest.GetCredUlid() != "credential-1" {
		t.Fatalf("detail request = %#v", client.detailRequest)
	}

	var response struct {
		Data ListCertificatesRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data.Certificates) != 1 {
		t.Fatalf("certificates = %d, want 1", len(response.Data.Certificates))
	}
	certificate := response.Data.Certificates[0]
	if certificate.CatalogId != "definition-1" ||
		certificate.Name != "CFtP" ||
		certificate.CredUlid != "credential-1" ||
		certificate.CandidateUlid != "candidate-1" ||
		len(certificate.Files) != 1 ||
		certificate.Files[0].FileName != "certificate.pdf" {
		t.Fatalf("certificate = %#v", certificate)
	}
}

func TestListCertificatesPropagatesCandidateCredentialLookupError(t *testing.T) {
	handler := &Handler{
		Creds: &certificateRegressionClient{
			listErr: status.Error(codes.Unavailable, "credential service unavailable"),
		},
	}
	recorder := httptest.NewRecorder()

	handler.ListCertificates(
		recorder,
		newCandidateHandlerRequest(http.MethodGet, "/api/certificates", "", "candidate-1", nil),
	)

	assertHandlerAPIError(t, recorder, http.StatusServiceUnavailable, ErrServiceUnavailable)
}
