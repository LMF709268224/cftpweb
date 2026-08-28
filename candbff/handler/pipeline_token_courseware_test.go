package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc"
)

type lessonAccessURLClientStub struct {
	lmspb.LmsServiceClient
	request *lmspb.GetLessonAccessUrlRequest
}

func (s *lessonAccessURLClientStub) GetLessonAccessUrl(
	_ context.Context,
	request *lmspb.GetLessonAccessUrlRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetLessonAccessUrlResponse, error) {
	s.request = request
	return &lmspb.GetLessonAccessUrlResponse{
		LessonUlid: request.GetLessonUlid(),
		AccessUrl:  "https://courseware.example/learn/candidate-token",
	}, nil
}

func TestGetLessonAccessURLUsesAuthenticatedCandidate(t *testing.T) {
	client := &lessonAccessURLClientStub{}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/pipeline/lessons/lesson-1/access-url",
		"",
		"candidate-from-session",
		map[string]string{"lessonId": "lesson-1"},
	)

	(&Handler{Lms: client}).GetLessonAccessURL(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request.GetCandidateUlid() != "candidate-from-session" || client.request.GetLessonUlid() != "lesson-1" {
		t.Fatalf("access token request = %#v", client.request)
	}
}
