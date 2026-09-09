package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"google.golang.org/grpc"
)

type userStatusMembershipClient struct {
	gmbrpb.GmbrServiceClient
	activeRequest *gmbrpb.GetActiveMembershipRequest
	planRequest   *gmbrpb.GetMembershipRequest
	active        *gmbrpb.GetActiveMembershipResponse
	plan          *gmbrpb.Membership
	err           error
}

func (client *userStatusMembershipClient) GetActiveMembership(
	_ context.Context,
	request *gmbrpb.GetActiveMembershipRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.GetActiveMembershipResponse, error) {
	client.activeRequest = request
	if client.err != nil {
		return nil, client.err
	}
	return client.active, nil
}

func (client *userStatusMembershipClient) GetMembership(
	_ context.Context,
	request *gmbrpb.GetMembershipRequest,
	_ ...grpc.CallOption,
) (*gmbrpb.Membership, error) {
	client.planRequest = request
	return client.plan, nil
}

type userStatusProgClient struct {
	gprogpb.ProgServiceClient
	request   *gprogpb.ListCandidatePipelinesReq
	pipelines *gprogpb.ListCandidatePipelinesRsp
	err       error
}

func (client *userStatusProgClient) ListCandidatePipelines(
	_ context.Context,
	request *gprogpb.ListCandidatePipelinesReq,
	_ ...grpc.CallOption,
) (*gprogpb.ListCandidatePipelinesRsp, error) {
	client.request = request
	if client.err != nil {
		return nil, client.err
	}
	return client.pipelines, nil
}

type userStatusConfigClient struct {
	gccpb.CCServiceClient
	request *gccpb.GetPipelineRequest
	config  *gccpb.PipelineConfig
}

func (client *userStatusConfigClient) GetPipeline(
	_ context.Context,
	request *gccpb.GetPipelineRequest,
	_ ...grpc.CallOption,
) (*gccpb.PipelineConfig, error) {
	client.request = request
	return client.config, nil
}

type userStatusCredentialClient struct {
	gcredspb.CredentialServiceClient
	request *gcredspb.GetCandidateCredentialCountRequest
	count   uint32
	err     error
}

func (client *userStatusCredentialClient) GetCandidateCredentialCount(
	_ context.Context,
	request *gcredspb.GetCandidateCredentialCountRequest,
	_ ...grpc.CallOption,
) (*gcredspb.GetCandidateCredentialCountResponse, error) {
	client.request = request
	if client.err != nil {
		return nil, client.err
	}
	return &gcredspb.GetCandidateCredentialCountResponse{Count: client.count}, nil
}

func userMeStatusRequest() *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	return request.WithContext(WithCandidate(request.Context(), "candidate-1", "candidate@example.com", "candidate", "token"))
}

func TestGetUserMeAggregatesAccountStatusFromServices(t *testing.T) {
	membershipClient := &userStatusMembershipClient{
		active: &gmbrpb.GetActiveMembershipResponse{Membership: &gmbrpb.UserMembership{
			MembershipRecordUlid: "membership-record-1",
			MembershipUlid:       "membership-plan-1",
			MembershipGpath:      "/memberships/fellow",
			Status:               "active",
			ExpiresAt:            "2027-09-09T00:00:00Z",
		}},
		plan: &gmbrpb.Membership{
			MembershipUlid:  "membership-plan-1",
			MembershipGpath: "/memberships/fellow",
			Name:            "CFtP Fellow",
			TierLevel:       3,
		},
	}
	progClient := &userStatusProgClient{pipelines: &gprogpb.ListCandidatePipelinesRsp{
		Pipelines: []*gprogpb.PipelineSummary{{
			PipelineUlid:   "pipeline-1",
			PipelineCcUlid: "pipeline-config-1",
			Status:         gprogpb.PipelineStatus_PIPELINE_STATUS_RUNNING,
		}},
	}}
	configClient := &userStatusConfigClient{config: &gccpb.PipelineConfig{
		PipelineUlid:  "pipeline-config-1",
		PipelineGpath: "/pipelines/cftp",
		Name:          "CFtP",
	}}
	credentialClient := &userStatusCredentialClient{count: 2}
	handler := &Handler{
		profileUsers: &fakeProfileUserStore{current: &casdoorsdk.User{Name: "candidate", Email: "candidate@example.com"}},
		Gmbr:         membershipClient,
		Gprog:        progClient,
		Gcc:          configClient,
		Creds:        credentialClient,
	}
	recorder := httptest.NewRecorder()

	handler.GetUserMe(recorder, userMeStatusRequest())

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if membershipClient.activeRequest.GetCandidateUlid() != "candidate-1" ||
		membershipClient.planRequest.GetMembershipUlid() != "membership-plan-1" {
		t.Fatalf("membership requests = active:%#v plan:%#v", membershipClient.activeRequest, membershipClient.planRequest)
	}
	if progClient.request.GetCandidateUlid() != "candidate-1" || configClient.request.GetPipelineUlid() != "pipeline-config-1" {
		t.Fatalf("certification requests = pipelines:%#v config:%#v", progClient.request, configClient.request)
	}
	if credentialClient.request.GetCandidateUlid() != "candidate-1" || credentialClient.request.GetLimit() != 1000 {
		t.Fatalf("credential request = %#v", credentialClient.request)
	}

	var response struct {
		Data UserMeRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	status := response.Data.AccountStatus
	if !status.Membership.Available || !status.Membership.IsMember ||
		status.Membership.PlanName != "CFtP Fellow" || status.Membership.TierLevel != 3 {
		t.Fatalf("membership status = %#v", status.Membership)
	}
	if !status.Certification.Available || !status.Certification.IsCandidate ||
		status.Certification.PurchaseCount != 1 || len(status.Certification.Programs) != 1 ||
		status.Certification.Programs[0].Name != "CFtP" ||
		status.Certification.Programs[0].PipelineGpath != "/pipelines/cftp" {
		t.Fatalf("certification status = %#v", status.Certification)
	}
	if !status.Qualification.Available || !status.Qualification.HasQualification || status.Qualification.CredentialCount != 2 {
		t.Fatalf("qualification status = %#v", status.Qualification)
	}
}

func TestGetUserMeMarksFailedStatusSourcesUnavailable(t *testing.T) {
	serviceError := errors.New("service unavailable")
	handler := &Handler{
		profileUsers: &fakeProfileUserStore{current: &casdoorsdk.User{Name: "candidate"}},
		Gmbr:         &userStatusMembershipClient{err: serviceError},
		Gprog:        &userStatusProgClient{err: serviceError},
		Creds:        &userStatusCredentialClient{count: 0},
	}
	recorder := httptest.NewRecorder()

	handler.GetUserMe(recorder, userMeStatusRequest())

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		Data UserMeRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.AccountStatus.Membership.Available || response.Data.AccountStatus.Certification.Available {
		t.Fatalf("failed status sources must remain unavailable: %#v", response.Data.AccountStatus)
	}
	if !response.Data.AccountStatus.Qualification.Available || response.Data.AccountStatus.Qualification.HasQualification {
		t.Fatalf("qualification status = %#v", response.Data.AccountStatus.Qualification)
	}
}

func TestGetUserMeDistinguishesAvailableNegativeStatuses(t *testing.T) {
	membershipClient := &userStatusMembershipClient{active: &gmbrpb.GetActiveMembershipResponse{}}
	handler := &Handler{
		profileUsers: &fakeProfileUserStore{current: &casdoorsdk.User{Name: "candidate"}},
		Gmbr:         membershipClient,
		Gprog:        &userStatusProgClient{pipelines: &gprogpb.ListCandidatePipelinesRsp{}},
		Creds:        &userStatusCredentialClient{count: 0},
	}
	recorder := httptest.NewRecorder()

	handler.GetUserMe(recorder, userMeStatusRequest())

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		Data UserMeRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	status := response.Data.AccountStatus
	if !status.Membership.Available || status.Membership.IsMember || membershipClient.planRequest != nil {
		t.Fatalf("membership status = %#v; plan request = %#v", status.Membership, membershipClient.planRequest)
	}
	if !status.Certification.Available || status.Certification.IsCandidate || status.Certification.PurchaseCount != 0 || len(status.Certification.Programs) != 0 {
		t.Fatalf("certification status = %#v", status.Certification)
	}
	if !status.Qualification.Available || status.Qualification.HasQualification || status.Qualification.CredentialCount != 0 {
		t.Fatalf("qualification status = %#v", status.Qualification)
	}
}
