package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type examReadClientStub struct {
	gexampb.GExamServiceClient
	listRequest        *gexampb.ListExamsRequest
	countRequest       *gexampb.GetExamCountRequest
	detailRequest      *gexampb.GetExamRequest
	resultRequest      *gexampb.GetExamRequest
	transitionsRequest *gexampb.GetExamRequest
}

func (s *examReadClientStub) ListExams(_ context.Context, req *gexampb.ListExamsRequest, _ ...grpc.CallOption) (*gexampb.ListExamsResponse, error) {
	s.listRequest = req
	return &gexampb.ListExamsResponse{
		Exams: []*gexampb.ExamInfo{{
			ExamUlid:             "exam-1",
			ProgramCode:          "REG",
			ExamCode:             "REG-101",
			ExamStatus:           "DONE",
			ResultStatus:         "AVAILABLE",
			TotalScore:           88.5,
			IsPassed:             true,
			CandidateFirstName:   "Regression",
			CandidateLastName:    "Candidate",
			CandidateEmail:       "candidate@example.test",
			ConfirmationNumber:   "CONF-001",
			AppointmentStartTime: "2026-08-11T00:00:00Z",
		}},
		HasMore:    true,
		NextCursor: "next-page",
	}, nil
}

func (s *examReadClientStub) GetExamCount(_ context.Context, req *gexampb.GetExamCountRequest, _ ...grpc.CallOption) (*gexampb.GetExamCountResponse, error) {
	s.countRequest = req
	return &gexampb.GetExamCountResponse{Count: 1}, nil
}

func (s *examReadClientStub) GetExamDetail(_ context.Context, req *gexampb.GetExamRequest, _ ...grpc.CallOption) (*gexampb.ExamDetail, error) {
	s.detailRequest = req
	return &gexampb.ExamDetail{
		ExamUlid:           "exam-1",
		ExamCode:           "REG-101",
		CandidateUlid:      "candidate-1",
		CandidateFirstName: "Regression",
		CandidateLastName:  "Candidate",
		CandidateEmail:     "candidate@example.test",
		ExamStatus:         "DONE",
		ResultStatus:       "AVAILABLE",
		ConfirmationNumber: "CONF-001",
		PipelineUlid:       "pipeline-1",
		CourseUnitUlid:     "course-unit-1",
		CertificationName:  "Regression Certification",
	}, nil
}

func (s *examReadClientStub) GetExamResultDetail(_ context.Context, req *gexampb.GetExamRequest, _ ...grpc.CallOption) (*gexampb.ExamResultDetail, error) {
	s.resultRequest = req
	return &gexampb.ExamResultDetail{
		ExamUlid:         "exam-1",
		TotalScore:       88.5,
		IsPassed:         true,
		ScoreDetailsJson: `{"theory":88.5}`,
	}, nil
}

func (s *examReadClientStub) GetExamStatusTransitions(_ context.Context, req *gexampb.GetExamRequest, _ ...grpc.CallOption) (*gexampb.ExamStatusTransitionsResponse, error) {
	s.transitionsRequest = req
	return &gexampb.ExamStatusTransitionsResponse{
		ExamUlid: "exam-1",
		Transitions: []*gexampb.ExamStatusTransition{{
			MsgFp:          "event-1",
			ExamUlid:       "exam-1",
			EventType:      "result_created",
			StatusType:     "RESULT",
			FromStatus:     "NONE",
			ToStatus:       "AVAILABLE",
			TransitionedAt: "2026-08-11T01:00:00Z",
		}},
	}, nil
}

func TestListAdminExamsReturnsFilteredReadOnlyPage(t *testing.T) {
	client := &examReadClientStub{}
	h := &Handler{Gexam: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/exams?status=DONE&result_status=AVAILABLE&candidate_ulid=candidate-1&confirmation_number=CONF-001&course_unit_ulid=course-unit-1&cursor=current-page&page_size=10&sort=1", nil)

	h.ListAdminExams(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	filters := client.listRequest.GetFilters()
	if filters.GetStatus() != "DONE" || filters.GetResultStatus() != "AVAILABLE" || filters.GetCandidateUlid() != "candidate-1" || filters.GetConfirmationNumber() != "CONF-001" || filters.GetCourseUnitUlid() != "course-unit-1" {
		t.Fatalf("exam filters = %+v", filters)
	}
	if client.listRequest.GetCursor() != "current-page" || client.listRequest.GetPageSize() != 10 || int32(client.listRequest.GetSortOrder()) != 1 || client.countRequest == nil {
		t.Fatalf("exam requests = %+v / %+v", client.listRequest, client.countRequest)
	}

	var payload struct {
		Data struct {
			Total      uint32 `json:"total"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
			Exams      []struct {
				ExamULID       string  `json:"exam_ulid"`
				ExamCode       string  `json:"exam_code"`
				TotalScore     float64 `json:"total_score"`
				CandidateEmail string  `json:"candidate_email"`
			} `json:"exams"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 1 || !payload.Data.HasMore || payload.Data.NextCursor != "next-page" || len(payload.Data.Exams) != 1 || payload.Data.Exams[0].ExamULID != "exam-1" || payload.Data.Exams[0].ExamCode != "REG-101" || payload.Data.Exams[0].TotalScore != 88.5 || payload.Data.Exams[0].CandidateEmail != "candidate@example.test" {
		t.Fatalf("exam page = %+v", payload.Data)
	}
}

func TestAdminExamReadOnlyDetailViews(t *testing.T) {
	client := &examReadClientStub{}
	h := &Handler{Gexam: client}
	tests := []struct {
		name   string
		path   string
		handle func(http.ResponseWriter, *http.Request)
		assert func(map[string]interface{}) bool
	}{
		{
			name:   "detail",
			path:   "/api/exams/exam-1",
			handle: h.GetAdminExamDetail,
			assert: func(data map[string]interface{}) bool {
				return data["candidate_email"] == "candidate@example.test" && data["certification_name"] == "Regression Certification"
			},
		},
		{
			name:   "result",
			path:   "/api/exams/exam-1/result",
			handle: h.GetAdminExamResult,
			assert: func(data map[string]interface{}) bool {
				return data["total_score"] == 88.5 && data["is_passed"] == true
			},
		},
		{
			name:   "transitions",
			path:   "/api/exams/exam-1/transitions",
			handle: h.GetAdminExamTransitions,
			assert: func(data map[string]interface{}) bool {
				items, ok := data["transitions"].([]interface{})
				return ok && len(items) == 1
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, test.path, nil)
			routeContext := chi.NewRouteContext()
			routeContext.URLParams.Add("exam_ulid", "exam-1")
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))

			test.handle(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
			}
			var payload struct {
				Data map[string]interface{} `json:"data"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if !test.assert(payload.Data) {
				t.Fatalf("%s response = %+v", test.name, payload.Data)
			}
		})
	}

	if client.detailRequest.GetExamUlid() != "exam-1" || client.resultRequest.GetExamUlid() != "exam-1" || client.transitionsRequest.GetExamUlid() != "exam-1" {
		t.Fatalf("detail requests = %+v / %+v / %+v", client.detailRequest, client.resultRequest, client.transitionsRequest)
	}
}
