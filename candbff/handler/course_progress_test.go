package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type courseProgressLMSClientStub struct {
	lmspb.LmsServiceClient
	listResp *lmspb.ListCandidateEnrollmentsResponse
	listErr  error
}

func (s *courseProgressLMSClientStub) ListCandidateEnrollments(
	_ context.Context,
	_ *lmspb.ListCandidateEnrollmentsRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListCandidateEnrollmentsResponse, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	if s.listResp != nil {
		return s.listResp, nil
	}
	return &lmspb.ListCandidateEnrollmentsResponse{}, nil
}

func courseProgressRequest(courseID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/courses/"+courseID+"/sync-progress", nil)
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("courseId", courseID)
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, routeContext)
	return req.WithContext(WithCandidate(ctx, "candidate-1", "", "", ""))
}

func TestSyncCourseProgressPropagatesLMSError(t *testing.T) {
	h := &Handler{Lms: &courseProgressLMSClientStub{
		listErr: status.Error(codes.Unavailable, "lms unavailable"),
	}}
	rec := httptest.NewRecorder()

	h.SyncCourseProgress(rec, courseProgressRequest("course-1"))

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusServiceUnavailable, rec.Body.String())
	}
}

func TestSyncCourseProgressKeepsTemporaryStateWhenEnrollmentMissing(t *testing.T) {
	h := &Handler{Lms: &courseProgressLMSClientStub{}}
	rec := httptest.NewRecorder()

	h.SyncCourseProgress(rec, courseProgressRequest("course-1"))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var response struct {
		Data SyncCourseProgressRsp `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !response.Data.Success || response.Data.CourseStatus != "learning" || response.Data.ProgressPercentage != 0 {
		t.Fatalf("response = %+v, want temporary learning state", response.Data)
	}
}
