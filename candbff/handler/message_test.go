package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	gmsgpb "github.com/afnandelfin620-star/cftptest/cftp/gmsg"
	"google.golang.org/grpc"
)

type messageRegressionClient struct {
	gmsgpb.MessageServiceClient

	listRequest   *gmsgpb.ListMessagesRequest
	countRequest  *gmsgpb.GetMessageCountRequest
	markRequest   *gmsgpb.MarkAsReadRequest
	deleteRequest *gmsgpb.DeleteMessagesRequest
	getRequest    *gmsgpb.GetMessageRequest
	markCalls     int
	deleteCalls   int
}

func (c *messageRegressionClient) ListMessages(
	_ context.Context,
	request *gmsgpb.ListMessagesRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.ListMessagesResponse, error) {
	c.listRequest = request
	return &gmsgpb.ListMessagesResponse{
		Messages: []*gmsgpb.MessageItem{
			{
				Id:           7,
				MessageUlid:  "message-1",
				UserUlid:     "candidate-1",
				TemplatePath: "/candidate/order",
				Status:       gmsgpb.MessageStatus_READ,
				Payload:      `{"order":"ORDER-1","amount":"US$20"}`,
			},
		},
		NextCursor: "next-message",
		PrevCursor: "prev-message",
		HasMore:    true,
	}, nil
}

func (c *messageRegressionClient) GetTemplate(
	_ context.Context,
	_ *gmsgpb.GetTemplateRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.Template, error) {
	return &gmsgpb.Template{
		TitleTpl:   "订单 {{order}}",
		ContentTpl: "金额 {{ amount }}",
	}, nil
}

func (c *messageRegressionClient) GetMessageCount(
	_ context.Context,
	request *gmsgpb.GetMessageCountRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.GetMessageCountResponse, error) {
	c.countRequest = request
	return &gmsgpb.GetMessageCountResponse{Count: 12}, nil
}

func (c *messageRegressionClient) MarkAsRead(
	_ context.Context,
	request *gmsgpb.MarkAsReadRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.CommonResponse, error) {
	c.markCalls++
	c.markRequest = request
	return &gmsgpb.CommonResponse{}, nil
}

func (c *messageRegressionClient) DeleteMessages(
	_ context.Context,
	request *gmsgpb.DeleteMessagesRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.CommonResponse, error) {
	c.deleteCalls++
	c.deleteRequest = request
	return &gmsgpb.CommonResponse{}, nil
}

func (c *messageRegressionClient) GetMessage(
	_ context.Context,
	request *gmsgpb.GetMessageRequest,
	_ ...grpc.CallOption,
) (*gmsgpb.MessageItem, error) {
	c.getRequest = request
	return &gmsgpb.MessageItem{
		Id:          8,
		MessageUlid: request.GetMessageUlid(),
		UserUlid:    request.GetUserUlid(),
		Status:      gmsgpb.MessageStatus_UNREAD,
		Payload:     `{"title":"Direct title","content":"Direct content"}`,
	}, nil
}

func TestListMessagesNormalizesStatusAndCapsPageSize(t *testing.T) {
	client := &messageRegressionClient{}
	handler := &Handler{Gmsg: client}
	recorder := httptest.NewRecorder()
	request := newCandidateHandlerRequest(
		http.MethodGet,
		"/api/messages?status=read&page_size=500&cursor=message-cursor",
		"",
		"candidate-1",
		nil,
	)

	handler.ListMessages(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if client.listRequest.GetFilters().GetUserUlid() != "candidate-1" {
		t.Fatalf("candidate = %q, want candidate-1", client.listRequest.GetFilters().GetUserUlid())
	}
	if client.listRequest.GetFilters().GetStatus() != gmsgpb.MessageStatus_READ {
		t.Fatalf("status filter = %v, want READ", client.listRequest.GetFilters().GetStatus())
	}
	if client.listRequest.GetPageSize() != maxMessageListLimit ||
		client.listRequest.GetCursor() != "message-cursor" {
		t.Fatalf("pagination = (%d, %q)", client.listRequest.GetPageSize(), client.listRequest.GetCursor())
	}

	var response struct {
		Data MessageListRsp `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(response.Data.Messages) != 1 {
		t.Fatalf("messages = %d, want 1", len(response.Data.Messages))
	}
	if response.Data.Messages[0].Title != "订单 ORDER-1" ||
		response.Data.Messages[0].Content != "金额 US$20" {
		t.Fatalf("rendered message = %#v", response.Data.Messages[0])
	}
	if response.Data.NextCursor != "next-message" ||
		response.Data.PrevCursor != "prev-message" ||
		!response.Data.HasMore {
		t.Fatalf("pagination response = %#v", response.Data)
	}
}

func TestMessageHandlersKeepCandidateScope(t *testing.T) {
	client := &messageRegressionClient{}
	handler := &Handler{Gmsg: client}

	countRecorder := httptest.NewRecorder()
	handler.GetUnreadMessageCount(
		countRecorder,
		newCandidateHandlerRequest(http.MethodGet, "/api/messages/unread-count", "", "candidate-1", nil),
	)
	if countRecorder.Code != http.StatusOK {
		t.Fatalf("count status = %d; body=%q", countRecorder.Code, countRecorder.Body.String())
	}
	if client.countRequest.GetFilters().GetUserUlid() != "candidate-1" ||
		client.countRequest.GetFilters().GetStatus() != gmsgpb.MessageStatus_UNREAD ||
		client.countRequest.GetLimit() != 99 {
		t.Fatalf("count request = %#v", client.countRequest)
	}

	markRecorder := httptest.NewRecorder()
	handler.MarkMessagesRead(
		markRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/messages/read",
			`{"message_ids":[" message-1 ","","message-2"]}`,
			"candidate-1",
			nil,
		),
	)
	if markRecorder.Code != http.StatusOK {
		t.Fatalf("mark status = %d; body=%q", markRecorder.Code, markRecorder.Body.String())
	}
	if client.markRequest.GetUserUlid() != "candidate-1" ||
		!reflect.DeepEqual(client.markRequest.GetMessageIds(), []string{"message-1", "message-2"}) {
		t.Fatalf("mark request = %#v", client.markRequest)
	}

	deleteRecorder := httptest.NewRecorder()
	handler.DeleteMessage(
		deleteRecorder,
		newCandidateHandlerRequest(
			http.MethodDelete,
			"/api/messages",
			`{"message_ids":["message-3"]}`,
			"candidate-1",
			nil,
		),
	)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d; body=%q", deleteRecorder.Code, deleteRecorder.Body.String())
	}
	if client.deleteRequest.GetUserUlid() != "candidate-1" ||
		!reflect.DeepEqual(client.deleteRequest.GetMessageIds(), []string{"message-3"}) {
		t.Fatalf("delete request = %#v", client.deleteRequest)
	}

	getRecorder := httptest.NewRecorder()
	handler.GetMessage(
		getRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/messages/message-4",
			"",
			"candidate-1",
			map[string]string{"messageId": "message-4"},
		),
	)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("get status = %d; body=%q", getRecorder.Code, getRecorder.Body.String())
	}
	if client.getRequest.GetUserUlid() != "candidate-1" ||
		client.getRequest.GetMessageUlid() != "message-4" {
		t.Fatalf("get request = %#v", client.getRequest)
	}
}

func TestMessageHandlersRejectInvalidOperationsBeforeCallingService(t *testing.T) {
	client := &messageRegressionClient{}
	handler := &Handler{Gmsg: client}

	statusRecorder := httptest.NewRecorder()
	handler.ListMessages(
		statusRecorder,
		newCandidateHandlerRequest(
			http.MethodGet,
			"/api/messages?status=unsupported",
			"",
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, statusRecorder, http.StatusBadRequest, ErrInvalidRequest)
	if client.listRequest != nil {
		t.Fatal("ListMessages was called for unsupported status")
	}

	markRecorder := httptest.NewRecorder()
	handler.MarkMessagesRead(
		markRecorder,
		newCandidateHandlerRequest(
			http.MethodPost,
			"/api/messages/read",
			`{"message_ids":[" ",""]}`,
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, markRecorder, http.StatusBadRequest, ErrInvalidRequest)
	if client.markCalls != 0 {
		t.Fatalf("MarkAsRead calls = %d, want 0", client.markCalls)
	}

	deleteRecorder := httptest.NewRecorder()
	handler.DeleteMessage(
		deleteRecorder,
		newCandidateHandlerRequest(
			http.MethodDelete,
			"/api/messages",
			`{"message_ids":[]}`,
			"candidate-1",
			nil,
		),
	)
	assertHandlerAPIError(t, deleteRecorder, http.StatusBadRequest, ErrInvalidRequest)
	if client.deleteCalls != 0 {
		t.Fatalf("DeleteMessages calls = %d, want 0", client.deleteCalls)
	}
}
