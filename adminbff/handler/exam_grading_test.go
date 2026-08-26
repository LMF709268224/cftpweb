package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
)

type examGradingClientStub struct {
	gexampb.GExamServiceClient
	countRequest  *gexampb.GetPendingGradingExamCountRequest
	listRequest   *gexampb.ListPendingGradingExamsRequest
	detailRequest *gexampb.GetExamEssayDetailsRequest
	submitRequest *gexampb.SubmitExamEssayGradeRequest
}

func (s *examGradingClientStub) GetPendingGradingExamCount(_ context.Context, request *gexampb.GetPendingGradingExamCountRequest, _ ...grpc.CallOption) (*gexampb.GetPendingGradingExamCountResponse, error) {
	s.countRequest = request
	return &gexampb.GetPendingGradingExamCountResponse{Count: 1}, nil
}

func (s *examGradingClientStub) ListPendingGradingExams(_ context.Context, request *gexampb.ListPendingGradingExamsRequest, _ ...grpc.CallOption) (*gexampb.ListPendingGradingExamsResponse, error) {
	s.listRequest = request
	return &gexampb.ListPendingGradingExamsResponse{Items: []*gexampb.PendingGradingExamItem{{
		ExamUlid:           "exam-essay-1",
		CandidateFirstName: "Ada",
		CandidateLastName:  "Lovelace",
		CandidateEmail:     "ada@example.test",
		ObjectiveScore:     60,
		EssayCount:         2,
		ResultStatus:       "PENDING_GRADING",
	}}}, nil
}

func (s *examGradingClientStub) GetExamEssayDetails(_ context.Context, request *gexampb.GetExamEssayDetailsRequest, _ ...grpc.CallOption) (*gexampb.GetExamEssayDetailsResponse, error) {
	s.detailRequest = request
	return &gexampb.GetExamEssayDetailsResponse{
		ExamUlid:           request.GetExamUlid(),
		CandidateFirstName: "Ada",
		CandidateLastName:  "Lovelace",
		ObjectiveScore:     60,
		ResultStatus:       "PENDING_GRADING",
		Essays:             []*gexampb.ExamEssayItemDetail{{QuestionSeq: 1, CandidateResponse: "Essay response", MaxScore: 20}},
	}, nil
}

func (s *examGradingClientStub) SubmitExamEssayGrade(_ context.Context, request *gexampb.SubmitExamEssayGradeRequest, _ ...grpc.CallOption) (*gexampb.SubmitExamEssayGradeResponse, error) {
	s.submitRequest = request
	return &gexampb.SubmitExamEssayGradeResponse{ExamUlid: request.GetExamUlid(), FinalTotalScore: 78, IsPassed: request.GetIsPassed(), ResultStatus: "FETCHED"}, nil
}

func examGradingRequest(method, target, body, examID string) *http.Request {
	request := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if examID != "" {
		routeContext := chi.NewRouteContext()
		routeContext.URLParams.Add("exam_ulid", examID)
		request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
	}
	return request
}

func TestListPendingGradingExamsForwardsFilters(t *testing.T) {
	client := &examGradingClientStub{}
	recorder := httptest.NewRecorder()
	request := examGradingRequest(http.MethodGet, "/api/exams/pending-grading?program_code=CFTP&exam_code=L2A&keyword=ada&page_size=25", "", "")

	(&Handler{Gexam: client}).ListPendingGradingExams(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest.GetFilters().GetProgramCode() != "CFTP" || client.listRequest.GetFilters().GetExamCode() != "L2A" || client.listRequest.GetFilters().GetKeyword() != "ada" || client.listRequest.GetPageSize() != 25 {
		t.Fatalf("list request = %+v", client.listRequest)
	}
	var payload struct {
		Data struct {
			Total int `json:"total"`
			Items []struct {
				ExamULID string `json:"exam_ulid"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 1 || len(payload.Data.Items) != 1 || payload.Data.Items[0].ExamULID != "exam-essay-1" {
		t.Fatalf("pending grading response = %+v", payload.Data)
	}
}

func TestGetExamEssayDetailsUsesRouteExam(t *testing.T) {
	client := &examGradingClientStub{}
	recorder := httptest.NewRecorder()
	request := examGradingRequest(http.MethodGet, "/api/exams/exam-essay-1/essay-details", "", "exam-essay-1")

	(&Handler{Gexam: client}).GetExamEssayDetails(recorder, request)

	if recorder.Code != http.StatusOK || client.detailRequest.GetExamUlid() != "exam-essay-1" {
		t.Fatalf("status = %d, request = %+v; body=%s", recorder.Code, client.detailRequest, recorder.Body.String())
	}
}

func TestSubmitExamEssayGradeUsesAuthenticatedIdentity(t *testing.T) {
	client := &examGradingClientStub{}
	recorder := httptest.NewRecorder()
	request := examGradingRequest(http.MethodPost, "/api/exams/exam-essay-1/essay-grade", `{
		"grader_id":"forged-web-id",
		"grader_name":"Forged Web Name",
		"is_passed":true,
		"overall_comment":"Meets the standard",
		"items":[{"question_seq":1,"score":18,"max_score":20,"comment":"Clear analysis"}]
	}`, "exam-essay-1")
	request = request.WithContext(WithCandidate(request.Context(), "professor-token-id", "professor@example.test", "Professor Lee", "token"))

	(&Handler{Gexam: client}).SubmitExamEssayGrade(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	got := client.submitRequest
	if got == nil || got.GetExamUlid() != "exam-essay-1" || got.GetGraderId() != "professor-token-id" || got.GetGraderName() != "Professor Lee" || !got.GetIsPassed() {
		t.Fatalf("submit request = %+v", got)
	}
	if len(got.GetItems()) != 1 || got.GetItems()[0].GetScore() != 18 || got.GetItems()[0].GetMaxScore() != 20 {
		t.Fatalf("grade items = %+v", got.GetItems())
	}
}
