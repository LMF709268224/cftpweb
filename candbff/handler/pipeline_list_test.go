package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
)

type emptyPipelineListProgClient struct {
	gprogpb.ProgServiceClient
}

func (s *emptyPipelineListProgClient) ListCandidatePipelines(
	_ context.Context,
	_ *gprogpb.ListCandidatePipelinesReq,
	_ ...grpc.CallOption,
) (*gprogpb.ListCandidatePipelinesRsp, error) {
	return &gprogpb.ListCandidatePipelinesRsp{}, nil
}

type emptyPipelineListLMSClient struct {
	lmspb.LmsServiceClient
}

func (s *emptyPipelineListLMSClient) ListCandidateEnrollments(
	_ context.Context,
	_ *lmspb.ListCandidateEnrollmentsRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListCandidateEnrollmentsResponse, error) {
	return &lmspb.ListCandidateEnrollmentsResponse{}, nil
}

func TestListMyPipelinesReturnsEmptyList(t *testing.T) {
	handler := &Handler{
		Gprog: &emptyPipelineListProgClient{},
		Lms:   &emptyPipelineListLMSClient{},
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/pipeline", nil)
	request = request.WithContext(WithCandidate(request.Context(), "candidate-1", "", "", ""))

	handler.ListMyPipelines(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}

	var response struct {
		Data map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	list, ok := response.Data["list"]
	if !ok {
		t.Fatal("data.list is missing")
	}
	if string(list) != "[]" {
		t.Fatalf("data.list = %s, want []", list)
	}
}
