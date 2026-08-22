package handler

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gccpb "github.com/afnandelfin620-star/cftptest/cftp/gcc"
	"google.golang.org/grpc"
)

type pipelineStructureClientStub struct {
	gccpb.CCServiceClient
	request *gccpb.UpdatePipelineStructureRequest
}

func (s *pipelineStructureClientStub) UpdatePipelineStructure(_ context.Context, req *gccpb.UpdatePipelineStructureRequest, _ ...grpc.CallOption) (*gccpb.PipelineConfig, error) {
	s.request = req
	return &gccpb.PipelineConfig{PipelineUlid: req.GetPipelineUlid()}, nil
}

func TestCatalogHandlersReportUnavailableContract(t *testing.T) {
	tests := []struct {
		name   string
		method string
		target string
		handle func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{name: "list", method: http.MethodGet, target: "/api/catalogs", handle: (*Handler).ListCatalogs},
		{name: "create", method: http.MethodPost, target: "/api/catalogs", handle: (*Handler).CreateCatalog},
		{name: "update", method: http.MethodPut, target: "/api/catalogs/catalog-1", handle: (*Handler).UpdateCatalog},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.handle(&Handler{}, recorder, httptest.NewRequest(test.method, test.target, nil))

			if recorder.Code != http.StatusNotImplemented {
				t.Fatalf("status = %d, want %d; body = %s", recorder.Code, http.StatusNotImplemented, recorder.Body.String())
			}
			var response struct {
				ErrorCode ErrorCode `json:"error_code"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if response.ErrorCode != ErrNotImplemented {
				t.Fatalf("error code = %q, want %q", response.ErrorCode, ErrNotImplemented)
			}
		})
	}
}

func TestUpdatePipelineStructureUsesSemanticQualificationAndCertificateFields(t *testing.T) {
	const body = `{
		"stages":[{"name":"Stage","units":[{"glms_course_ulid":"01KYN000000000000000000001"}]}],
		"prerequisite_quals":[{"qual_ulid":"01KYN000000000000000000002","name_hint":"CFTA"}],
		"final_audit_quals":[{"qual_ulid":"01KYN000000000000000000003","name_hint":"Work Experience"}],
		"award_certs":[{"qual_ulid":"01KYN000000000000000000004","pdf_template_ulid":"01KYN000000000000000000005","name_hint":"CFTP"}],
		"forbidden_quals":[{"qual_ulid":"01KYN000000000000000000006","name_hint":"Advanced"}],
		"conflict_pipeline_gpaths":["/gcc/pipeline/core/cfta"]
	}`
	client := &pipelineStructureClientStub{}
	recorder := httptest.NewRecorder()
	request := requestWithURLParam(http.MethodPut, "/api/pipelines/01KYN000000000000000000007/structure", "pipeline_id", "01KYN000000000000000000007")
	request.Body = http.NoBody
	request = request.Clone(request.Context())
	request.Body = io.NopCloser(strings.NewReader(body))

	(&Handler{Gcc: client}).UpdatePipelineStructure(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.request == nil {
		t.Fatal("UpdatePipelineStructure was not called")
	}
	if got := client.request.GetAwardCerts()[0]; got.GetPdfTemplateUlid() != "01KYN000000000000000000005" || got.GetNameHint() != "CFTP" {
		t.Fatalf("award certificate fields were not preserved: %+v", got)
	}
	if len(client.request.GetPrerequisiteQuals()) != 1 || len(client.request.GetFinalAuditQuals()) != 1 || len(client.request.GetForbiddenQuals()) != 1 {
		t.Fatalf("qualification requirements were not preserved: %+v", client.request)
	}
	if got := client.request.GetConflictPipelineGpaths(); len(got) != 1 || got[0] != "/gcc/pipeline/core/cfta" {
		t.Fatalf("conflict pipeline gpaths = %v", got)
	}
}

func TestUpdatePipelineStructureRejectsObsoleteTopLevelFields(t *testing.T) {
	client := &pipelineStructureClientStub{}
	recorder := httptest.NewRecorder()
	request := requestWithURLParam(http.MethodPut, "/api/pipelines/pipeline-1/structure", "pipeline_id", "pipeline-1")
	request.Body = io.NopCloser(strings.NewReader(`{"stages":[],"certs":[]}`))

	(&Handler{Gcc: client}).UpdatePipelineStructure(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if client.request != nil {
		t.Fatal("obsolete structure request reached GCC")
	}
}
