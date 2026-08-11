package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	"google.golang.org/grpc"
)

type webhookReadClientStub struct {
	gexampb.GExamServiceClient
	listRequest   *gexampb.ListWebhookMessagesRequest
	detailRequest *gexampb.GetWebhookMessageDetailRequest
}

func (s *webhookReadClientStub) ListWebhookMessages(_ context.Context, req *gexampb.ListWebhookMessagesRequest, _ ...grpc.CallOption) (*gexampb.ListWebhookMessagesResponse, error) {
	s.listRequest = req
	return &gexampb.ListWebhookMessagesResponse{
		WebhookMessages: []*gexampb.WebhookMessageSummary{{
			Id:                 101,
			MsgFp:              "webhook-fingerprint-1",
			EventType:          "result_created",
			EventTimestamp:     "2026-08-11T00:00:00Z",
			ExamUlid:           "exam-1",
			ConfirmationNumber: "CONF-001",
			ProcessedStatus:    "PROCESSED",
			CreatedAt:          "2026-08-11T00:01:00Z",
		}},
		HasMore:    true,
		NextCursor: "next-page",
	}, nil
}

func (s *webhookReadClientStub) GetWebhookMessageDetail(_ context.Context, req *gexampb.GetWebhookMessageDetailRequest, _ ...grpc.CallOption) (*gexampb.WebhookMessageDetail, error) {
	s.detailRequest = req
	return &gexampb.WebhookMessageDetail{
		Id:                 101,
		MsgFp:              "webhook-fingerprint-1",
		EventType:          "result_created",
		ExamUlid:           "exam-1",
		ConfirmationNumber: "CONF-001",
		PayloadJson:        `{"result":"pass"}`,
		ProcessedStatus:    "PROCESSED",
		ProcessedAt:        "2026-08-11T00:02:00Z",
		CreatedAt:          "2026-08-11T00:01:00Z",
	}, nil
}

func TestListWebhookMessagesPassesReadOnlyStatusFilter(t *testing.T) {
	client := &webhookReadClientStub{}
	h := &Handler{Gexam: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/audit/webhooks?status=PROCESSED&cursor=current-page&page_size=10", nil)

	h.ListWebhookMessages(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest.GetFilters().GetProcessedStatus() != "PROCESSED" || client.listRequest.GetCursor() != "current-page" || client.listRequest.GetPageSize() != 10 {
		t.Fatalf("list request = %+v", client.listRequest)
	}
	var payload struct {
		Data struct {
			WebhookMessages []struct {
				ID              uint64 `json:"id"`
				MessageFP       string `json:"msg_fp"`
				ProcessedStatus string `json:"processed_status"`
			} `json:"webhook_messages"`
			HasMore bool `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data.WebhookMessages) != 1 || payload.Data.WebhookMessages[0].ID != 101 || payload.Data.WebhookMessages[0].MessageFP != "webhook-fingerprint-1" || payload.Data.WebhookMessages[0].ProcessedStatus != "PROCESSED" || !payload.Data.HasMore {
		t.Fatalf("webhook page = %+v", payload.Data)
	}
}

func TestGetWebhookMessageDetailReturnsReadOnlyPayload(t *testing.T) {
	client := &webhookReadClientStub{}
	h := &Handler{Gexam: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/audit/webhooks/detail?msg_fp=webhook-fingerprint-1", nil)

	h.GetWebhookMessageDetail(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.detailRequest.GetMsgFp() != "webhook-fingerprint-1" {
		t.Fatalf("detail request = %+v", client.detailRequest)
	}
	var payload struct {
		Data struct {
			MessageFP   string `json:"msg_fp"`
			PayloadJSON string `json:"payload_json"`
			ExamULID    string `json:"exam_ulid"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.MessageFP != "webhook-fingerprint-1" || payload.Data.PayloadJSON != `{"result":"pass"}` || payload.Data.ExamULID != "exam-1" {
		t.Fatalf("webhook detail = %+v", payload.Data)
	}
}
