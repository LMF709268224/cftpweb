package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	"github.com/go-chi/chi/v5"
	"github.com/xuri/excelize/v2"
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
		CandidateUlid:      "candidate-1",
		CandidateFirstName: "Ada",
		CandidateLastName:  "Lovelace",
		CandidateEmail:     "ada@example.test",
		ProgramCode:        "CFTP",
		ExamCode:           "ESSAY-1",
		ExamForm:           "A",
		ObjectiveScore:     60,
		ResultStatus:       "PENDING_GRADING",
		Essays:             []*gexampb.ExamEssayItemDetail{{QuestionSeq: 1, QuestionName: "Risk analysis", SectionName: "Essay", CandidateResponse: "Essay response", MaxScore: 20}},
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

func filledExamGradingWorkbookRequest(t *testing.T, target, examID string, overrides ...map[string]any) *http.Request {
	t.Helper()
	detail := &gexampb.GetExamEssayDetailsResponse{
		ExamUlid:           examID,
		CandidateFirstName: "Ada",
		CandidateLastName:  "Lovelace",
		CandidateEmail:     "ada@example.test",
		ProgramCode:        "CFTP",
		ExamCode:           "ESSAY-1",
		ExamForm:           "A",
		ObjectiveScore:     60,
		ResultStatus:       "PENDING_GRADING",
		Essays:             []*gexampb.ExamEssayItemDetail{{QuestionSeq: 1, QuestionName: "Risk analysis", SectionName: "Essay", CandidateResponse: "Essay response", MaxScore: 20}},
	}
	book, err := buildEssayGradingWorkbook(detail)
	if err != nil {
		t.Fatalf("build workbook: %v", err)
	}
	defer func() { _ = book.Close() }()
	values := map[string]any{
		"B11": "Professor Lee",
		"B12": "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		"B13": "TRUE",
		"B14": "Meets the standard",
		"F17": 18,
		"G17": "Clear analysis",
	}
	for _, override := range overrides {
		for cell, value := range override {
			values[cell] = value
		}
	}
	for cell, value := range values {
		if err := book.SetCellValue(essayGradingSheet, cell, value); err != nil {
			t.Fatalf("set %s: %v", cell, err)
		}
	}

	var workbook bytes.Buffer
	if err := book.Write(&workbook); err != nil {
		t.Fatalf("write workbook: %v", err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "essay-grading.xlsx")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write(workbook.Bytes()); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	request := httptest.NewRequest(http.MethodPost, target, bytes.NewReader(body.Bytes()))
	request.Header.Set("Content-Type", writer.FormDataContentType())
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("exam_ulid", examID)
	return request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, routeContext))
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

func TestExportExamEssayGradeWorkbookReturnsProfessorTemplate(t *testing.T) {
	client := &examGradingClientStub{}
	recorder := httptest.NewRecorder()
	request := examGradingRequest(http.MethodGet, "/api/exams/exam-essay-1/essay-grade/export", "", "exam-essay-1")

	(&Handler{Gexam: client}).ExportExamEssayGradeWorkbook(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if recorder.Header().Get("Content-Type") != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("content type = %q", recorder.Header().Get("Content-Type"))
	}
	book, err := excelize.OpenReader(bytes.NewReader(recorder.Body.Bytes()))
	if err != nil {
		t.Fatalf("open exported workbook: %v", err)
	}
	defer func() { _ = book.Close() }()
	if value, _ := book.GetCellValue(essayGradingSheet, "B4"); value != "exam-essay-1" {
		t.Fatalf("exported exam_ulid = %q", value)
	}
	if value, _ := book.GetCellValue(essayGradingSheet, "D17"); value != "Essay response" {
		t.Fatalf("exported response = %q", value)
	}
	if client.submitRequest != nil {
		t.Fatal("export must not submit a grade")
	}
}

func TestPreviewExamEssayGradeWorkbookDoesNotSubmit(t *testing.T) {
	client := &examGradingClientStub{}
	recorder := httptest.NewRecorder()
	request := filledExamGradingWorkbookRequest(t, "/api/exams/exam-essay-1/essay-grade/import/preview", "exam-essay-1")

	(&Handler{Gexam: client}).PreviewExamEssayGradeWorkbook(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload struct {
		Data essayGradingWorkbook `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if payload.Data.GraderName != "Professor Lee" || payload.Data.FinalScore != 78 || len(payload.Data.Items) != 1 {
		t.Fatalf("preview = %+v", payload.Data)
	}
	if client.submitRequest != nil {
		t.Fatal("preview must not submit a grade")
	}
}

func TestPreviewExamEssayGradeWorkbookRejectsModifiedMaxScore(t *testing.T) {
	client := &examGradingClientStub{}
	recorder := httptest.NewRecorder()
	request := filledExamGradingWorkbookRequest(t, "/api/exams/exam-essay-1/essay-grade/import/preview", "exam-essay-1", map[string]any{"E17": 200})

	(&Handler{Gexam: client}).PreviewExamEssayGradeWorkbook(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if client.submitRequest != nil {
		t.Fatal("invalid workbook must not submit a grade")
	}
}

func TestImportExamEssayGradeWorkbookUsesProfessorIdentityFromWorkbook(t *testing.T) {
	client := &examGradingClientStub{}
	recorder := httptest.NewRecorder()
	request := filledExamGradingWorkbookRequest(t, "/api/exams/exam-essay-1/essay-grade/import", "exam-essay-1")
	request = request.WithContext(WithCandidate(request.Context(), "admin-token-id", "admin@example.test", "Administrator", "token"))

	(&Handler{Gexam: client}).ImportExamEssayGradeWorkbook(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	got := client.submitRequest
	if got == nil || got.GetExamUlid() != "exam-essay-1" || got.GetGraderId() != "01ARZ3NDEKTSV4RRFFQ69G5FAV" || got.GetGraderName() != "Professor Lee" || !got.GetIsPassed() {
		t.Fatalf("submit request = %+v", got)
	}
	if len(got.GetItems()) != 1 || got.GetItems()[0].GetScore() != 18 || got.GetItems()[0].MaxScore != nil {
		t.Fatalf("grade items = %+v", got.GetItems())
	}
}
