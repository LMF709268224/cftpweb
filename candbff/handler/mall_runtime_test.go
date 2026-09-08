package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mallRuntimeCCClientStub struct {
	gccpb.CCServiceClient
	pipeline *gccpb.PipelineConfig
}

func (s *mallRuntimeCCClientStub) GetPipeline(
	_ context.Context,
	_ *gccpb.GetPipelineRequest,
	_ ...grpc.CallOption,
) (*gccpb.PipelineConfig, error) {
	if s.pipeline != nil {
		return s.pipeline, nil
	}
	return &gccpb.PipelineConfig{PipelineUlid: "pipeline-config-1"}, nil
}

type mallRuntimeProgClientStub struct {
	gprogpb.ProgServiceClient
	listResp   *gprogpb.ListCandidatePipelinesRsp
	listErr    error
	detailResp *gprogpb.GetPipelineDetailRsp
	detailErr  error
}

type mallRuntimeLMSClientStub struct {
	lmspb.LmsServiceClient
	enrollments []*lmspb.CandidateEnrollmentSummary
}

func (s *mallRuntimeLMSClientStub) ListCandidateEnrollments(
	_ context.Context,
	_ *lmspb.ListCandidateEnrollmentsRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListCandidateEnrollmentsResponse, error) {
	return &lmspb.ListCandidateEnrollmentsResponse{Enrollments: s.enrollments}, nil
}

type mallRuntimeMallClientStub struct {
	mallpb.MallServiceClient
	listPipelineOrdersRequests []*mallpb.ListPipelineOrdersRequest
	pipelineOrderDetailRequest *mallpb.GetPipelineOrderDetailRequest
	bundleOrderDetailRequest   *mallpb.GetBundleOrderDetailRequest
}

func (s *mallRuntimeMallClientStub) ListPipelineOrders(
	_ context.Context,
	req *mallpb.ListPipelineOrdersRequest,
	_ ...grpc.CallOption,
) (*mallpb.ListPipelineOrdersResponse, error) {
	s.listPipelineOrdersRequests = append(s.listPipelineOrdersRequests, req)
	return &mallpb.ListPipelineOrdersResponse{
		Items: []*mallpb.PipelineOrderSummary{{
			PipelineOrderUlid: "pipeline-order-1",
			CandidateUlid:     "candidate-1",
			PipelineCcUlid:    "pipeline-config-1",
			OrderStatus:       "COMPLETED",
			BundleOrderUlid:   "bundle-order-1",
		}},
	}, nil
}

func (s *mallRuntimeMallClientStub) GetPipelineOrderDetail(
	_ context.Context,
	req *mallpb.GetPipelineOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetPipelineOrderDetailResponse, error) {
	s.pipelineOrderDetailRequest = req
	return &mallpb.GetPipelineOrderDetailResponse{
		Found: true,
		Detail: &mallpb.PipelineOrderDetail{
			Summary: &mallpb.PipelineOrderSummary{
				PipelineOrderUlid: "pipeline-order-1",
				CandidateUlid:     "candidate-1",
				PipelineCcUlid:    "pipeline-config-1",
				OrderStatus:       "COMPLETED",
				BundleOrderUlid:   "bundle-order-1",
			},
			InstantiatedPipelineUlid: "pipeline-instance-1",
		},
	}, nil
}

func (s *mallRuntimeMallClientStub) GetBundleOrderDetail(
	_ context.Context,
	req *mallpb.GetBundleOrderDetailRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetBundleOrderDetailResponse, error) {
	s.bundleOrderDetailRequest = req
	return &mallpb.GetBundleOrderDetailResponse{
		Found: true,
		Detail: &mallpb.BundleOrderDetail{
			Summary: &mallpb.BundleOrderSummary{
				BundleOrderUlid: "bundle-order-1",
				CandidateUlid:   "candidate-1",
				BundleUlid:      "retired-bundle-1",
				OrderStatus:     "COMPLETED",
			},
		},
	}, nil
}

func (s *mallRuntimeProgClientStub) ListCandidatePipelines(
	_ context.Context,
	_ *gprogpb.ListCandidatePipelinesReq,
	_ ...grpc.CallOption,
) (*gprogpb.ListCandidatePipelinesRsp, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp != nil {
		return s.listResp, nil
	}
	return &gprogpb.ListCandidatePipelinesRsp{}, nil
}

func (s *mallRuntimeProgClientStub) GetPipelineDetail(
	_ context.Context,
	_ *gprogpb.GetPipelineDetailReq,
	_ ...grpc.CallOption,
) (*gprogpb.GetPipelineDetailRsp, error) {
	if s.detailErr != nil {
		return nil, s.detailErr
	}
	if s.detailResp != nil {
		return s.detailResp, nil
	}
	return &gprogpb.GetPipelineDetailRsp{}, nil
}

func mallPipelineRequest() *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/api/mall/pipelines/pipeline-config-1/runtime", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("pipelineId", "pipeline-config-1")
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
	return req.WithContext(WithCandidate(ctx, "candidate-1", "", "", ""))
}

func TestPipelineDetailHandlersPropagateCandidatePipelineLookupErrors(t *testing.T) {
	tests := []struct {
		name    string
		handler func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{name: "detail", handler: (*Handler).GetPipelineDetail},
		{name: "runtime", handler: (*Handler).GetPipelineRuntime},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handler{
				Gcc: &mallRuntimeCCClientStub{},
				Gprog: &mallRuntimeProgClientStub{
					listErr: status.Error(codes.Unavailable, "gprog unavailable"),
				},
			}
			rec := httptest.NewRecorder()

			tt.handler(h, rec, mallPipelineRequest())

			if rec.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
			}
		})
	}
}

func TestGetPipelineRuntimePropagatesRuntimeLookupError(t *testing.T) {
	h := &Handler{
		Gcc: &mallRuntimeCCClientStub{},
		Gprog: &mallRuntimeProgClientStub{
			listResp: &gprogpb.ListCandidatePipelinesRsp{
				Pipelines: []*gprogpb.PipelineSummary{{
					PipelineUlid:   "pipeline-instance-1",
					PipelineCcUlid: "pipeline-config-1",
				}},
			},
			detailErr: status.Error(codes.Unavailable, "runtime unavailable"),
		},
	}
	rec := httptest.NewRecorder()

	h.GetPipelineRuntime(rec, mallPipelineRequest())

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestGetPipelineRuntimeKeepsNotPurchasedStateWhenLookupSucceeds(t *testing.T) {
	h := &Handler{
		Gcc:   &mallRuntimeCCClientStub{},
		Gprog: &mallRuntimeProgClientStub{},
	}
	rec := httptest.NewRecorder()

	h.GetPipelineRuntime(rec, mallPipelineRequest())

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestGetPipelineRuntimeReturnsSourceBundleFromExactPipelineInstance(t *testing.T) {
	mallClient := &mallRuntimeMallClientStub{}
	h := &Handler{
		Gcc: &mallRuntimeCCClientStub{},
		Gprog: &mallRuntimeProgClientStub{
			listResp: &gprogpb.ListCandidatePipelinesRsp{
				Pipelines: []*gprogpb.PipelineSummary{{
					PipelineUlid:   "pipeline-instance-1",
					PipelineCcUlid: "pipeline-config-1",
				}},
			},
		},
		Mall: mallClient,
	}
	recorder := httptest.NewRecorder()

	h.GetPipelineRuntime(recorder, mallPipelineRequest())

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		Data PipelineRuntimeRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	if response.Data.BundleUlid != "retired-bundle-1" {
		t.Fatalf("bundle_ulid = %q, want retired-bundle-1", response.Data.BundleUlid)
	}
	if len(mallClient.listPipelineOrdersRequests) != 1 {
		t.Fatalf("ListPipelineOrders requests = %d, want 1", len(mallClient.listPipelineOrdersRequests))
	}
	filters := mallClient.listPipelineOrdersRequests[0].GetFilters()
	if filters.GetCandidateUlid() != "candidate-1" ||
		filters.GetPipelineCcUlid() != "pipeline-config-1" ||
		filters.GetOrderStatus() != "COMPLETED" ||
		filters.GetPaymentMode() != "" {
		t.Fatalf("ListPipelineOrders filters = %+v, want exact candidate/pipeline completed orders", filters)
	}
	if mallClient.pipelineOrderDetailRequest.GetPipelineOrderUlid() != "pipeline-order-1" {
		t.Fatalf("pipeline order detail id = %q, want pipeline-order-1", mallClient.pipelineOrderDetailRequest.GetPipelineOrderUlid())
	}
	if mallClient.bundleOrderDetailRequest.GetBundleOrderUlid() != "bundle-order-1" {
		t.Fatalf("bundle order detail id = %q, want bundle-order-1", mallClient.bundleOrderDetailRequest.GetBundleOrderUlid())
	}
}

func TestGetPipelineRuntimeIncludesLearningAccessForCompletedEnrollment(t *testing.T) {
	h := &Handler{
		Gcc: &mallRuntimeCCClientStub{pipeline: &gccpb.PipelineConfig{
			PipelineUlid: "pipeline-config-1",
			Stages: []*gccpb.StageConfig{{
				StageUlid: "stage-config-1",
				Units: []*gccpb.UnitConfig{
					{UnitUlid: "unit-config-completed", GlmsCourseUlid: "course-completed"},
					{UnitUlid: "unit-config-unavailable", GlmsCourseUlid: "course-unavailable"},
				},
			}},
		}},
		Gprog: &mallRuntimeProgClientStub{
			listResp: &gprogpb.ListCandidatePipelinesRsp{Pipelines: []*gprogpb.PipelineSummary{{
				PipelineUlid:   "pipeline-instance-1",
				PipelineCcUlid: "pipeline-config-1",
			}}},
			detailResp: &gprogpb.GetPipelineDetailRsp{Stages: []*gprogpb.StageDetail{{
				Stage: &gprogpb.StageSummary{
					StageCcUlid: "stage-config-1",
					Status:      gprogpb.StageStatus_STAGE_STATUS_COMPLETED,
				},
			}}},
		},
		Lms: &mallRuntimeLMSClientStub{enrollments: []*lmspb.CandidateEnrollmentSummary{{
			EnrollmentId: "enrollment-completed",
			CourseUlid:   "course-version-completed",
			CourseId:     "course-completed",
			Status:       "completed",
		}}},
	}
	recorder := httptest.NewRecorder()

	h.GetPipelineRuntime(recorder, mallPipelineRequest())

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var response struct {
		Data PipelineRuntimeRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	completed := response.Data.Config.Stages[0].Units[0]
	if !completed.HasLearningAccess || completed.EnrollmentStatus != "completed" {
		t.Fatalf("completed unit access = %+v, want completed enrollment access", completed)
	}
	unavailable := response.Data.Config.Stages[0].Units[1]
	if unavailable.HasLearningAccess || unavailable.EnrollmentStatus != "" {
		t.Fatalf("unavailable unit access = %+v, want no learning access", unavailable)
	}
}

func TestMergeRuntimeStatusesIncludesStagePaymentState(t *testing.T) {
	config := &PipelineConfig{
		Stages: []StageConfig{
			{StageUlid: "stage-config-1"},
			{StageUlid: "stage-config-2"},
		},
	}
	runtime := &gprogpb.GetPipelineDetailRsp{
		Pipeline: &gprogpb.PipelineDetail{
			CourseSelectionJson: `{"stages":[{"stage_cc_ulid":"stage-config-1","is_paid":true},{"stage_cc_ulid":"stage-config-2","is_paid":true}]}`,
		},
		Stages: []*gprogpb.StageDetail{{
			Stage: &gprogpb.StageSummary{
				StageCcUlid: "stage-config-1",
				Status:      gprogpb.StageStatus_STAGE_STATUS_RUNNING,
			},
		}},
	}

	mergeRuntimeStatuses(config, runtime)

	if config.Stages[0].IsPaid == nil || !*config.Stages[0].IsPaid {
		t.Fatalf("first stage is_paid = %v, want true", config.Stages[0].IsPaid)
	}
	if config.Stages[1].IsPaid == nil || !*config.Stages[1].IsPaid {
		t.Fatalf("future stage is_paid = %v, want true", config.Stages[1].IsPaid)
	}
	if config.Stages[0].RuntimeStatus != "STAGE_STATUS_RUNNING" {
		t.Fatalf("first stage runtime_status = %q, want STAGE_STATUS_RUNNING", config.Stages[0].RuntimeStatus)
	}
	if config.Stages[1].RuntimeStatus != "" {
		t.Fatalf("future stage runtime_status = %q, want empty before instantiation", config.Stages[1].RuntimeStatus)
	}
}

func TestMergeRuntimeStatusesPreservesUnpaidFutureStage(t *testing.T) {
	config := &PipelineConfig{
		Stages: []StageConfig{{StageUlid: "stage-config-2"}},
	}
	runtime := &gprogpb.GetPipelineDetailRsp{
		Pipeline: &gprogpb.PipelineDetail{
			CourseSelectionJson: `{"stages":[{"stage_cc_ulid":"stage-config-2","is_paid":false}]}`,
		},
	}

	mergeRuntimeStatuses(config, runtime)

	if config.Stages[0].IsPaid == nil || *config.Stages[0].IsPaid {
		t.Fatalf("future stage is_paid = %v, want false", config.Stages[0].IsPaid)
	}
}
