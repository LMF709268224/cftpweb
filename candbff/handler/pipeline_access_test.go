package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type pipelineAccessProgClientStub struct {
	gprogpb.ProgServiceClient
}

func (s *pipelineAccessProgClientStub) ListCandidatePipelines(
	_ context.Context,
	_ *gprogpb.ListCandidatePipelinesReq,
	_ ...grpc.CallOption,
) (*gprogpb.ListCandidatePipelinesRsp, error) {
	return &gprogpb.ListCandidatePipelinesRsp{
		Pipelines: []*gprogpb.PipelineSummary{{
			PipelineCcUlid: "pipeline-config-1",
		}},
	}, nil
}

type pipelineAccessCCClientStub struct {
	gccpb.CCServiceClient
	err error
}

func (s *pipelineAccessCCClientStub) GetPipeline(
	_ context.Context,
	_ *gccpb.GetPipelineRequest,
	_ ...grpc.CallOption,
) (*gccpb.PipelineConfig, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &gccpb.PipelineConfig{
		PipelineUlid: "pipeline-config-1",
		Stages: []*gccpb.StageConfig{{
			Units: []*gccpb.UnitConfig{{
				GlmsCourseUlid: "course-1",
			}},
		}},
	}, nil
}

type pipelineAccessLMSClientStub struct {
	lmspb.LmsServiceClient
	materialsErr error
}

func (s *pipelineAccessLMSClientStub) GetCourseSummary(
	_ context.Context,
	_ *lmspb.GetCourseSummaryCandidateRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetCourseSummaryResponse, error) {
	return &lmspb.GetCourseSummaryResponse{}, nil
}

func (s *pipelineAccessLMSClientStub) ListCourseMaterials(
	_ context.Context,
	_ *lmspb.ListCourseMaterialsCandidateRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListCourseMaterialsResponse, error) {
	if s.materialsErr != nil {
		return nil, s.materialsErr
	}
	return &lmspb.ListCourseMaterialsResponse{}, nil
}

func TestCandidateCourseIDsPropagatesPipelineConfigError(t *testing.T) {
	wantErr := status.Error(codes.Unavailable, "gcc unavailable")
	h := &Handler{
		Gprog: &pipelineAccessProgClientStub{},
		Gcc:   &pipelineAccessCCClientStub{err: wantErr},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/materials", nil)

	_, err := h.candidateCourseIDs(req, "candidate-1")

	if status.Code(err) != codes.Unavailable {
		t.Fatalf("candidateCourseIDs() error = %v, want Unavailable", err)
	}
}

func TestListMaterialsPropagatesCourseMaterialError(t *testing.T) {
	h := &Handler{
		Gprog: &pipelineAccessProgClientStub{},
		Gcc:   &pipelineAccessCCClientStub{},
		Lms: &pipelineAccessLMSClientStub{
			materialsErr: status.Error(codes.Unavailable, "lms unavailable"),
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/materials", nil)
	req = req.WithContext(WithCandidate(req.Context(), "candidate-1", "", "", ""))
	rec := httptest.NewRecorder()

	h.ListMaterials(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}
