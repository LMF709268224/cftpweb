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
	err                 error
	finalEligibilityErr error
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

func (s *pipelineAccessCCClientStub) ListPipelines(
	_ context.Context,
	_ *gccpb.ListPipelinesRequest,
	_ ...grpc.CallOption,
) (*gccpb.ListPipelinesResponse, error) {
	return &gccpb.ListPipelinesResponse{
		Pipelines: []*gccpb.PipelineSummary{{
			PipelineUlid: "pipeline-config-1",
		}},
	}, nil
}

func (s *pipelineAccessCCClientStub) GetPipelineFinalEligibility(
	_ context.Context,
	_ *gccpb.GetPipelineFinalEligibilityRequest,
	_ ...grpc.CallOption,
) (*gccpb.GetPipelineFinalEligibilityResponse, error) {
	if s.finalEligibilityErr != nil {
		return nil, s.finalEligibilityErr
	}
	return &gccpb.GetPipelineFinalEligibilityResponse{}, nil
}

type pipelineAccessLMSClientStub struct {
	lmspb.LmsServiceClient
	materialsErr        error
	enrollmentDetailErr error
	quizAttemptsErr     error
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

func (s *pipelineAccessLMSClientStub) ListCandidateEnrollments(
	_ context.Context,
	_ *lmspb.ListCandidateEnrollmentsRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListCandidateEnrollmentsResponse, error) {
	return &lmspb.ListCandidateEnrollmentsResponse{
		Enrollments: []*lmspb.CandidateEnrollmentSummary{{
			EnrollmentId: "enrollment-1",
		}},
	}, nil
}

func (s *pipelineAccessLMSClientStub) GetCandidateEnrollmentDetail(
	_ context.Context,
	_ *lmspb.GetCandidateEnrollmentDetailRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetCandidateEnrollmentDetailResponse, error) {
	if s.enrollmentDetailErr != nil {
		return nil, s.enrollmentDetailErr
	}
	return &lmspb.GetCandidateEnrollmentDetailResponse{}, nil
}

func (s *pipelineAccessLMSClientStub) ListQuizAttemptsCandidate(
	_ context.Context,
	_ *lmspb.ListQuizAttemptsCandidateRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListQuizAttemptsResponse, error) {
	if s.quizAttemptsErr != nil {
		return nil, s.quizAttemptsErr
	}
	return &lmspb.ListQuizAttemptsResponse{}, nil
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

func TestListPipelinesPropagatesFinalEligibilityError(t *testing.T) {
	h := &Handler{
		Gcc: &pipelineAccessCCClientStub{
			finalEligibilityErr: status.Error(codes.Unavailable, "gcc eligibility unavailable"),
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/mall/pipelines", nil)
	req = req.WithContext(WithCandidate(req.Context(), "candidate-1", "", "", ""))
	rec := httptest.NewRecorder()

	h.ListPipelines(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestGetProgressPropagatesEnrollmentDetailError(t *testing.T) {
	h := &Handler{
		Lms: &pipelineAccessLMSClientStub{
			enrollmentDetailErr: status.Error(codes.Unavailable, "enrollment detail unavailable"),
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/pipeline/progress", nil)
	req = req.WithContext(WithCandidate(req.Context(), "candidate-1", "", "", ""))
	rec := httptest.NewRecorder()

	h.GetProgress(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestQuizProgressByCoursePropagatesAttemptError(t *testing.T) {
	h := &Handler{
		Lms: &pipelineAccessLMSClientStub{
			quizAttemptsErr: status.Error(codes.Unavailable, "quiz attempts unavailable"),
		},
	}
	course := &lmspb.CompleteCourse{
		Quizzes: []*lmspb.QuizDetail{{
			Quiz: &lmspb.Quiz{QuizUlid: "quiz-1"},
		}},
	}

	_, err := h.quizProgressByCourse(context.Background(), "candidate-1", course)

	if status.Code(err) != codes.Unavailable {
		t.Fatalf("quizProgressByCourse() error = %v, want Unavailable", err)
	}
}
