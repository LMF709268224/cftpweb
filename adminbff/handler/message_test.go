package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"google.golang.org/grpc"
)

type messageClientStub struct {
	gmsgpb.MessageServiceClient
	sendRequest *gmsgpb.SendMessageRequest
}

func (s *messageClientStub) SendMessage(
	_ context.Context,
	req *gmsgpb.SendMessageRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.SendMessageResponse, error) {
	s.sendRequest = req
	return &gmsgpb.SendMessageResponse{Count: uint32(len(req.GetUserIds()))}, nil
}

func TestSendMessageForwardsAdminContract(t *testing.T) {
	client := &messageClientStub{}
	h := &Handler{Gmsg: client}
	request := httptest.NewRequest(http.MethodPost, "/api/messages/send", strings.NewReader(
		`{"user_ids":["user-1","user-2"],"template_id":"qualification/result","payload":"{\"result\":\"approved\"}"}`,
	))
	request = request.WithContext(WithCandidate(request.Context(), "admin-1", "admin@example.test", "Admin", "token"))
	recorder := httptest.NewRecorder()

	h.SendMessage(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.sendRequest == nil {
		t.Fatal("SendMessage() did not call the message service")
	}
	got := client.sendRequest
	if len(got.GetUserIds()) != 2 || got.GetTemplatePath() != "qualification/result" || got.GetSenderUlid() != "admin-1" {
		t.Fatalf("send request = %+v", got)
	}
	if got.GetMsgSource() != gmsgpb.MsgSource_MANUAL_ADMIN || got.GetMsgType() != gmsgpb.MsgType_SYSTEM_NOTICE {
		t.Fatalf("message source/type = %v/%v", got.GetMsgSource(), got.GetMsgType())
	}
	if strings.TrimSpace(got.GetMessageUlid()) == "" {
		t.Fatal("message_ulid was not generated")
	}
}

func TestSendMessageRejectsIncompleteRequestsBeforeDownstream(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "malformed JSON", body: `{"user_ids":`},
		{name: "missing users", body: `{"template_path":"qualification/result"}`},
		{name: "missing template", body: `{"user_ids":["user-1"]}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := &messageClientStub{}
			h := &Handler{Gmsg: client}
			recorder := httptest.NewRecorder()

			h.SendMessage(recorder, httptest.NewRequest(http.MethodPost, "/api/messages/send", strings.NewReader(test.body)))

			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
			}
			if client.sendRequest != nil {
				t.Fatal("message service was called for an incomplete request")
			}
		})
	}
}
