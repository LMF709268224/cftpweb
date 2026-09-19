package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	"google.golang.org/grpc"
)

type credentialVerificationClient struct {
	gcredspb.CredentialServiceClient
	validityRequest *gcredspb.CheckCredentialValidityRequest
}

func (c *credentialVerificationClient) GetPrimaryPublicKey(
	context.Context,
	*gcredspb.GetPrimaryPublicKeyRequest,
	...grpc.CallOption,
) (*gcredspb.GetPrimaryPublicKeyResponse, error) {
	return &gcredspb.GetPrimaryPublicKeyResponse{
		PublicKeyPem:            "-----BEGIN CERTIFICATE-----\nroot\n-----END CERTIFICATE-----",
		Algorithm:               "ECDSA_P256",
		KeyId:                   "root-key",
		RevokedLeafFingerprints: []string{"revoked-leaf"},
	}, nil
}

func (c *credentialVerificationClient) CheckCredentialValidity(
	_ context.Context,
	request *gcredspb.CheckCredentialValidityRequest,
	_ ...grpc.CallOption,
) (*gcredspb.CheckCredentialValidityResponse, error) {
	c.validityRequest = request
	return &gcredspb.CheckCredentialValidityResponse{
		Exists:      true,
		IsValid:     true,
		Status:      gcredspb.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
		CredDefName: "CFtP Certificate",
		IssueDate:   "2026-09-19T00:00:00Z",
		ValidUntil:  "2027-09-19T00:00:00Z",
	}, nil
}

func TestGetCredentialVerificationTrustAnchorReturnsPublicData(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&Handler{Creds: &credentialVerificationClient{}}).GetCredentialVerificationTrustAnchor(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/public/test-verify-creds/api/primary-key", nil),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	response := decodeHandlerAPIResponse(t, recorder)
	data, ok := response.Data.(map[string]interface{})
	if !ok || data["algorithm"] != "ECDSA_P256" || data["key_id"] != "root-key" {
		t.Fatalf("unexpected trust anchor response: %+v", response.Data)
	}
}

func TestCheckCredentialVerificationValidityForwardsOnlyULIDAndHash(t *testing.T) {
	client := &credentialVerificationClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(
		http.MethodGet,
		"/api/public/test-verify-creds/api/check-validity?cred_ulid=01J8TESTULID00000000000000&pdf_hash="+"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		nil,
	)
	(&Handler{Creds: client}).CheckCredentialVerificationValidity(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.validityRequest == nil || client.validityRequest.GetCredUlid() != "01J8TESTULID00000000000000" || client.validityRequest.GetPdfHash() == "" {
		t.Fatalf("unexpected gRPC request: %+v", client.validityRequest)
	}
}

func TestCheckCredentialVerificationValidityRequiresBothQueryFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/public/test-verify-creds/api/check-validity?cred_ulid=only", nil)
	(&Handler{Creds: &credentialVerificationClient{}}).CheckCredentialVerificationValidity(recorder, request)
	assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
}
