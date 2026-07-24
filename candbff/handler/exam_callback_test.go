package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"github.com/go-chi/chi/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type examCallbackProgClientStub struct {
	gprogpb.ProgServiceClient
	err     error
	lastReq *gprogpb.ExamUrlCallbackReq
	calls   int
}

func (s *examCallbackProgClientStub) ExamUrlCallback(
	_ context.Context,
	req *gprogpb.ExamUrlCallbackReq,
	_ ...grpc.CallOption,
) (*gprogpb.ExamUrlCallbackRsp, error) {
	s.calls++
	s.lastReq = req
	if s.err != nil {
		return nil, s.err
	}
	return &gprogpb.ExamUrlCallbackRsp{ExamUlid: req.GetExamUlid(), ExamStatus: "SCHEDULED"}, nil
}

func examCallbackRequest(body string, urlType string, examID string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/api/public/webhooks/exams/callback/test/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	routeContext := chi.NewRouteContext()
	routeContext.URLParams.Add("urlType", urlType)
	routeContext.URLParams.Add("examId", examID)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
}

func TestParseExamURLTypeValueRejectsUnknownNumericValue(t *testing.T) {
	if got, ok := parseExamURLTypeValue("999"); ok || got != gprogpb.ExamURLType_EXAM_URL_TYPE_UNKNOWN {
		t.Fatalf("parseExamURLTypeValue(999) = (%v, %t), want unknown false", got, ok)
	}
}

func TestRenderExamCallbackHTMLEscapesMessage(t *testing.T) {
	rec := httptest.NewRecorder()
	renderExamCallbackHTML(rec, false, `<script>alert("xss")</script>`)

	body := rec.Body.String()
	if strings.Contains(body, "<script>alert") {
		t.Fatalf("response contains unescaped script: %s", body)
	}
	if !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("response does not contain escaped message: %s", body)
	}
}

func TestThirdPartyExamCallbackRejectsOversizedBody(t *testing.T) {
	prog := &examCallbackProgClientStub{}
	h := &Handler{Gprog: prog}
	body := url.Values{"apptdata": {strings.Repeat("x", maxExamCallbackBodyBytes+1)}}.Encode()
	req := examCallbackRequest(body, "schd", "exam-1")
	rec := httptest.NewRecorder()

	h.ThirdPartyExamCallback(rec, req)

	if prog.calls != 0 {
		t.Fatalf("gprog calls = %d, want 0", prog.calls)
	}
	if !strings.Contains(rec.Body.String(), "invalid callback data") {
		t.Fatalf("response = %q, want invalid callback data", rec.Body.String())
	}
}

func TestThirdPartyExamCallbackNormalizesTypeAndRelaysPayload(t *testing.T) {
	prog := &examCallbackProgClientStub{}
	h := &Handler{Gprog: prog}
	payload := `<appointment id="123">scheduled</appointment>`
	req := examCallbackRequest(url.Values{"apptdata": {payload}}.Encode(), "SCHD", " exam-1 ")
	rec := httptest.NewRecorder()

	h.ThirdPartyExamCallback(rec, req)

	if prog.calls != 1 {
		t.Fatalf("gprog calls = %d, want 1", prog.calls)
	}
	if prog.lastReq.GetExamUlid() != "exam-1" || prog.lastReq.GetUrlType() != "schd" {
		t.Fatalf("gprog request = %+v, want trimmed exam ID and canonical URL type", prog.lastReq)
	}
	var callbackBody map[string]string
	if err := json.Unmarshal([]byte(prog.lastReq.GetCallbackBody()), &callbackBody); err != nil {
		t.Fatalf("callback body is not valid JSON: %v", err)
	}
	if callbackBody["raw_xml"] != payload {
		t.Fatalf("raw_xml = %q, want original payload", callbackBody["raw_xml"])
	}
	if !strings.Contains(rec.Body.String(), "synced successfully") {
		t.Fatalf("response = %q, want success page", rec.Body.String())
	}
}

func TestThirdPartyExamCallbackDoesNotExposeBackendError(t *testing.T) {
	prog := &examCallbackProgClientStub{
		err: status.Error(codes.Internal, `<script>alert("backend")</script>`),
	}
	h := &Handler{Gprog: prog}
	req := examCallbackRequest(url.Values{"apptdata": {"payload"}}.Encode(), "schd", "exam-1")
	rec := httptest.NewRecorder()

	h.ThirdPartyExamCallback(rec, req)

	body := rec.Body.String()
	if strings.Contains(body, "alert") || strings.Contains(body, "backend</") {
		t.Fatalf("response exposes backend error: %s", body)
	}
	if !strings.Contains(body, "backend processing failed") {
		t.Fatalf("response = %q, want generic backend failure", body)
	}
}
