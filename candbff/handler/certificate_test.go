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
	listResponse      *gcredspb.ListCandidateCredentialsResponse
	listErr           error
	detailResponse    *gcredspb.Credential
	detailErr         error
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
	if c.listResponse != nil {
		return c.listResponse, nil
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
				Source:        "pdf_cert",
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
	if c.detailErr != nil {
		return nil, c.detailErr
	}
	if c.detailResponse != nil {
		return c.detailResponse, nil
	}
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

func TestListCertificatesExcludesApplicationCredentials(t *testing.T) {
	client := &certificateRegressionClient{
		listResponse: &gcredspb.ListCandidateCredentialsResponse{
			Credentials: []*gcredspb.CredentialSummary{
				{
					CredUlid:      "manual-credential",
					CandidateUlid: "candidate-1",
					CredDefUlid:   "manual-definition",
					Status:        gcredspb.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
					Source:        "application",
				},
				{
					CredUlid:      "certificate-credential",
					CandidateUlid: "candidate-1",
					CredDefUlid:   "certificate-definition",
					Status:        gcredspb.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
					Source:        "pdf_cert",
				},
			},
		},
	}
	handler := &Handler{Creds: client}
	recorder := httptest.NewRecorder()

	handler.ListCertificates(
		recorder,
		newCandidateHandlerRequest(http.MethodGet, "/api/certificates", "", "candidate-1", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
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
	if response.Data.Certificates[0].CredUlid != "certificate-credential" {
		t.Fatalf("credential = %q, want certificate-credential", response.Data.Certificates[0].CredUlid)
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

func TestDownloadCertificateReturnsFreshCandidateScopedRedirect(t *testing.T) {
	client := &certificateRegressionClient{
		detailResponse: &gcredspb.Credential{
			CredUlid:      "01M0HNR3F7ZC2BACKN1X24C9SC",
			CandidateUlid: "candidate-1",
			Files: []*gcredspb.FileInfo{
				{
					FileUsage: "certificate",
					ViewUrl:   "https://files.example.test/fresh-certificate.pdf?signature=new",
				},
			},
		},
	}
	handler := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/certificates/01M0HNR3F7ZC2BACKN1X24C9SC/download",
		"",
		"candidate-1",
		map[string]string{"id": "01M0HNR3F7ZC2BACKN1X24C9SC"},
	)

	handler.DownloadCertificate(recorder, request)

	if recorder.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusTemporaryRedirect, recorder.Body.String())
	}
	if client.detailRequest.GetCredUlid() != "01M0HNR3F7ZC2BACKN1X24C9SC" {
		t.Fatalf("credential detail request = %#v", client.detailRequest)
	}
	if location := recorder.Header().Get("Location"); location != "https://files.example.test/fresh-certificate.pdf?signature=new" {
		t.Fatalf("Location = %q", location)
	}
	if cacheControl := recorder.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Fatalf("Cache-Control = %q, want %q", cacheControl, "no-store")
	}
}

func TestDownloadCertificateRejectsAnotherCandidatesCredential(t *testing.T) {
	client := &certificateRegressionClient{
		detailResponse: &gcredspb.Credential{
			CredUlid:      "01M0HNR3F7ZC2BACKN1X24C9SC",
			CandidateUlid: "candidate-2",
			Files: []*gcredspb.FileInfo{
				{FileUsage: "certificate", ViewUrl: "https://files.example.test/private.pdf"},
			},
		},
	}
	handler := &Handler{Creds: client}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/certificates/01M0HNR3F7ZC2BACKN1X24C9SC/download",
		"",
		"candidate-1",
		map[string]string{"id": "01M0HNR3F7ZC2BACKN1X24C9SC"},
	)

	handler.DownloadCertificate(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusNotFound, ErrNotFound)
	if location := recorder.Header().Get("Location"); location != "" {
		t.Fatalf("Location = %q, want empty", location)
	}
}

func TestDownloadCertificatePropagatesCredentialLookupError(t *testing.T) {
	handler := &Handler{
		Creds: &certificateRegressionClient{
			detailErr: status.Error(codes.Unavailable, "credential service unavailable"),
		},
	}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/certificates/01M0HNR3F7ZC2BACKN1X24C9SC/download",
		"",
		"candidate-1",
		map[string]string{"id": "01M0HNR3F7ZC2BACKN1X24C9SC"},
	)

	handler.DownloadCertificate(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusServiceUnavailable, ErrServiceUnavailable)
}
