package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type retakeMallClientStub struct {
	mallpb.MallServiceClient
	statusErr error
	listErr   error
}

func (s *retakeMallClientStub) GetCourseUnitRetakePaymentStatus(
	context.Context,
	*mallpb.GetCourseUnitRetakePaymentStatusRequest,
	...grpc.CallOption,
) (*mallpb.GetCourseUnitRetakePaymentStatusResponse, error) {
	if s.statusErr != nil {
		return nil, s.statusErr
	}
	return &mallpb.GetCourseUnitRetakePaymentStatusResponse{}, nil
}

func (s *retakeMallClientStub) ListCourseRetakeOrders(
	context.Context,
	*mallpb.ListCourseRetakeOrdersRequest,
	...grpc.CallOption,
) (*mallpb.ListCourseRetakeOrdersResponse, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return &mallpb.ListCourseRetakeOrdersResponse{}, nil
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
	t.Run("status lookup unavailable", func(t *testing.T) {
		wantErr := status.Error(codes.Unavailable, "gmall unavailable")
		h := &Handler{Mall: &retakeMallClientStub{statusErr: wantErr}}

		_, err := h.retakePaymentSnapshot(context.Background(), "candidate", "unit", "unit-config", "bundle-order", 1)
		if !errors.Is(err, wantErr) {
			t.Fatalf("retakePaymentSnapshot() error = %v, want %v", err, wantErr)
		}
	})

	t.Run("order list unavailable after status not found", func(t *testing.T) {
		wantErr := status.Error(codes.Unavailable, "order list unavailable")
		h := &Handler{Mall: &retakeMallClientStub{
			statusErr: status.Error(codes.NotFound, "payment status not found"),
			listErr:   wantErr,
		}}

		_, err := h.retakePaymentSnapshot(context.Background(), "candidate", "unit", "unit-config", "bundle-order", 1)
		if !errors.Is(err, wantErr) {
			t.Fatalf("retakePaymentSnapshot() error = %v, want %v", err, wantErr)
		}
	})
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
