package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	lmspb "github.com/afnandelfin620-star/cftptest/cftp/glms"
	"google.golang.org/grpc"
)

type enrollmentLMSClientStub struct {
	lmspb.LmsServiceClient
	listRequest    *lmspb.ListCandidateEnrollmentsRequest
	listResponse   *lmspb.ListCandidateEnrollmentsResponse
	detailRequest  *lmspb.GetCandidateEnrollmentDetailRequest
	detailResponse *lmspb.GetCandidateEnrollmentDetailResponse
}

func (s *enrollmentLMSClientStub) ListCandidateEnrollments(
	_ context.Context,
	request *lmspb.ListCandidateEnrollmentsRequest,
	_ ...grpc.CallOption,
) (*lmspb.ListCandidateEnrollmentsResponse, error) {
	s.listRequest = request
	return s.listResponse, nil
}

func (s *enrollmentLMSClientStub) GetCandidateEnrollmentDetail(
	_ context.Context,
	request *lmspb.GetCandidateEnrollmentDetailRequest,
	_ ...grpc.CallOption,
) (*lmspb.GetCandidateEnrollmentDetailResponse, error) {
	s.detailRequest = request
	return s.detailResponse, nil
}

func TestListCandidateEnrollmentsForwardsCandidateFiltersAndPagination(t *testing.T) {
	lms := &enrollmentLMSClientStub{
		listResponse: &lmspb.ListCandidateEnrollmentsResponse{
			Enrollments: []*lmspb.CandidateEnrollmentSummary{{
				EnrollmentId:       "enrollment-1",
				CourseUlid:         "course-1",
				CourseTitle:        "Candidate Course",
				Status:             "learning",
				ProgressPercentage: 35,
			}},
			NextCursor: "next-enrollment",
			PrevCursor: "prev-enrollment",
			HasMore:    true,
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/enrollments?status=learning&pageSize=25&pageToken=current-page",
		"",
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{Lms: lms}).ListCandidateEnrollments(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if lms.listRequest == nil {
		t.Fatal("ListCandidateEnrollments was not called")
	}
	if lms.listRequest.GetFilters().GetCandidateUlid() != "candidate-1" ||
		lms.listRequest.GetFilters().GetStatus() != "learning" ||
		lms.listRequest.GetPageSize() != 25 ||
		lms.listRequest.GetCursor() != "current-page" {
		t.Fatalf("ListCandidateEnrollments request = %+v", lms.listRequest)
	}
	var response struct {
		Data lmspb.ListCandidateEnrollmentsResponse `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	if len(response.Data.GetEnrollments()) != 1 ||
		response.Data.GetEnrollments()[0].GetEnrollmentId() != "enrollment-1" ||
		response.Data.GetNextCursor() != "next-enrollment" ||
		response.Data.GetPrevCursor() != "prev-enrollment" ||
		!response.Data.GetHasMore() {
		t.Fatalf(
			"enrollment response = count:%d next:%q prev:%q has_more:%t",
			len(response.Data.GetEnrollments()),
			response.Data.GetNextCursor(),
			response.Data.GetPrevCursor(),
			response.Data.GetHasMore(),
		)
	}
}

func TestListCandidateEnrollmentsUsesDefaultPageSizeForInvalidInput(t *testing.T) {
	lms := &enrollmentLMSClientStub{
		listResponse: &lmspb.ListCandidateEnrollmentsResponse{},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/enrollments?pageSize=invalid",
		"",
		"candidate-1",
		nil,
	)
	recorder := httptest.NewRecorder()

	(&Handler{Lms: lms}).ListCandidateEnrollments(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if lms.listRequest == nil || lms.listRequest.GetPageSize() != 20 {
		t.Fatalf("page_size = %d, want 20", lms.listRequest.GetPageSize())
	}
}

func TestGetCandidateEnrollmentDetailForwardsCandidateOwnership(t *testing.T) {
	lms := &enrollmentLMSClientStub{
		detailResponse: &lmspb.GetCandidateEnrollmentDetailResponse{
			EnrollmentId:       "enrollment-1",
			CandidateUlid:      "candidate-1",
			CourseUlid:         "course-1",
			Status:             "completed",
			ProgressPercentage: 100,
		},
	}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/enrollments/enrollment-1",
		"",
		"candidate-1",
		map[string]string{"enrollmentId": " enrollment-1 "},
	)
	recorder := httptest.NewRecorder()

	(&Handler{Lms: lms}).GetCandidateEnrollmentDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if lms.detailRequest == nil ||
		lms.detailRequest.GetCandidateUlid() != "candidate-1" ||
		lms.detailRequest.GetEnrollmentId() != "enrollment-1" {
		t.Fatalf("GetCandidateEnrollmentDetail request = %+v", lms.detailRequest)
	}
}

func TestGetCandidateEnrollmentDetailRejectsMissingIDBeforeCallingLMS(t *testing.T) {
	lms := &enrollmentLMSClientStub{}
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/enrollments/",
		"",
		"candidate-1",
		map[string]string{"enrollmentId": "   "},
	)
	recorder := httptest.NewRecorder()

	(&Handler{Lms: lms}).GetCandidateEnrollmentDetail(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
	if lms.detailRequest != nil {
		t.Fatal("missing enrollment ID must not call LMS")
	}
}
