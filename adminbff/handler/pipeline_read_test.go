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

type pipelineReadClientStub struct {
	gccpb.CCServiceClient
	listRequest   *gccpb.ListPipelinesAdminRequest
	detailRequest *gccpb.GetPipelineRequest
}

func (s *pipelineReadClientStub) ListPipelinesAdmin(_ context.Context, req *gccpb.ListPipelinesAdminRequest, _ ...grpc.CallOption) (*gccpb.ListPipelinesResponse, error) {
	s.listRequest = req
	return &gccpb.ListPipelinesResponse{
		Pipelines: []*gccpb.PipelineSummary{{
			PipelineUlid:  "pipeline-1",
			PipelineGpath: "/pipelines/regression",
			Name:          "Regression Pipeline",
			Description:   "Read-only pipeline summary",
			CategoryTips:  "Automation",
			Status:        "Active",
			Version:       3,
		}},
		HasMore:    true,
		NextCursor: "next-pipeline-page",
	}, nil
}

func (s *pipelineReadClientStub) GetPipeline(_ context.Context, req *gccpb.GetPipelineRequest, _ ...grpc.CallOption) (*gccpb.PipelineConfig, error) {
	s.detailRequest = req
	return &gccpb.PipelineConfig{
		PipelineUlid:  "pipeline-1",
		PipelineGpath: "/pipelines/regression",
		Name:          "Regression Pipeline",
		Description:   "Read-only pipeline detail",
		CategoryTips:  "Automation",
		Status:        "Active",
		Version:       3,
	}, nil
}

func TestListPipelinesReturnsFilteredReadOnlyPage(t *testing.T) {
	client := &pipelineReadClientStub{}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/pipelines?status=published&category_tips=Automation&only_current=true&page_token=current-pipeline&page_size=10", nil)

	(&Handler{Gcc: client}).ListPipelines(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil {
		t.Fatal("ListPipelines() did not call the read-only pipeline query")
	}
	filters := client.listRequest.GetFilters()
	if filters.GetStatus() != "Active" || filters.GetCategoryTips() != "Automation" || !filters.GetOnlyCurrent() || client.listRequest.GetCursor() != "current-pipeline" || client.listRequest.GetPageSize() != 10 {
		t.Fatalf("pipeline request = %+v", client.listRequest)
	}

	var payload struct {
		Data struct {
			Pipelines []struct {
				PipelineULID string `json:"pipeline_ulid"`
				Name         string `json:"name"`
			} `json:"pipelines"`
			HasMore    bool   `json:"has_more"`
			NextCursor string `json:"next_cursor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.Pipelines) != 1 || payload.Data.Pipelines[0].PipelineULID != "pipeline-1" || payload.Data.Pipelines[0].Name != "Regression Pipeline" || !payload.Data.HasMore || payload.Data.NextCursor != "next-pipeline-page" {
		t.Fatalf("pipeline page = %+v", payload.Data)
	}
}

func TestGetPipelineReturnsReadOnlyDetail(t *testing.T) {
	client := &pipelineReadClientStub{}
	recorder := httptest.NewRecorder()

	(&Handler{Gcc: client}).GetPipeline(recorder, requestWithURLParam(http.MethodGet, "/api/pipelines/pipeline-1", "pipeline_id", "pipeline-1"))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest == nil || client.detailRequest.GetPipelineUlid() != "pipeline-1" {
		t.Fatalf("pipeline detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			PipelineULID string `json:"pipeline_ulid"`
			Description  string `json:"description"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.PipelineULID != "pipeline-1" || payload.Data.Description != "Read-only pipeline detail" {
		t.Fatalf("pipeline detail = %+v", payload.Data)
	}
}
