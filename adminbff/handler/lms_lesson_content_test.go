package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type lessonContentClient struct {
	lmspb.LmsServiceClient
	lesson            *lmspb.Lesson
	update            *lmspb.UpdateLessonRequest
	readErr, writeErr error
}

func (s *lessonContentClient) GetLessonAdmin(context.Context, *lmspb.GetLessonRequest, ...grpc.CallOption) (*lmspb.GetLessonResponse, error) {
	return &lmspb.GetLessonResponse{Lesson: s.lesson}, s.readErr
}
func (s *lessonContentClient) UpdateLessonAdmin(_ context.Context, req *lmspb.UpdateLessonRequest, _ ...grpc.CallOption) (*lmspb.UpdateLessonResponse, error) {
	s.update = req
	return &lmspb.UpdateLessonResponse{Success: true}, s.writeErr
}
func contentRequest(body string) *http.Request {
	r := requestWithURLParam(http.MethodPatch, "/api/lms/lessons/lesson-1/content", "lesson_id", "lesson-1")
	r.Body = io.NopCloser(strings.NewReader(body))
	return r
}
func TestLessonContentPreservesAuthoritativeFields(t *testing.T) {
	for _, body := range []string{`{"title":"Corrected","body":"<p>New body</p>","version":7}`, `{"title":"Corrected","version":7}`} {
		client := &lessonContentClient{lesson: &lmspb.Lesson{
			LessonUlid: "lesson-1", ChapterUlid: "chapter-1", Title: "Before", Body: "Original body", Version: 7, SortOrder: 3,
			LessonType: lmspb.LessonType(1), MediaObjectKey: "private/video", MediaFileHash: "hash",
			ExternalUrl: "https://example.test", ExternalCoursewareUlid: "courseware-id", VideoProvider: "other", VideoStreamUid: "stream-id", VideoEmbedCode: "embed-data", MetaJson: `{"captions":"preserve"}`,
		}}
		w := httptest.NewRecorder()
		(&Handler{Lms: client}).UpdateLmsLessonContent(w, contentRequest(body))
		if w.Code != 200 || client.update == nil {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		got, original := client.update, client.lesson
		if got.Title != "Corrected" || got.Version != 7 || got.SortOrder != original.SortOrder || got.LessonType != original.LessonType || got.MediaObjectKey != original.MediaObjectKey || got.MediaFileHash != original.MediaFileHash || got.VideoProvider != original.VideoProvider || got.VideoStreamUid != original.VideoStreamUid || got.VideoEmbedCode != original.VideoEmbedCode || got.MetaJson != original.MetaJson || got.ExternalUrl != original.ExternalUrl || got.ExternalCoursewareUlid != original.ExternalCoursewareUlid {
			t.Fatalf("unrelated fields changed: %+v", got)
		}
		if strings.Contains(body, `"body"`) && got.Body != "<p>New body</p>" || !strings.Contains(body, `"body"`) && got.Body != original.Body {
			t.Fatalf("body=%s", got.Body)
		}
	}
}
func TestLessonContentDoesNotGuessMissingDataOrRetryConflicts(t *testing.T) {
	for _, tc := range []struct {
		name              string
		lesson            *lmspb.Lesson
		readErr, writeErr error
		want              int
		writes            bool
	}{
		{"missing lesson", nil, nil, nil, 502, false},
		{"wrong lesson", &lmspb.Lesson{LessonUlid: "other", Version: 7}, nil, nil, 502, false},
		{"stale version", &lmspb.Lesson{LessonUlid: "lesson-1", Version: 8}, nil, nil, 409, false},
		{"read failure", nil, status.Error(codes.Unavailable, "unavailable"), nil, 503, false},
		{"write conflict", &lmspb.Lesson{LessonUlid: "lesson-1", Version: 7}, nil, status.Error(codes.Aborted, "conflict"), 409, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &lessonContentClient{lesson: tc.lesson, readErr: tc.readErr, writeErr: tc.writeErr}
			w := httptest.NewRecorder()
			(&Handler{Lms: client}).UpdateLmsLessonContent(w, contentRequest(`{"title":"Fix","body":"Text","version":7}`))
			if w.Code != tc.want || (client.update != nil) != tc.writes {
				t.Fatalf("status=%d update=%+v", w.Code, client.update)
			}
		})
	}
}
