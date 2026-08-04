package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mallRuntimeCCClientStub struct {
	gccpb.CCServiceClient
}

func (s *mallRuntimeCCClientStub) GetPipeline(
	_ context.Context,
	_ *gccpb.GetPipelineRequest,
	_ ...grpc.CallOption,
) (*gccpb.PipelineConfig, error) {
	return &gccpb.PipelineConfig{PipelineUlid: "pipeline-config-1"}, nil
}

type mallRuntimeProgClientStub struct {
	gprogpb.ProgServiceClient
	listResp  *gprogpb.ListCandidatePipelinesRsp
	listErr   error
	detailErr error
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
