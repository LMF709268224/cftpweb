package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type lessonVideoClientStub struct {
	lmspb.LmsServiceClient
	request  *lmspb.GetLessonVideoPlayURLRequest
	response *lmspb.GetLessonVideoPlayURLResponse
	err      error
}

func (s *lessonVideoClientStub) GetLessonVideoPlayURL(
	_ context.Context,
	request *lmspb.GetLessonVideoPlayURLRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetLessonVideoPlayURLResponse, error) {
	s.request = request
	if s.err != nil {
		return nil, s.err
	}
	if s.response != nil {
		return s.response, nil
	}
	return &lmspb.GetLessonVideoPlayURLResponse{}, nil
}

func TestGetLessonVideoPlayURLForwardsCandidateAndLesson(t *testing.T) {
	client := &lessonVideoClientStub{
		response: &lmspb.GetLessonVideoPlayURLResponse{
			PlayUrl:   "https://iframe.videodelivery.net/signed-token",
			ExpiresAt: "2026-08-17T12:00:00Z",
		},
	}
	handler := &Handler{Lms: client}
	recorder := httptest.NewRecorder()

	handler.GetLessonVideoPlayURL(
		recorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/pipeline/lessons/lesson-1/video-play-url",
			"",
			"candidate-1",
			map[string]string{"lessonId": "lesson-1"},
		),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request.GetCandidateUlid() != "candidate-1" || client.request.GetLessonUlid() != "lesson-1" {
		t.Fatalf("video play request = %#v", client.request)
	}
	var response struct {
		Data GetAccessURLRsp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if response.Data.URL != client.response.GetPlayUrl() || response.Data.ExpiresAt != client.response.GetExpiresAt() {
		t.Fatalf("response = %#v", response)
	}
}

func TestGetLessonVideoPlayURLPropagatesPermissionError(t *testing.T) {
	handler := &Handler{Lms: &lessonVideoClientStub{err: status.Error(codes.PermissionDenied, "lesson is not available")}}
	recorder := httptest.NewRecorder()

	handler.GetLessonVideoPlayURL(
		recorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/pipeline/lessons/lesson-1/video-play-url",
			"",
			"candidate-1",
			map[string]string{"lessonId": "lesson-1"},
		),
	)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusForbidden, recorder.Body.String())
	}
}

func TestGetLessonVideoPlayURLRejectsEmptyServiceURL(t *testing.T) {
	handler := &Handler{Lms: &lessonVideoClientStub{response: &lmspb.GetLessonVideoPlayURLResponse{}}}
	recorder := httptest.NewRecorder()

	handler.GetLessonVideoPlayURL(
		recorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/pipeline/lessons/lesson-1/video-play-url",
			"",
			"candidate-1",
			map[string]string{"lessonId": "lesson-1"},
		),
	)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusBadGateway, recorder.Body.String())
	}
}
