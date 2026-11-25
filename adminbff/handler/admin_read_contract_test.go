package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type membershipContractStub struct {
	gmbrpb.GmbrServiceClient
	response *gmbrpb.GetActiveMembershipResponse
	err      error
}

func (s *membershipContractStub) GetActiveMembership(context.Context, *gmbrpb.GetActiveMembershipRequest, ...grpc.CallOption) (*gmbrpb.GetActiveMembershipResponse, error) {
	return s.response, s.err
}

func contractData(t *testing.T, recorder *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected HTTP response: %d %s", recorder.Code, recorder.Body)
	}
	var result struct{ Data map[string]interface{} }
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	return result.Data
}

func TestActiveMembershipDistinguishesNoRecordFromFailure(t *testing.T) {
	for _, tc := range []struct {
		name       string
		response   *gmbrpb.GetActiveMembershipResponse
		err        error
		wantStatus int
	}{
		{"no record", &gmbrpb.GetActiveMembershipResponse{}, nil, http.StatusOK},
		{"missing response", nil, nil, http.StatusBadGateway},
		{"service failure", nil, status.Error(codes.Unavailable, "offline"), http.StatusServiceUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			h := &Handler{Gmbr: &membershipContractStub{response: tc.response, err: tc.err}}
			h.GetActiveMembership(r, httptest.NewRequest(http.MethodGet, "/api/memberships/active?candidate_ulid=candidate-1", nil))
			if r.Code != tc.wantStatus {
				t.Fatalf("status=%d, body=%s", r.Code, r.Body)
			}
			if r.Code == http.StatusOK {
				data := contractData(t, r)
				value, present := data["membership"]
				if !present || value != nil {
					t.Fatalf("expected explicit null membership: %v", data)
				}
			}
		})
	}
}

type lmsReadContractStub struct {
	lmspb.LmsServiceClient
	enrollments *lmspb.ListCourseEnrollmentsResponse
	course      *lmspb.GetCompleteCourseResponse
}

func (s *lmsReadContractStub) ListCourseEnrollmentsAdmin(context.Context, *lmspb.ListCourseEnrollmentsRequest, ...grpc.CallOption) (*lmspb.ListCourseEnrollmentsResponse, error) {
	return s.enrollments, nil
}
func (s *lmsReadContractStub) GetCompleteCourseAdmin(context.Context, *lmspb.GetCompleteCourseRequest, ...grpc.CallOption) (*lmspb.GetCompleteCourseResponse, error) {
	return s.course, nil
}

func TestEnrollmentHTTPPreservesEmptyPageAndActualZeroProgress(t *testing.T) {
	client := &lmsReadContractStub{enrollments: &lmspb.ListCourseEnrollmentsResponse{}}
	h := &Handler{Lms: client}
	read := func() *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		h.ListLmsCourseEnrollments(r, httptest.NewRequest(http.MethodGet, "/api/lms/enrollments?candidate_id=candidate-1", nil))
		return r
	}
	data := contractData(t, read())
	if items, ok := data["enrollments"].([]interface{}); !ok || len(items) != 0 {
		t.Fatalf("empty list missing: %v", data)
	}
	if data["has_more"] != false || data["next_cursor"] != "" {
		t.Fatalf("last-page fields missing: %v", data)
	}
	client.enrollments = &lmspb.ListCourseEnrollmentsResponse{Enrollments: []*lmspb.CourseEnrollment{{EnrollmentId: "e1", ProgressPercentage: 0, Status: "learning"}}, HasMore: true, NextCursor: "cursor-2"}
	data = contractData(t, read())
	row := data["enrollments"].([]interface{})[0].(map[string]interface{})
	if row["progress_percentage"] != float64(0) || row["enrollment_id"] != "e1" || data["has_more"] != true || data["next_cursor"] != "cursor-2" {
		t.Fatalf("actual values lost: %v", data)
	}
	client.enrollments = nil
	if r := read(); r.Code != http.StatusBadGateway {
		t.Fatalf("nil response became success: %s", r.Body)
	}
	client.enrollments = &lmspb.ListCourseEnrollmentsResponse{Enrollments: []*lmspb.CourseEnrollment{nil}}
	if r := read(); r.Code != http.StatusBadGateway {
		t.Fatalf("nil entity became success: %s", r.Body)
	}
}

func TestCompleteCourseHTTPPreservesEmptyOutlineWithoutInventingEntities(t *testing.T) {
	client := &lmsReadContractStub{course: &lmspb.GetCompleteCourseResponse{CompleteCourse: &lmspb.CompleteCourse{Course: &lmspb.Course{CourseUlid: "course-1"}}}}
	h := &Handler{Lms: client}
	read := func() *httptest.ResponseRecorder {
		r := httptest.NewRecorder()
		h.GetCompleteLmsCourse(r, requestWithURLParam(http.MethodGet, "/api/lms/courses/course-1/complete", "course_id", "course-1"))
		return r
	}
	data := contractData(t, read())["complete_course"].(map[string]interface{})
	if chapters, ok := data["chapters"].([]interface{}); !ok || len(chapters) != 0 {
		t.Fatalf("empty chapters missing: %v", data)
	}
	client.course.CompleteCourse.Chapters = []*lmspb.ChapterDetail{{Chapter: &lmspb.Chapter{ChapterUlid: "chapter-1"}}}
	data = contractData(t, read())["complete_course"].(map[string]interface{})
	chapter := data["chapters"].([]interface{})[0].(map[string]interface{})
	if lessons, ok := chapter["lessons"].([]interface{}); !ok || len(lessons) != 0 {
		t.Fatalf("empty lessons missing: %v", chapter)
	}
	client.course.CompleteCourse.Chapters[0].Lessons = []*lmspb.LessonDetail{{Lesson: &lmspb.Lesson{LessonUlid: "lesson-1", Body: "", MediaObjectKey: "private.pdf"}}}
	data = contractData(t, read())["complete_course"].(map[string]interface{})
	chapter = data["chapters"].([]interface{})[0].(map[string]interface{})
	lesson := chapter["lessons"].([]interface{})[0].(map[string]interface{})["lesson"].(map[string]interface{})
	if lesson["body"] != "" || lesson["media_object_key"] != "private.pdf" {
		t.Fatalf("actual lesson fields changed: %v", lesson)
	}
	client.course.CompleteCourse.Chapters[0].Lessons[0].Lesson = nil
	if r := read(); r.Code != http.StatusBadGateway {
		t.Fatalf("nil lesson became success: %s", r.Body)
	}
	client.course = &lmspb.GetCompleteCourseResponse{}
	if r := read(); r.Code != http.StatusBadGateway {
		t.Fatalf("missing tree became empty course: %s", r.Body)
	}
}
