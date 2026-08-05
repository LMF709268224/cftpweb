package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
)

type dashboardProgClient struct {
	gprogpb.ProgServiceClient

	listRequest   *gprogpb.ListPipelinesReq
	countRequests []*gprogpb.GetPipelineCountRequest
}

func (c *dashboardProgClient) ListPipelines(
	_ context.Context,
	request *gprogpb.ListPipelinesReq,
	_ ...grpc.CallOption,
) (*gprogpb.ListPipelinesRsp, error) {
	c.listRequest = request
	return &gprogpb.ListPipelinesRsp{}, nil
}

func (c *dashboardProgClient) GetPipelineCount(
	_ context.Context,
	request *gprogpb.GetPipelineCountRequest,
	_ ...grpc.CallOption,
) (*gprogpb.GetPipelineCountResponse, error) {
	c.countRequests = append(c.countRequests, request)
	counts := map[gprogpb.PipelineStatus]uint32{
		gprogpb.PipelineStatus_PIPELINE_STATUS_RUNNING:         2,
		gprogpb.PipelineStatus_PIPELINE_STATUS_WAIT_FINAL_ELIG: 3,
		gprogpb.PipelineStatus_PIPELINE_STATUS_ISSUING_CERT:    4,
	}
	return &gprogpb.GetPipelineCountResponse{Count: counts[request.GetFilters().GetStatus()]}, nil
}

type dashboardCredentialClient struct {
	gcredspb.CredentialServiceClient
	request *gcredspb.GetCandidateCredentialCountRequest
}

func (c *dashboardCredentialClient) GetCandidateCredentialCount(
	_ context.Context,
	request *gcredspb.GetCandidateCredentialCountRequest,
	_ ...grpc.CallOption,
) (*gcredspb.GetCandidateCredentialCountResponse, error) {
	c.request = request
	return &gcredspb.GetCandidateCredentialCountResponse{Count: 5}, nil
}

type dashboardMessageClient struct {
	gmsgpb.MessageServiceClient
	request *gmsgpb.GetMessageCountRequest
}

func (c *dashboardMessageClient) GetMessageCount(
	_ context.Context,
	request *gmsgpb.GetMessageCountRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.GetMessageCountResponse, error) {
	c.request = request
	return &gmsgpb.GetMessageCountResponse{Count: 6}, nil
}

func TestDashboardAggregatesCandidateScopedStats(t *testing.T) {
	progClient := &dashboardProgClient{}
	credentialClient := &dashboardCredentialClient{}
	messageClient := &dashboardMessageClient{}
	handler := &Handler{
		Gprog: progClient,
		Creds: credentialClient,
		Gmsg:  messageClient,
	}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(http.MethodGet, "/api/dashboard", "", "candidate-1", nil)
	request = request.WithContext(WithCandidate(request.Context(), "candidate-1", "", "Candidate One", ""))

	handler.Dashboard(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if progClient.listRequest.GetFilters().GetCandidateUlid() != "candidate-1" ||
		progClient.listRequest.GetPageSize() != 3 {
		t.Fatalf("pipeline list request = %#v", progClient.listRequest)
	}
	if len(progClient.countRequests) != 3 {
		t.Fatalf("pipeline count calls = %d, want 3", len(progClient.countRequests))
	}
	for _, request := range progClient.countRequests {
		if request.GetFilters().GetCandidateUlid() != "candidate-1" || request.GetLimit() != 1000 {
			t.Fatalf("pipeline count request = %#v", request)
		}
	}
	if credentialClient.request.GetCandidateUlid() != "candidate-1" ||
		credentialClient.request.GetLimit() != 1000 {
		t.Fatalf("credential count request = %#v", credentialClient.request)
	}
	if messageClient.request.GetFilters().GetUserUlid() != "candidate-1" ||
		messageClient.request.GetFilters().GetStatus() != gmsgpb.MessageStatus_UNREAD ||
		messageClient.request.GetLimit() != 99 {
		t.Fatalf("message count request = %#v", messageClient.request)
	}

	var response struct {
		Data DashboardRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.CandidateName != "Candidate One" ||
		response.Data.Stats.CoursesInProgress != 9 ||
		response.Data.Stats.CertificationsEarned != 5 ||
		response.Data.Stats.MembershipLevel != "Standard" ||
		response.Data.UnreadMessagesCount != 6 {
		t.Fatalf("dashboard response = %#v", response.Data)
	}
}
