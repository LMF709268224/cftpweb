package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	gcredspb "github.com/afnandelfin620-star/cftptest/cftp/gcreds"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmidpb "github.com/afnandelfin620-star/cftptest/cftp/gmid"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
	"google.golang.org/grpc"
)

type adminReadPayClient struct {
	gpaypb.PayServiceClient
	request      *gpaypb.ListSubscriptionsRequest
	orderRequest *gpaypb.ListOrdersRequest
}

func (s *adminReadPayClient) ListSubscriptions(_ context.Context, request *gpaypb.ListSubscriptionsRequest, _ ...grpc.CallOption) (*gpaypb.ListSubscriptionsResponse, error) {
	s.request = request
	return &gpaypb.ListSubscriptionsResponse{
		Subscriptions: []*gpaypb.SubscriptionSummary{{
			OrderUlid:            "order-regression",
			StripeSubscriptionId: "sub_regression",
			CustomerUlid:         "customer-regression",
			Amount:               1299,
			Currency:             "usd",
			Status:               gpaypb.OrderStatus_ORDER_STATUS_COMPLETED,
		}},
		HasMore:    true,
		NextCursor: "next-subscriptions",
	}, nil
}

func (s *adminReadPayClient) ListOrders(_ context.Context, request *gpaypb.ListOrdersRequest, _ ...grpc.CallOption) (*gpaypb.ListOrdersResponse, error) {
	s.orderRequest = request
	return &gpaypb.ListOrdersResponse{Orders: []*gpaypb.OrderSummary{{
		OrderUlid: "order-regression",
		Currency:  "USD",
		Amount:    12900,
		Status:    gpaypb.OrderStatus_ORDER_STATUS_COMPLETED,
		PaidAt:    time.Now().Unix(),
	}}}, nil
}

type adminReadCredentialClient struct {
	gcredspb.CredentialServiceClient
	request *gcredspb.CheckCandidateQualificationRequest
}

func (s *adminReadCredentialClient) CheckCandidateQualification(_ context.Context, request *gcredspb.CheckCandidateQualificationRequest, _ ...grpc.CallOption) (*gcredspb.CheckCandidateQualificationResponse, error) {
	s.request = request
	return &gcredspb.CheckCandidateQualificationResponse{
		Eligible:         true,
		CredentialStatus: gcredspb.CredentialStatus_CREDENTIAL_STATUS_ACTIVE,
		Message:          "qualification active",
	}, nil
}

type adminReadProgClient struct {
	gprogpb.ProgServiceClient
	request *gprogpb.ListPipelinesReq
}

func (s *adminReadProgClient) ListPipelines(_ context.Context, request *gprogpb.ListPipelinesReq, _ ...grpc.CallOption) (*gprogpb.ListPipelinesRsp, error) {
	s.request = request
	return &gprogpb.ListPipelinesRsp{Pipelines: []*gprogpb.PipelineSummary{{
		PipelineUlid:     "pipeline-regression",
		CandidateUlid:    "candidate-regression",
		CurrentStageUlid: "stage-regression",
		Status:           gprogpb.PipelineStatus_PIPELINE_STATUS_RUNNING,
	}}}, nil
}

type adminReadMallClient struct {
	mallpb.MallServiceClient
	request *mallpb.ListOrdersRequest
}

func (s *adminReadMallClient) ListOrders(_ context.Context, request *mallpb.ListOrdersRequest, _ ...grpc.CallOption) (*mallpb.ListOrdersResponse, error) {
	s.request = request
	return &mallpb.ListOrdersResponse{Items: []*mallpb.OrderSummary{{
		OrderUlid:     "order-regression",
		CandidateUlid: "candidate-regression",
		CurrencyCode:  "USD",
		AmountMinor:   12900,
		PaymentStatus: "PAID",
		CreatedAt:     time.Now().Format(time.RFC3339),
	}}}, nil
}

type adminReadMidClient struct {
	gmidpb.MidServiceClient
}

func (s *adminReadMidClient) GetUlidByUUID(_ context.Context, request *gmidpb.GetUlidByUUIDRequest, _ ...grpc.CallOption) (*gmidpb.GetUlidByUUIDResponse, error) {
	return &gmidpb.GetUlidByUUIDResponse{UserUlid: "candidate-" + request.GetUserUuid()}, nil
}

func TestListPaySubscriptionsForwardsReadFiltersAndCursor(t *testing.T) {
	client := &adminReadPayClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/pay/subscriptions?customer_ulid=customer-regression&status=completed&cursor=current-subscriptions&page_size=25&sort=1", nil)

	(&Handler{Gpay: client}).ListPaySubscriptions(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request.GetFilters().GetCustomerUlid() != "customer-regression" || client.request.GetFilters().GetStatus() != gpaypb.OrderStatus_ORDER_STATUS_COMPLETED || client.request.GetCursor() != "current-subscriptions" || client.request.GetPageSize() != 25 || client.request.GetSortOrder() != gpaypb.SortOrder(1) {
		t.Fatalf("subscription request = %+v", client.request)
	}
	var payload struct {
		Data struct {
			Subscriptions []struct {
				OrderULID string `json:"order_ulid"`
			} `json:"subscriptions"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Subscriptions) != 1 || payload.Data.Subscriptions[0].OrderULID != "order-regression" || !payload.Data.HasMore || payload.Data.NextCursor != "next-subscriptions" {
		t.Fatalf("subscription page = %+v", payload.Data)
	}
}

func TestCheckCandidateQualificationForwardsReadTarget(t *testing.T) {
	client := &adminReadCredentialClient{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/permissions/check?candidate_ulid=candidate-regression&cred_def_ulid=definition-regression", nil)

	(&Handler{Creds: client}).CheckCandidateQualification(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request.GetCandidateUlid() != "candidate-regression" || client.request.GetCredDefUlid() != "definition-regression" {
		t.Fatalf("qualification request = %+v", client.request)
	}
	var payload struct {
		Data struct {
			Eligible bool   `json:"eligible"`
			Message  string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !payload.Data.Eligible || payload.Data.Message != "qualification active" {
		t.Fatalf("qualification response = %+v", payload.Data)
	}
}

func TestCheckCandidateQualificationRejectsMissingReadTarget(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/permissions/check?candidate_ulid=candidate-regression", nil)

	(&Handler{}).CheckCandidateQualification(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
}

func TestGetAdminMeReturnsReadOnlyProfile(t *testing.T) {
	original := getAdminProfile
	getAdminProfile = func(name string) (*casdoorsdk.User, error) {
		if name != "regression-admin" {
			t.Fatalf("profile name = %q", name)
		}
		return &casdoorsdk.User{
			Name:        name,
			Email:       "regression-admin@example.test",
			DisplayName: "Regression Administrator",
			Affiliation: "CFTP",
			Title:       "Operations",
			RealName:    "Admin Reader",
			Bio:         "Read-only regression profile",
			Gender:      "female",
			Birthday:    "1990-08-11",
			Education:   "Regression University",
		}, nil
	}
	t.Cleanup(func() { getAdminProfile = original })

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/user/me", nil)
	request = request.WithContext(WithCandidate(request.Context(), "admin-id", "regression-admin@example.test", "regression-admin", "token"))
	(&Handler{}).GetAdminMe(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload struct {
		Data UserMeRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Name != "regression-admin" || payload.Data.Email != "regression-admin@example.test" || payload.Data.DisplayName != "Regression Administrator" || payload.Data.Education != "Regression University" {
		t.Fatalf("profile response = %+v", payload.Data)
	}
}

func TestOpsDashboardReturnsFilteredReadOnlyOverview(t *testing.T) {
	originalRole := getDashboardRole
	roleConfig := dashboardRoleConfigFromEnv()
	getDashboardRole = func(string) (*casdoorsdk.Role, error) { return nil, nil }
	t.Cleanup(func() {
		getDashboardRole = originalRole
	})

	prog := &adminReadProgClient{}
	pay := &adminReadPayClient{}
	profiles := NewCandidateProfileCache(&adminReadMidClient{})
	profiles.ready = true
	profiles.users = []*casdoorsdk.User{{
		Id:            "user-regression",
		Name:          "regression-admin",
		DisplayName:   "Regression Administrator",
		Email:         "regression-admin@example.test",
		EmailVerified: true,
		Location:      "Shanghai",
		Roles:         []*casdoorsdk.Role{{Name: roleConfig.admin}},
	}}
	profiles.ulidsByUUID["user-regression"] = "candidate-user-regression"
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/dashboard/ops?user_keyword=regression&user_role=admin&user_status=active&user_page=1&user_page_size=5", nil)
	(&Handler{Gprog: prog, Gpay: pay, CandidateProfiles: profiles}).OpsDashboard(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if prog.request.GetPageSize() != 100 || pay.orderRequest.GetPageSize() != 100 || pay.orderRequest.GetFilters().GetStatusFilter() != gpaypb.OrderStatus_ORDER_STATUS_COMPLETED {
		t.Fatalf("dashboard downstream requests = %+v / %+v", prog.request, pay.orderRequest)
	}
	var payload struct {
		Data struct {
			CandidateTotal         int  `json:"candidate_total"`
			UserTotal              int  `json:"user_total"`
			UserPageSize           int  `json:"user_page_size"`
			StageBucketsExact      bool `json:"stage_buckets_exact"`
			TodayRevenueExact      bool `json:"today_revenue_exact"`
			AggregationSampleLimit int  `json:"aggregation_sample_limit"`
			Users                  []struct {
				Name          string `json:"name"`
				CandidateULID string `json:"candidate_ulid"`
			} `json:"users"`
			StageBuckets []struct {
				StageID string `json:"stage_id"`
				Count   int    `json:"count"`
			} `json:"stage_buckets"`
			TodayRevenue []struct {
				Currency    string `json:"currency"`
				AmountMinor int64  `json:"amount_minor"`
			} `json:"today_revenue"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.UserTotal != 1 || payload.Data.UserPageSize != 5 || len(payload.Data.Users) != 1 || payload.Data.Users[0].Name != "Regression Administrator" || payload.Data.Users[0].CandidateULID != "candidate-user-regression" {
		t.Fatalf("dashboard users = %+v", payload.Data)
	}
	if len(payload.Data.StageBuckets) != 1 || payload.Data.StageBuckets[0].StageID != "stage-regression" || payload.Data.StageBuckets[0].Count != 1 {
		t.Fatalf("dashboard stages = %+v", payload.Data.StageBuckets)
	}
	if len(payload.Data.TodayRevenue) != 1 || payload.Data.TodayRevenue[0].Currency != "USD" || payload.Data.TodayRevenue[0].AmountMinor != 12900 {
		t.Fatalf("dashboard revenue = %+v", payload.Data.TodayRevenue)
	}
	if !payload.Data.StageBucketsExact || !payload.Data.TodayRevenueExact || payload.Data.AggregationSampleLimit != adminDashboardSampleLimit {
		t.Fatalf("dashboard exactness metadata = %+v", payload.Data)
	}
}
