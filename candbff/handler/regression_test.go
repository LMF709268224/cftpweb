package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type retakeMallClientStub struct {
	mallpb.MallServiceClient
	statusResp    *mallpb.GetCourseUnitRetakePaymentStatusResponse
	statusErr     error
	lastStatusReq *mallpb.GetCourseUnitRetakePaymentStatusRequest
}

func (s *retakeMallClientStub) GetCourseUnitRetakePaymentStatus(
	_ context.Context,
	req *mallpb.GetCourseUnitRetakePaymentStatusRequest,
	_ ...grpc.CallOption,
) (*mallpb.GetCourseUnitRetakePaymentStatusResponse, error) {
	s.lastStatusReq = req
	if s.statusErr != nil {
		return nil, s.statusErr
	}
	if s.statusResp != nil {
		return s.statusResp, nil
	}
	return &mallpb.GetCourseUnitRetakePaymentStatusResponse{}, nil
}

type retakeProgClientStub struct {
	gprogpb.ProgServiceClient
	detailResp    *gprogpb.GetCourseUnitDetailRsp
	detailErr     error
	lastDetailReq *gprogpb.GetCourseUnitDetailReq
}

func (s *retakeProgClientStub) GetCourseUnitDetail(
	_ context.Context,
	req *gprogpb.GetCourseUnitDetailReq,
	_ ...grpc.CallOption,
) (*gprogpb.GetCourseUnitDetailRsp, error) {
	s.lastDetailReq = req
	if s.detailErr != nil {
		return nil, s.detailErr
	}
	return s.detailResp, nil
}

func TestApplyExamHistoryDefaults(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/exams/history?result_status=NO_SHOW", nil)

	applyExamHistoryDefaults(req)

	query := req.URL.Query()
	if got := query.Get("status"); got != "DONE" {
		t.Fatalf("status = %q, want DONE", got)
	}
	if got := query.Get("result_status"); got != "NO_SHOW" {
		t.Fatalf("result_status = %q, want NO_SHOW", got)
	}
}

func TestApplyExamHistoryDefaultsPreservesExplicitStatus(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/exams/history?status=SCHEDULED", nil)

	applyExamHistoryDefaults(req)

	if got := req.URL.Query().Get("status"); got != "SCHEDULED" {
		t.Fatalf("status = %q, want SCHEDULED", got)
	}
}

func TestRetakePaymentSnapshotPropagatesLookupErrors(t *testing.T) {
	wantErr := status.Error(codes.Unavailable, "gmall unavailable")
	h := &Handler{Mall: &retakeMallClientStub{statusErr: wantErr}}

	_, err := h.retakePaymentSnapshot(context.Background(), "unit", "unit-config", "bundle-order", "pipeline", 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("retakePaymentSnapshot() error = %v, want %v", err, wantErr)
	}
}

func TestRetakePaymentSnapshotUsesExactStatusResponse(t *testing.T) {
	mall := &retakeMallClientStub{statusResp: &mallpb.GetCourseUnitRetakePaymentStatusResponse{
		Found:                 true,
		Paid:                  false,
		IsFree:                false,
		Message:               "payment required",
		CourseRetakeOrderUlid: "retake-order",
		OrderStatus:           "WAIT_RETAKE_PAYMENT",
		PayOrderUlid:          "pay-order",
	}}
	h := &Handler{Mall: mall}

	got, err := h.retakePaymentSnapshot(context.Background(), "unit", "unit-config", "bundle-order", "pipeline", 2)
	if err != nil {
		t.Fatalf("retakePaymentSnapshot() error = %v", err)
	}
	if mall.lastStatusReq.GetCourseUnitUlid() != "unit" ||
		mall.lastStatusReq.GetCourseUnitCcUlid() != "unit-config" ||
		mall.lastStatusReq.GetBundleOrderUlid() != "bundle-order" ||
		mall.lastStatusReq.GetPipelineUlid() != "pipeline" ||
		mall.lastStatusReq.GetRetriedCount() != 2 {
		t.Fatalf("status request = %+v, want exact retake lookup fields", mall.lastStatusReq)
	}
	if !got.found || got.paid || got.isFree {
		t.Fatalf("payment flags = found:%t paid:%t free:%t", got.found, got.paid, got.isFree)
	}
	if got.message != "payment required" {
		t.Fatalf("payment message = %q, want payment required", got.message)
	}
	if got.courseRetakeOrderUlid != "retake-order" || got.orderStatus != "WAIT_RETAKE_PAYMENT" || got.payOrderUlid != "pay-order" {
		t.Fatalf("payment order snapshot = %+v", got)
	}
}

func TestRetakePipelineUlidUsesCourseUnitDetail(t *testing.T) {
	prog := &retakeProgClientStub{detailResp: &gprogpb.GetCourseUnitDetailRsp{
		CourseUnitUlid:   "unit",
		CourseUnitCcUlid: "unit-config",
		PipelineUlid:     "pipeline",
	}}
	h := &Handler{Gprog: prog}

	got, err := h.retakePipelineUlid(context.Background(), "unit", "unit-config")
	if err != nil {
		t.Fatalf("retakePipelineUlid() error = %v", err)
	}
	if got != "pipeline" {
		t.Fatalf("retakePipelineUlid() = %q, want pipeline", got)
	}
	if prog.lastDetailReq.GetCourseUnitUlid() != "unit" {
		t.Fatalf("course unit detail request = %+v", prog.lastDetailReq)
	}
}

func TestRetakePipelineUlidRejectsMismatchedCourseUnitConfig(t *testing.T) {
	h := &Handler{Gprog: &retakeProgClientStub{detailResp: &gprogpb.GetCourseUnitDetailRsp{
		CourseUnitUlid:   "unit",
		CourseUnitCcUlid: "actual-config",
		PipelineUlid:     "pipeline",
	}}}

	_, err := h.retakePipelineUlid(context.Background(), "unit", "requested-config")
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("retakePipelineUlid() error = %v, want InvalidArgument", err)
	}
}

func TestRetakePipelineUlidRejectsMissingPipeline(t *testing.T) {
	h := &Handler{Gprog: &retakeProgClientStub{detailResp: &gprogpb.GetCourseUnitDetailRsp{
		CourseUnitUlid:   "unit",
		CourseUnitCcUlid: "unit-config",
	}}}

	_, err := h.retakePipelineUlid(context.Background(), "unit", "unit-config")
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("retakePipelineUlid() error = %v, want FailedPrecondition", err)
	}
}

func TestRetakeActionForPayment(t *testing.T) {
	tests := []struct {
		name    string
		payment retakePaymentSnapshot
		want    string
	}{
		{name: "free without order", payment: retakePaymentSnapshot{isFree: true}, want: retakeActionCreateRetakeOrder},
		{name: "free completed order", payment: retakePaymentSnapshot{isFree: true, found: true, orderStatus: "COMPLETED"}, want: retakeActionApplyRetake},
		{name: "free incomplete order", payment: retakePaymentSnapshot{isFree: true, found: true, orderStatus: "FAILED"}, want: retakeActionNone},
		{name: "paid order", payment: retakePaymentSnapshot{found: true, paid: true}, want: retakeActionApplyRetake},
		{name: "unpaid existing order", payment: retakePaymentSnapshot{found: true}, want: retakeActionContinuePayment},
		{name: "paid order missing", payment: retakePaymentSnapshot{}, want: retakeActionCreateRetakeOrder},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := retakeActionForPayment(tt.payment); got != tt.want {
				t.Fatalf("retakeActionForPayment() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsCurrentCourseUnitExam(t *testing.T) {
	tests := []struct {
		name            string
		examUlid        string
		currentExamUlid string
		want            bool
	}{
		{name: "current exam", examUlid: "exam-current", currentExamUlid: "exam-current", want: true},
		{name: "trims identifiers", examUlid: " exam-current ", currentExamUlid: "exam-current", want: true},
		{name: "superseded exam", examUlid: "exam-old", currentExamUlid: "exam-current", want: false},
		{name: "missing list exam", currentExamUlid: "exam-current", want: false},
		{name: "missing current exam", examUlid: "exam-current", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCurrentCourseUnitExam(tt.examUlid, tt.currentExamUlid); got != tt.want {
				t.Fatalf("isCurrentCourseUnitExam(%q, %q) = %t, want %t", tt.examUlid, tt.currentExamUlid, got, tt.want)
			}
		})
	}
}

func TestValidateTelemetryBatch(t *testing.T) {
	t.Run("accepts valid batch and trims fields", func(t *testing.T) {
		batch := TelemetryBatch{Events: []TelemetryEvent{{
			EventName: " page_view ",
			Timestamp: " 2026-07-21T10:00:00Z ",
			URL:       " https://example.test/exams ",
			Payload:   map[string]interface{}{"tab": "history"},
		}}}

		if err := validateTelemetryBatch(&batch); err != nil {
			t.Fatalf("validateTelemetryBatch() error = %v", err)
		}
		if batch.Events[0].EventName != "page_view" {
			t.Fatalf("event_name = %q, want page_view", batch.Events[0].EventName)
		}
	})

	t.Run("rejects too many events", func(t *testing.T) {
		batch := TelemetryBatch{Events: make([]TelemetryEvent, maxTelemetryEvents+1)}
		if err := validateTelemetryBatch(&batch); err == nil {
			t.Fatal("validateTelemetryBatch() error = nil, want batch size error")
		}
	})

	t.Run("rejects oversized payload", func(t *testing.T) {
		batch := TelemetryBatch{Events: []TelemetryEvent{{
			EventName: "client_error",
			Payload: map[string]interface{}{
				"message": strings.Repeat("x", maxTelemetryPayloadJSONSize),
			},
		}}}
		if err := validateTelemetryBatch(&batch); err == nil {
			t.Fatal("validateTelemetryBatch() error = nil, want payload size error")
		}
	})
}

func TestReportTelemetryLimits(t *testing.T) {
	t.Run("rejects oversized request body", func(t *testing.T) {
		body := `{"events":[{"event_name":"client_error","payload":{"message":"` +
			strings.Repeat("x", (1<<20)+1) +
			`"}}]}`
		req := httptest.NewRequest("POST", "/api/public/telemetry", strings.NewReader(body))
		rec := httptest.NewRecorder()

		(&Handler{}).ReportTelemetry(rec, req)

		if rec.Code != 413 {
			t.Fatalf("status = %d, want 413", rec.Code)
		}
	})

	t.Run("rejects oversized event batch", func(t *testing.T) {
		events := make([]TelemetryEvent, maxTelemetryEvents+1)
		for i := range events {
			events[i].EventName = "page_view"
		}
		body, err := json.Marshal(TelemetryBatch{Events: events})
		if err != nil {
			t.Fatalf("json.Marshal() error = %v", err)
		}
		req := httptest.NewRequest("POST", "/api/public/telemetry", strings.NewReader(string(body)))
		rec := httptest.NewRecorder()

		(&Handler{}).ReportTelemetry(rec, req)

		if rec.Code != 400 {
			t.Fatalf("status = %d, want 400", rec.Code)
		}
	})
}

func TestMessageListResponseIncludesPrevCursor(t *testing.T) {
	body, err := json.Marshal(MessageListRsp{PrevCursor: "previous-page"})
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if !strings.Contains(string(body), `"prev_cursor":"previous-page"`) {
		t.Fatalf("response JSON %s does not include prev_cursor", body)
	}
}
