package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc"
)

type lessonAccessTokenClientStub struct {
	lmspb.LmsServiceClient
	request *lmspb.GetLessonAccessTokenRequest
}

func (s *lessonAccessTokenClientStub) GetLessonAccessToken(
	_ context.Context,
	request *lmspb.GetLessonAccessTokenRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetLessonAccessTokenResponse, error) {
	s.request = request
	return &lmspb.GetLessonAccessTokenResponse{
		LessonUlid:     request.GetLessonUlid(),
		Token:          "candidate-token",
		BaseUrl:        "https://courseware.example/learn",
		TokenParamName: "auth_token",
	}, nil
}

func TestGetLessonAccessTokenUsesAuthenticatedCandidate(t *testing.T) {
	client := &lessonAccessTokenClientStub{}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/pipeline/lessons/lesson-1/access-token",
		"",
		"candidate-from-session",
		map[string]string{"lessonId": "lesson-1"},
	)

	(&Handler{Lms: client}).GetLessonAccessToken(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request.GetCandidateUlid() != "candidate-from-session" || client.request.GetLessonUlid() != "lesson-1" {
		t.Fatalf("access token request = %#v", client.request)
	}
}
