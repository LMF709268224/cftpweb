package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"google.golang.org/grpc"
)

type messageReadClientStub struct {
	gmsgpb.MessageServiceClient
	listRequest  *gmsgpb.ListMessagesAdminRequest
	countRequest *gmsgpb.GetMessageCountAdminRequest
}

func (s *messageReadClientStub) ListMessagesAdmin(
	_ context.Context,
	req *gmsgpb.ListMessagesAdminRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.ListMessagesAdminResponse, error) {
	s.listRequest = req
	return &gmsgpb.ListMessagesAdminResponse{
		NextCursor: "next-page",
		HasMore:    true,
	}, nil
}

func (s *messageReadClientStub) GetMessageCountAdmin(
	_ context.Context,
	req *gmsgpb.GetMessageCountAdminRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.GetMessageCountAdminResponse, error) {
	s.countRequest = req
	return &gmsgpb.GetMessageCountAdminResponse{Count: 4}, nil
}

func TestListSentMessagesReturnsReadOnlyMessagePage(t *testing.T) {
	client := &messageReadClientStub{}
	h := &Handler{Gmsg: client}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/messages?status=1&page_size=25&cursor=current-page", nil)

	h.ListSentMessages(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest == nil || client.countRequest == nil {
		t.Fatal("ListSentMessages() did not call both read-only message queries")
	}
	if client.listRequest.GetPageSize() != 25 || client.listRequest.GetCursor() != "current-page" {
		t.Fatalf("list request = %+v", client.listRequest)
	}
	if client.listRequest.GetFilters().GetStatus() != gmsgpb.MessageStatus(1) || client.countRequest.GetFilters().GetStatus() != gmsgpb.MessageStatus(1) {
		t.Fatalf("message filters = %v / %v", client.listRequest.GetFilters(), client.countRequest.GetFilters())
	}

	var payload struct {
		Data struct {
			Total      uint32 `json:"total"`
			NextCursor string `json:"next_cursor"`
			HasMore    bool   `json:"has_more"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Data.Total != 4 || payload.Data.NextCursor != "next-page" || !payload.Data.HasMore {
		t.Fatalf("message page = %+v", payload.Data)
	}
}
