package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	"google.golang.org/grpc"
)

type examGradingFilterClientStub struct {
	gccpb.CCServiceClient
	listRequests   []*gccpb.ListPipelinesAdminRequest
	detailRequests []*gccpb.GetPipelineRequest
	details        map[string]*gccpb.PipelineConfig
}

func (s *examGradingFilterClientStub) ListPipelinesAdmin(_ context.Context, req *gccpb.ListPipelinesAdminRequest, _ ...grpc.CallOption) (*gccpb.ListPipelinesResponse, error) {
	s.listRequests = append(s.listRequests, req)
	if req.GetCursor() == "" {
		return &gccpb.ListPipelinesResponse{
			Pipelines:  []*gccpb.PipelineSummary{{PipelineUlid: "pipeline-2"}},
			HasMore:    true,
			NextCursor: "next-page",
		}, nil
	}
	return &gccpb.ListPipelinesResponse{
		Pipelines: []*gccpb.PipelineSummary{{PipelineUlid: "pipeline-1"}},
	}, nil
}

func (s *examGradingFilterClientStub) GetPipeline(_ context.Context, req *gccpb.GetPipelineRequest, _ ...grpc.CallOption) (*gccpb.PipelineConfig, error) {
	s.detailRequests = append(s.detailRequests, req)
	return s.details[req.GetPipelineUlid()], nil
}

func gradingFilterPipeline(t *testing.T, payload string) *gccpb.PipelineConfig {
	t.Helper()
	var pipeline gccpb.PipelineConfig
	if err := json.Unmarshal([]byte(payload), &pipeline); err != nil {
		t.Fatalf("decode pipeline fixture: %v", err)
	}
	return &pipeline
}

func TestListExamGradingFilterOptionsLoadsActiveCurrentPipelineConfiguration(t *testing.T) {
	client := &examGradingFilterClientStub{details: map[string]*gccpb.PipelineConfig{
		"pipeline-1": gradingFilterPipeline(t, `{
			"stages": [{"units": [
				{"program_code": "CFTP", "exam_code": "L2B", "exam_form": "B"},
				{"program_code": "CFTP", "exam_code": "L2A", "exam_form": "A"},
				{"program_code": "", "exam_code": "INCOMPLETE", "exam_form": "X"}
			]}]
		}`),
		"pipeline-2": gradingFilterPipeline(t, `{
			"stages": [{"units": [
				{"program_code": "CFTE", "exam_code": "L1", "exam_form": "Online"},
				{"program_code": "CFTP", "exam_code": "L2A", "exam_form": "A"}
			]}]
		}`),
	}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/exams/grading-filter-options", nil)

	(&Handler{Gcc: client}).ListExamGradingFilterOptions(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if len(client.listRequests) != 2 || client.listRequests[0].GetCursor() != "" || client.listRequests[1].GetCursor() != "next-page" {
		t.Fatalf("pipeline list requests = %+v", client.listRequests)
	}
	filters := client.listRequests[0].GetFilters()
	if filters.GetStatus() != "Active" || !filters.GetOnlyCurrent() || client.listRequests[0].GetPageSize() != 100 {
		t.Fatalf("pipeline filters = %+v", client.listRequests[0])
	}
	if len(client.detailRequests) != 2 {
		t.Fatalf("pipeline detail requests = %d, want 2", len(client.detailRequests))
	}

	var payload struct {
		Data struct {
			Options []examGradingFilterOption `json:"options"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := []examGradingFilterOption{
		{ProgramCode: "CFTE", ExamCode: "L1", ExamForm: "Online"},
		{ProgramCode: "CFTP", ExamCode: "L2A", ExamForm: "A"},
		{ProgramCode: "CFTP", ExamCode: "L2B", ExamForm: "B"},
	}
	if len(payload.Data.Options) != len(want) {
		t.Fatalf("options = %+v, want %+v", payload.Data.Options, want)
	}
	for index := range want {
		if payload.Data.Options[index] != want[index] {
			t.Fatalf("options = %+v, want %+v", payload.Data.Options, want)
		}
	}
}
