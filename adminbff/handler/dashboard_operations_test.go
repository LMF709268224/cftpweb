package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"google.golang.org/grpc"
)

func TestDashboardRevenueUsesUTCAndTwoRollingDays(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	orders := []*gpaypb.OrderSummary{
		{Currency: "usd", Amount: 500, PaidAt: time.Date(2026, 11, 26, 1, 0, 0, 0, time.UTC).Unix()},
		{Currency: "USD", Amount: 200, PaidAt: time.Date(2026, 11, 25, 20, 0, 0, 0, time.UTC).Unix()},
		{Currency: "USD", Amount: 100, PaidAt: time.Date(2026, 11, 24, 20, 0, 0, 0, time.UTC).Unix()},
		{Currency: "USD", Amount: 900, PaidAt: time.Date(2026, 11, 24, 11, 0, 0, 0, time.UTC).Unix()},
		{Currency: "USD", Amount: 900, PaidAt: now.Add(time.Hour).Unix()},
	}
	today, trend := summarizeDashboardRevenue(orders, now)
	if len(today) != 1 || today[0].Currency != "USD" || today[0].AmountMinor != 500 || today[0].OrderCount != 1 {
		t.Fatalf("UTC today = %+v", today)
	}
	if len(trend) != 1 || trend[0].PreviousAmountMinor != 100 || trend[0].CurrentAmountMinor != 700 {
		t.Fatalf("48-hour trend = %+v", trend)
	}
}

func TestDashboardCandidateActivityDoesNotInventMissingSignupDates(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	roles := dashboardRoleConfigFromEnv()
	users := []*casdoorsdk.User{
		{Id: "recent", Roles: []*casdoorsdk.Role{{Name: roles.student}}, CreatedTime: now.Add(-time.Hour).Format(time.RFC3339)},
		{Id: "old", Roles: []*casdoorsdk.Role{{Name: roles.student}}, CreatedTime: now.Add(-72 * time.Hour).Format(time.RFC3339), EmailVerified: true},
		{Id: "staff", Roles: []*casdoorsdk.Role{{Name: roles.admin}}, CreatedTime: now.Format(time.RFC3339)},
	}
	newCandidates, unverified := dashboardCandidateActivity(users, roles, nil, now)
	if newCandidates == nil || *newCandidates != 1 || unverified != 1 {
		t.Fatalf("new=%v unverified=%d", newCandidates, unverified)
	}
	users[0].CreatedTime = ""
	newCandidates, _ = dashboardCandidateActivity(users, roles, nil, now)
	if newCandidates != nil {
		t.Fatalf("missing creation time should make new candidate count unavailable: %d", *newCandidates)
	}
}

type dashboardFunnelProgClient struct {
	gprogpb.ProgServiceClient
	pages []*gprogpb.ListPipelinesRsp
	calls int
}

func (s *dashboardFunnelProgClient) ListPipelines(_ context.Context, request *gprogpb.ListPipelinesReq, _ ...grpc.CallOption) (*gprogpb.ListPipelinesRsp, error) {
	if request.GetPageSize() != adminDashboardPageSize || request.GetSortOrder() != gprogpb.SortOrder_SORT_ORDER_DESC {
		return nil, fmt.Errorf("unexpected request: %+v", request)
	}
	index := s.calls
	s.calls++
	if index >= len(s.pages) {
		return nil, fmt.Errorf("unexpected extra page")
	}
	return s.pages[index], nil
}

func TestDashboardFunnelDeduplicatesCandidatesAcrossPages(t *testing.T) {
	client := &dashboardFunnelProgClient{pages: []*gprogpb.ListPipelinesRsp{
		{Pipelines: []*gprogpb.PipelineSummary{
			{CandidateUlid: "candidate-a", Status: gprogpb.PipelineStatus_PIPELINE_STATUS_RUNNING},
			{CandidateUlid: "candidate-b", Status: gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED},
		}, HasMore: true, NextCursor: "next"},
		{Pipelines: []*gprogpb.PipelineSummary{
			{CandidateUlid: "candidate-a", Status: gprogpb.PipelineStatus_PIPELINE_STATUS_COMPLETED},
		}, HasMore: false},
	}}
	withPipeline, completed, exact, err := (&Handler{Gprog: client}).dashboardPipelineCandidates(context.Background(), map[string]struct{}{"candidate-a": {}, "candidate-b": {}})
	if err != nil || !exact || withPipeline != 2 || completed != 2 || client.calls != 2 {
		t.Fatalf("with=%d completed=%d exact=%v calls=%d err=%v", withPipeline, completed, exact, client.calls, err)
	}
}

func TestDashboardFunnelMarksTruncatedCountsAsLowerBounds(t *testing.T) {
	pages := make([]*gprogpb.ListPipelinesRsp, adminDashboardSampleLimit/adminDashboardPageSize)
	for index := range pages {
		items := make([]*gprogpb.PipelineSummary, adminDashboardPageSize)
		for item := range items {
			items[item] = &gprogpb.PipelineSummary{CandidateUlid: fmt.Sprintf("candidate-%d-%d", index, item)}
		}
		pages[index] = &gprogpb.ListPipelinesRsp{Pipelines: items, HasMore: true, NextCursor: fmt.Sprintf("cursor-%d", index)}
	}
	client := &dashboardFunnelProgClient{pages: pages}
	candidateIDs := make(map[string]struct{}, adminDashboardSampleLimit)
	for _, page := range pages {
		for _, pipeline := range page.Pipelines {
			candidateIDs[pipeline.CandidateUlid] = struct{}{}
		}
	}
	withPipeline, completed, exact, err := (&Handler{Gprog: client}).dashboardPipelineCandidates(context.Background(), candidateIDs)
	if err != nil || exact || withPipeline != adminDashboardSampleLimit || completed != 0 {
		t.Fatalf("with=%d completed=%d exact=%v err=%v", withPipeline, completed, exact, err)
	}
}

type dashboardPendingCredsClient struct {
	gcredspb.CredentialServiceClient
	pages []*gcredspb.ListApplicationsResponse
	calls int
}

func (s *dashboardPendingCredsClient) ListApplications(_ context.Context, request *gcredspb.ListApplicationsRequest, _ ...grpc.CallOption) (*gcredspb.ListApplicationsResponse, error) {
	if len(request.GetFilters().GetStatuses()) != 1 || request.GetFilters().GetStatuses()[0] != "Pending" || request.GetPageSize() != adminDashboardPageSize {
		return nil, fmt.Errorf("unexpected request: %+v", request)
	}
	index := s.calls
	s.calls++
	if index >= len(s.pages) {
		return nil, fmt.Errorf("unexpected extra page")
	}
	return s.pages[index], nil
}

func TestDashboardPendingAgeUsesOnlyPendingCreationTime(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	client := &dashboardPendingCredsClient{pages: []*gcredspb.ListApplicationsResponse{
		{Applications: []*gcredspb.ApplicationSummary{
			{Status: "Pending", CreatedAt: now.Add(-49 * time.Hour).Format(time.RFC3339), UpdateCount: 2},
			{Status: "PENDING", CreatedAt: now.Add(-47 * time.Hour).Format(time.RFC3339)},
		}, HasMore: false},
	}}
	count, exact, err := (&Handler{Creds: client}).dashboardPendingCreatedOlder48h(context.Background(), now)
	if err != nil || !exact || count != 1 || client.calls != 1 {
		t.Fatalf("count=%d exact=%v calls=%d err=%v", count, exact, client.calls, err)
	}
}

func TestDashboardPendingAgeRejectsMissingCreationTime(t *testing.T) {
	client := &dashboardPendingCredsClient{pages: []*gcredspb.ListApplicationsResponse{{Applications: []*gcredspb.ApplicationSummary{{Status: "Pending"}}}}}
	_, _, err := (&Handler{Creds: client}).dashboardPendingCreatedOlder48h(context.Background(), time.Now())
	if err == nil {
		t.Fatal("missing creation time cannot be counted as a fresh application")
	}
}

func TestDashboardPendingAgeReadsBeyondFormerSampleLimit(t *testing.T) {
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	pages := make([]*gcredspb.ListApplicationsResponse, 6)
	for index := range pages {
		pageSize := adminDashboardPageSize
		if index == len(pages)-1 {
			pageSize = 1
		}
		applications := make([]*gcredspb.ApplicationSummary, pageSize)
		for item := range applications {
			applications[item] = &gcredspb.ApplicationSummary{Status: "Pending", CreatedAt: now.Add(-49 * time.Hour).Format(time.RFC3339)}
		}
		pages[index] = &gcredspb.ListApplicationsResponse{Applications: applications, HasMore: index < len(pages)-1, NextCursor: fmt.Sprintf("pending-%d", index)}
	}
	client := &dashboardPendingCredsClient{pages: pages}
	count, exact, err := (&Handler{Creds: client}).dashboardPendingCreatedOlder48h(context.Background(), now)
	if err != nil || !exact || count != 501 || client.calls != 6 {
		t.Fatalf("count=%d exact=%v calls=%d err=%v", count, exact, client.calls, err)
	}
}
