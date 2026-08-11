package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gmbrpb "github.com/afnandelfin620-star/cftptest/cftp/gmbr"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type adminOpsPayReadStub struct {
	gpaypb.PayServiceClient
	err              error
	subscriptionList *gpaypb.ListSubscriptionsRequest
	webhookList      *gpaypb.ListWebhookEventsRequest
	webhookDetail    *gpaypb.GetWebhookEventDetailRequest
	orderItemsList   *gpaypb.ListOrderItemsRequest
}

func (s *adminOpsPayReadStub) ListSubscriptions(_ context.Context, request *gpaypb.ListSubscriptionsRequest, _ ...grpc.CallOption) (*gpaypb.ListSubscriptionsResponse, error) {
	s.subscriptionList = request
	return &gpaypb.ListSubscriptionsResponse{
		Subscriptions: []*gpaypb.SubscriptionSummary{{OrderUlid: "order-response", CustomerUlid: "customer-response"}},
		NextCursor:    "subscription-next",
		HasMore:       true,
	}, s.err
}

func (s *adminOpsPayReadStub) ListWebhookEvents(_ context.Context, request *gpaypb.ListWebhookEventsRequest, _ ...grpc.CallOption) (*gpaypb.ListWebhookEventsResponse, error) {
	s.webhookList = request
	return &gpaypb.ListWebhookEventsResponse{
		Events:     []*gpaypb.WebhookEventSummary{{EventId: "webhook-response"}},
		NextCursor: "webhook-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsPayReadStub) GetWebhookEventDetail(_ context.Context, request *gpaypb.GetWebhookEventDetailRequest, _ ...grpc.CallOption) (*gpaypb.GetWebhookEventDetailResponse, error) {
	s.webhookDetail = request
	return &gpaypb.GetWebhookEventDetailResponse{Event: &gpaypb.WebhookEventDetail{
		EventId:     request.GetEventId(),
		PayloadJson: `{"source":"regression"}`,
	}}, s.err
}

func (s *adminOpsPayReadStub) ListOrderItems(_ context.Context, request *gpaypb.ListOrderItemsRequest, _ ...grpc.CallOption) (*gpaypb.ListOrderItemsResponse, error) {
	s.orderItemsList = request
	return &gpaypb.ListOrderItemsResponse{Items: []*gpaypb.OrderItemSummary{{
		OrderUlid: request.GetOrderUlid(),
		ItemId:    "order-item-response",
	}}}, s.err
}

type adminOpsMallReadStub struct {
	mallpb.MallServiceClient
	err         error
	mailList    *mallpb.ListMailTasksRequest
	mailSummary *mallpb.GetMailTaskSummaryRequest
	mailDetail  *mallpb.GetMailTaskDetailRequest
	natsList    *mallpb.ListNatsMessagesRequest
	natsSummary *mallpb.GetNatsMessageSummaryRequest
	natsDetail  *mallpb.GetNatsMessageDetailRequest
}

func (s *adminOpsMallReadStub) ListMailTasks(_ context.Context, request *mallpb.ListMailTasksRequest, _ ...grpc.CallOption) (*mallpb.ListMailTasksResponse, error) {
	s.mailList = request
	return &mallpb.ListMailTasksResponse{
		Items:      []*mallpb.MailTaskSummary{{MailTaskUlid: "mall-mail-response", CandidateUlid: "candidate-response"}},
		NextCursor: "mall-mail-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsMallReadStub) GetMailTaskSummary(_ context.Context, request *mallpb.GetMailTaskSummaryRequest, _ ...grpc.CallOption) (*mallpb.GetMailTaskSummaryResponse, error) {
	s.mailSummary = request
	return &mallpb.GetMailTaskSummaryResponse{
		Found:   true,
		Summary: &mallpb.MailTaskSummary{MailTaskUlid: request.GetMailTaskUlid()},
	}, s.err
}

func (s *adminOpsMallReadStub) GetMailTaskDetail(_ context.Context, request *mallpb.GetMailTaskDetailRequest, _ ...grpc.CallOption) (*mallpb.GetMailTaskDetailResponse, error) {
	s.mailDetail = request
	return &mallpb.GetMailTaskDetailResponse{
		Found: true,
		Detail: &mallpb.MailTaskDetail{
			Summary:      &mallpb.MailTaskSummary{MailTaskUlid: request.GetMailTaskUlid()},
			TemplatePath: "mall/mail/regression",
		},
	}, s.err
}

func (s *adminOpsMallReadStub) ListNatsMessages(_ context.Context, request *mallpb.ListNatsMessagesRequest, _ ...grpc.CallOption) (*mallpb.ListNatsMessagesResponse, error) {
	s.natsList = request
	return &mallpb.ListNatsMessagesResponse{
		Items:      []*mallpb.NatsMessageSummary{{MessageUlid: "mall-nats-response"}},
		NextCursor: "mall-nats-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsMallReadStub) GetNatsMessageSummary(_ context.Context, request *mallpb.GetNatsMessageSummaryRequest, _ ...grpc.CallOption) (*mallpb.GetNatsMessageSummaryResponse, error) {
	s.natsSummary = request
	return &mallpb.GetNatsMessageSummaryResponse{
		Found:   true,
		Summary: &mallpb.NatsMessageSummary{MessageUlid: request.GetMessageUlid()},
	}, s.err
}

func (s *adminOpsMallReadStub) GetNatsMessageDetail(_ context.Context, request *mallpb.GetNatsMessageDetailRequest, _ ...grpc.CallOption) (*mallpb.GetNatsMessageDetailResponse, error) {
	s.natsDetail = request
	return &mallpb.GetNatsMessageDetailResponse{
		Found: true,
		Detail: &mallpb.NatsMessageDetail{
			Summary:        &mallpb.NatsMessageSummary{MessageUlid: request.GetMessageUlid()},
			MessagePayload: `{"source":"regression"}`,
		},
	}, s.err
}

type adminOpsProgReadStub struct {
	gprogpb.ProgServiceClient
	err          error
	mailList     *gprogpb.ListMailTasksReq
	mailDetail   *gprogpb.GetMailTaskDetailReq
	stageList    *gprogpb.ListStagesReq
	stageDetail  *gprogpb.GetStageDetailReq
	unitList     *gprogpb.ListCourseUnitsReq
	unitDetail   *gprogpb.GetCourseUnitDetailReq
	driverList   *gprogpb.ListDriverEventsReq
	driverDetail *gprogpb.GetDriverEventDetailReq
	natsList     *gprogpb.ListNatsMessagesReq
	natsDetail   *gprogpb.GetNatsMessageDetailReq
}

func (s *adminOpsProgReadStub) ListMailTasks(_ context.Context, request *gprogpb.ListMailTasksReq, _ ...grpc.CallOption) (*gprogpb.ListMailTasksRsp, error) {
	s.mailList = request
	return &gprogpb.ListMailTasksRsp{
		Tasks:      []*gprogpb.MailTaskSummary{{MailTaskUlid: "prog-mail-response", CandidateUlid: "candidate-response"}},
		NextCursor: "prog-mail-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsProgReadStub) GetMailTaskDetail(_ context.Context, request *gprogpb.GetMailTaskDetailReq, _ ...grpc.CallOption) (*gprogpb.GetMailTaskDetailRsp, error) {
	s.mailDetail = request
	return &gprogpb.GetMailTaskDetailRsp{
		Summary:      &gprogpb.MailTaskSummary{MailTaskUlid: request.GetMailTaskUlid()},
		TemplatePath: "prog/mail/regression",
	}, s.err
}

func (s *adminOpsProgReadStub) ListStages(_ context.Context, request *gprogpb.ListStagesReq, _ ...grpc.CallOption) (*gprogpb.ListStagesRsp, error) {
	s.stageList = request
	return &gprogpb.ListStagesRsp{
		Stages:     []*gprogpb.StageSummary{{StageUlid: "stage-response"}},
		NextCursor: "stage-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsProgReadStub) GetStageDetail(_ context.Context, request *gprogpb.GetStageDetailReq, _ ...grpc.CallOption) (*gprogpb.GetStageDetailRsp, error) {
	s.stageDetail = request
	return &gprogpb.GetStageDetailRsp{Summary: &gprogpb.StageSummary{StageUlid: request.GetStageUlid()}}, s.err
}

func (s *adminOpsProgReadStub) ListCourseUnits(_ context.Context, request *gprogpb.ListCourseUnitsReq, _ ...grpc.CallOption) (*gprogpb.ListCourseUnitsRsp, error) {
	s.unitList = request
	return &gprogpb.ListCourseUnitsRsp{
		CourseUnits: []*gprogpb.CourseUnitSummary{{CourseUnitUlid: "course-unit-response"}},
		NextCursor:  "course-unit-next",
		HasMore:     true,
	}, s.err
}

func (s *adminOpsProgReadStub) GetCourseUnitDetail(_ context.Context, request *gprogpb.GetCourseUnitDetailReq, _ ...grpc.CallOption) (*gprogpb.GetCourseUnitDetailRsp, error) {
	s.unitDetail = request
	return &gprogpb.GetCourseUnitDetailRsp{CourseUnitUlid: request.GetCourseUnitUlid()}, s.err
}

func (s *adminOpsProgReadStub) ListDriverEvents(_ context.Context, request *gprogpb.ListDriverEventsReq, _ ...grpc.CallOption) (*gprogpb.ListDriverEventsRsp, error) {
	s.driverList = request
	return &gprogpb.ListDriverEventsRsp{
		Items:      []*gprogpb.DriverEventSummary{{EventUlid: "driver-response"}},
		NextCursor: "driver-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsProgReadStub) GetDriverEventDetail(_ context.Context, request *gprogpb.GetDriverEventDetailReq, _ ...grpc.CallOption) (*gprogpb.GetDriverEventDetailRsp, error) {
	s.driverDetail = request
	return &gprogpb.GetDriverEventDetailRsp{
		Summary:     &gprogpb.DriverEventSummary{EventUlid: request.GetEventUlid()},
		PayloadJson: `{"source":"regression"}`,
	}, s.err
}

func (s *adminOpsProgReadStub) ListNatsMessages(_ context.Context, request *gprogpb.ListNatsMessagesReq, _ ...grpc.CallOption) (*gprogpb.ListNatsMessagesRsp, error) {
	s.natsList = request
	return &gprogpb.ListNatsMessagesRsp{
		Items:      []*gprogpb.NatsMessageSummary{{MessageUlid: "prog-nats-response"}},
		NextCursor: "prog-nats-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsProgReadStub) GetNatsMessageDetail(_ context.Context, request *gprogpb.GetNatsMessageDetailReq, _ ...grpc.CallOption) (*gprogpb.GetNatsMessageDetailRsp, error) {
	s.natsDetail = request
	return &gprogpb.GetNatsMessageDetailRsp{
		Summary:        &gprogpb.NatsMessageSummary{MessageUlid: request.GetMessageUlid()},
		MessagePayload: `{"source":"regression"}`,
	}, s.err
}

type adminOpsExamReadStub struct {
	gexampb.GExamServiceClient
	err            error
	auditList      *gexampb.ListAuditMessagesRequest
	auditDetail    *gexampb.GetAuditMessageDetailRequest
	transitionList *gexampb.ListExamStatusTransitionsRequest
	reminderList   *gexampb.ListReminderMailsRequest
	reminderDetail *gexampb.GetReminderMailDetailRequest
}

func (s *adminOpsExamReadStub) ListAuditMessages(_ context.Context, request *gexampb.ListAuditMessagesRequest, _ ...grpc.CallOption) (*gexampb.ListAuditMessagesResponse, error) {
	s.auditList = request
	return &gexampb.ListAuditMessagesResponse{
		AuditMessages: []*gexampb.AuditMessageSummary{{MessageUlid: "exam-audit-response"}},
		NextCursor:    "exam-audit-next",
		HasMore:       true,
	}, s.err
}

func (s *adminOpsExamReadStub) GetAuditMessageDetail(_ context.Context, request *gexampb.GetAuditMessageDetailRequest, _ ...grpc.CallOption) (*gexampb.AuditMessageDetail, error) {
	s.auditDetail = request
	return &gexampb.AuditMessageDetail{MessageUlid: request.GetMessageUlid(), PayloadJson: `{"source":"regression"}`}, s.err
}

func (s *adminOpsExamReadStub) ListExamStatusTransitions(_ context.Context, request *gexampb.ListExamStatusTransitionsRequest, _ ...grpc.CallOption) (*gexampb.ListExamStatusTransitionsResponse, error) {
	s.transitionList = request
	return &gexampb.ListExamStatusTransitionsResponse{
		Transitions: []*gexampb.ExamStatusTransition{{MsgFp: "exam-transition-response"}},
		NextCursor:  "exam-transition-next",
		HasMore:     true,
	}, s.err
}

func (s *adminOpsExamReadStub) ListReminderMails(_ context.Context, request *gexampb.ListReminderMailsRequest, _ ...grpc.CallOption) (*gexampb.ListReminderMailsResponse, error) {
	s.reminderList = request
	return &gexampb.ListReminderMailsResponse{
		Mails:      []*gexampb.ReminderMailItem{{MailUlid: "exam-reminder-response"}},
		NextCursor: "exam-reminder-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsExamReadStub) GetReminderMailDetail(_ context.Context, request *gexampb.GetReminderMailDetailRequest, _ ...grpc.CallOption) (*gexampb.GetReminderMailDetailResponse, error) {
	s.reminderDetail = request
	return &gexampb.GetReminderMailDetailResponse{
		Mail:        &gexampb.ReminderMailItem{MailUlid: request.GetMailUlid()},
		PayloadJson: `{"source":"regression"}`,
	}, s.err
}

type adminOpsMembershipReadStub struct {
	gmbrpb.GmbrServiceClient
	err        error
	mailList   *gmbrpb.ListMembershipMailsRequest
	mailDetail *gmbrpb.GetMembershipMailDetailRequest
}

func (s *adminOpsMembershipReadStub) ListMembershipMails(_ context.Context, request *gmbrpb.ListMembershipMailsRequest, _ ...grpc.CallOption) (*gmbrpb.ListMembershipMailsResponse, error) {
	s.mailList = request
	return &gmbrpb.ListMembershipMailsResponse{
		Mails:      []*gmbrpb.MembershipMailItem{{MailUlid: "membership-mail-response", CandidateUlid: "candidate-response"}},
		NextCursor: "membership-mail-next",
		HasMore:    true,
	}, s.err
}

func (s *adminOpsMembershipReadStub) GetMembershipMailDetail(_ context.Context, request *gmbrpb.GetMembershipMailDetailRequest, _ ...grpc.CallOption) (*gmbrpb.GetMembershipMailDetailResponse, error) {
	s.mailDetail = request
	return &gmbrpb.GetMembershipMailDetailResponse{
		Mail:        &gmbrpb.MembershipMailItem{MailUlid: request.GetMailUlid()},
		PayloadJson: `{"source":"regression"}`,
	}, s.err
}

func assertAdminOpsReadOK(t *testing.T, handlerFunc http.HandlerFunc, request *http.Request) map[string]interface{} {
	t.Helper()
	recorder := httptest.NewRecorder()
	handlerFunc(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	var payload struct {
		Code      int                    `json:"code"`
		ErrorCode ErrorCode              `json:"error_code"`
		Data      map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, recorder.Body.String())
	}
	if payload.Code != http.StatusOK || payload.ErrorCode != "OK" || payload.Data == nil {
		t.Fatalf("response envelope = %+v", payload)
	}
	return payload.Data
}

func assertAdminOpsJSONPath(t *testing.T, root map[string]interface{}, want string, path ...interface{}) {
	t.Helper()
	var current interface{} = root
	for _, segment := range path {
		switch key := segment.(type) {
		case string:
			object, ok := current.(map[string]interface{})
			if !ok {
				t.Fatalf("JSON path %v reached %T, want object", path, current)
			}
			value, ok := object[key]
			if !ok {
				t.Fatalf("JSON path %v is missing key %q in %+v", path, key, object)
			}
			current = value
		case int:
			items, ok := current.([]interface{})
			if !ok || key < 0 || key >= len(items) {
				t.Fatalf("JSON path %v cannot read index %d from %T", path, key, current)
			}
			current = items[key]
		default:
			t.Fatalf("unsupported JSON path segment %T", segment)
		}
	}
	if got, ok := current.(string); !ok || got != want {
		t.Fatalf("JSON path %v = %#v, want %q", path, current, want)
	}
}

func TestAdminOpsReadListsForwardFiltersAndPagination(t *testing.T) {
	pay := &adminOpsPayReadStub{}
	mall := &adminOpsMallReadStub{}
	prog := &adminOpsProgReadStub{}
	exam := &adminOpsExamReadStub{}
	membership := &adminOpsMembershipReadStub{}
	handler := &Handler{Gpay: pay, Mall: mall, Gprog: prog, Gexam: exam, Gmbr: membership}

	payWebhookData := assertAdminOpsReadOK(t, handler.ListPayWebhookEvents, httptest.NewRequest(http.MethodGet, "/api/pay/webhook-events?event_type=invoice.paid&processed_status=PROCESSED&start_time=10&end_time=20&cursor=pay-cursor&page_size=17&sort=1", nil))
	assertAdminOpsJSONPath(t, payWebhookData, "webhook-response", "events", 0, "event_id")
	assertAdminOpsJSONPath(t, payWebhookData, "webhook-next", "next_cursor")
	if request := pay.webhookList; request.GetFilters().GetEventType() != "invoice.paid" || request.GetFilters().GetProcessedStatus() != "PROCESSED" || request.GetFilters().GetStartTime() != 10 || request.GetFilters().GetEndTime() != 20 || request.GetCursor() != "pay-cursor" || request.GetPageSize() != 17 || request.GetSortOrder() != gpaypb.SortOrder(1) {
		t.Fatalf("pay webhook list request = %+v", request)
	}

	payOrderItemData := assertAdminOpsReadOK(t, handler.ListPayOrderItems, httptest.NewRequest(http.MethodGet, "/api/pay/order-items?order_ulid=order-regression", nil))
	assertAdminOpsJSONPath(t, payOrderItemData, "order-item-response", "items", 0, "item_id")
	if pay.orderItemsList.GetOrderUlid() != "order-regression" {
		t.Fatalf("pay order items request = %+v", pay.orderItemsList)
	}

	mallMailData := assertAdminOpsReadOK(t, handler.ListMallMailTasks, httptest.NewRequest(http.MethodGet, "/api/mall/mail-tasks?candidate_ulid=candidate-regression&order_ulid=order-regression&task_status=FAILED&mail_type=PAYMENT&cursor=mail-cursor&page_size=18&sort=1", nil))
	assertAdminOpsJSONPath(t, mallMailData, "mall-mail-response", "items", 0, "mail_task_ulid")
	assertAdminOpsJSONPath(t, mallMailData, "mall-mail-next", "next_cursor")
	if request := mall.mailList; request.GetFilters().GetCandidateUlid() != "candidate-regression" || request.GetFilters().GetOrderUlid() != "order-regression" || request.GetFilters().GetTaskStatus() != "FAILED" || request.GetFilters().GetMailType() != "PAYMENT" || request.GetCursor() != "mail-cursor" || request.GetPageSize() != 18 || request.GetSortOrder() != mallpb.SortOrder(1) {
		t.Fatalf("mall mail list request = %+v", request)
	}

	mallNATSData := assertAdminOpsReadOK(t, handler.ListMallNatsMessages, httptest.NewRequest(http.MethodGet, "/api/mall/nats-messages?receive_status=PROCESSED&source_service=gpay&subject=orders&message_type=status&cursor=mall-nats&page_size=19&sort=1", nil))
	assertAdminOpsJSONPath(t, mallNATSData, "mall-nats-response", "items", 0, "message_ulid")
	assertAdminOpsJSONPath(t, mallNATSData, "mall-nats-next", "next_cursor")
	if request := mall.natsList; request.GetFilters().GetReceiveStatus() != "PROCESSED" || request.GetFilters().GetSourceService() != "gpay" || request.GetFilters().GetSubject() != "orders" || request.GetFilters().GetMessageType() != "status" || request.GetCursor() != "mall-nats" || request.GetPageSize() != 19 || request.GetSortOrder() != mallpb.SortOrder(1) {
		t.Fatalf("mall NATS list request = %+v", request)
	}

	membershipMailData := assertAdminOpsReadOK(t, handler.ListMembershipMails, httptest.NewRequest(http.MethodGet, "/api/memberships/mails?candidate_ulid=candidate-regression&task_status=SENT&notification_type=membership_activated&cursor=mbr-mail&page_size=16&sort=1", nil))
	assertAdminOpsJSONPath(t, membershipMailData, "membership-mail-response", "mails", 0, "mail_ulid")
	assertAdminOpsJSONPath(t, membershipMailData, "membership-mail-next", "next_cursor")
	if request := membership.mailList; request.GetFilters().GetCandidateUlid() != "candidate-regression" || request.GetFilters().GetTaskStatus() != "SENT" || request.GetFilters().GetNotificationType() != "membership_activated" || request.GetCursor() != "mbr-mail" || request.GetPageSize() != 16 || request.GetSortOrder() != gmbrpb.SortOrder(1) {
		t.Fatalf("membership mail list request = %+v", request)
	}

	progMailData := assertAdminOpsReadOK(t, handler.ListProgMailTasks, httptest.NewRequest(http.MethodGet, "/api/prog/mail-tasks?candidate_ulid=candidate-regression&pipeline_ulid=pipeline-regression&cursor=prog-mail&page_size=15&sort=1", nil))
	assertAdminOpsJSONPath(t, progMailData, "prog-mail-response", "tasks", 0, "mail_task_ulid")
	assertAdminOpsJSONPath(t, progMailData, "prog-mail-next", "next_cursor")
	if request := prog.mailList; request.GetFilters().GetCandidateUlid() != "candidate-regression" || request.GetFilters().GetPipelineUlid() != "pipeline-regression" || request.GetCursor() != "prog-mail" || request.GetPageSize() != 15 || request.GetSortOrder() != gprogpb.SortOrder(1) {
		t.Fatalf("prog mail list request = %+v", request)
	}

	progStageData := assertAdminOpsReadOK(t, handler.ListProgStages, httptest.NewRequest(http.MethodGet, "/api/prog/stages?pipeline_ulid=pipeline-regression&cursor=stages&page_size=14&sort=1", nil))
	assertAdminOpsJSONPath(t, progStageData, "stage-response", "stages", 0, "stage_ulid")
	assertAdminOpsJSONPath(t, progStageData, "stage-next", "next_cursor")
	if request := prog.stageList; request.GetFilters().GetPipelineUlid() != "pipeline-regression" || request.GetCursor() != "stages" || request.GetPageSize() != 14 || request.GetSortOrder() != gprogpb.SortOrder(1) {
		t.Fatalf("prog stage list request = %+v", request)
	}

	progUnitData := assertAdminOpsReadOK(t, handler.ListProgCourseUnits, httptest.NewRequest(http.MethodGet, "/api/prog/course-units?pipeline_ulid=pipeline-regression&stage_ulid=stage-regression&status=completed&cursor=units&page_size=13&sort=1", nil))
	assertAdminOpsJSONPath(t, progUnitData, "course-unit-response", "course_units", 0, "course_unit_ulid")
	assertAdminOpsJSONPath(t, progUnitData, "course-unit-next", "next_cursor")
	if request := prog.unitList; request.GetFilters().GetPipelineUlid() != "pipeline-regression" || request.GetFilters().GetStageUlid() != "stage-regression" || request.GetFilters().GetStatus() != gprogpb.CourseUnitStatus_COURSE_UNIT_STATUS_COMPLETED || request.GetCursor() != "units" || request.GetPageSize() != 13 || request.GetSortOrder() != gprogpb.SortOrder(1) {
		t.Fatalf("prog course unit list request = %+v", request)
	}

	progDriverData := assertAdminOpsReadOK(t, handler.ListProgDriverEvents, httptest.NewRequest(http.MethodGet, "/api/prog/driver-events?entity_type=PIPELINE&entity_ulid=pipeline-regression&event_status=PROCESSED&event_type=PIPELINE_NEXT_STAGE&cursor=driver&page_size=12&sort=1", nil))
	assertAdminOpsJSONPath(t, progDriverData, "driver-response", "items", 0, "event_ulid")
	assertAdminOpsJSONPath(t, progDriverData, "driver-next", "next_cursor")
	if request := prog.driverList; request.GetFilters().GetEntityType() != "PIPELINE" || request.GetFilters().GetEntityUlid() != "pipeline-regression" || request.GetFilters().GetEventStatus() != "PROCESSED" || request.GetFilters().GetEventType() != "PIPELINE_NEXT_STAGE" || request.GetCursor() != "driver" || request.GetPageSize() != 12 || request.GetSortOrder() != gprogpb.SortOrder(1) {
		t.Fatalf("prog driver list request = %+v", request)
	}

	progNATSData := assertAdminOpsReadOK(t, handler.ListProgNatsMessages, httptest.NewRequest(http.MethodGet, "/api/prog/nats-messages?receive_status=PROCESSED&source_service=gmall&cursor=prog-nats&page_size=11&sort=1", nil))
	assertAdminOpsJSONPath(t, progNATSData, "prog-nats-response", "items", 0, "message_ulid")
	assertAdminOpsJSONPath(t, progNATSData, "prog-nats-next", "next_cursor")
	if request := prog.natsList; request.GetFilters().GetReceiveStatus() != "PROCESSED" || request.GetFilters().GetSourceService() != "gmall" || request.GetCursor() != "prog-nats" || request.GetPageSize() != 11 || request.GetSortOrder() != gprogpb.SortOrder(1) {
		t.Fatalf("prog NATS list request = %+v", request)
	}

	examAuditData := assertAdminOpsReadOK(t, handler.ListExamAuditMessages, httptest.NewRequest(http.MethodGet, "/api/exam-ops/audit-messages?processed_status=PROCESSED&event_type=RESULT&start_time=start&end_time=end&cursor=exam-audit&page_size=10&sort=1", nil))
	assertAdminOpsJSONPath(t, examAuditData, "exam-audit-response", "audit_messages", 0, "message_ulid")
	assertAdminOpsJSONPath(t, examAuditData, "exam-audit-next", "next_cursor")
	if request := exam.auditList; request.GetFilters().GetProcessedStatus() != "PROCESSED" || request.GetFilters().GetEventType() != "RESULT" || request.GetFilters().GetStartTime() != "start" || request.GetFilters().GetEndTime() != "end" || request.GetCursor() != "exam-audit" || request.GetPageSize() != 10 || request.GetSortOrder() != gexampb.SortOrder(1) {
		t.Fatalf("exam audit list request = %+v", request)
	}

	examTransitionData := assertAdminOpsReadOK(t, handler.ListExamStatusTransitions, httptest.NewRequest(http.MethodGet, "/api/exam-ops/status-transitions?exam_ulid=exam-regression&status_type=EXAM&cursor=transitions&page_size=9&sort=1", nil))
	assertAdminOpsJSONPath(t, examTransitionData, "exam-transition-response", "transitions", 0, "msg_fp")
	assertAdminOpsJSONPath(t, examTransitionData, "exam-transition-next", "next_cursor")
	if request := exam.transitionList; request.GetFilters().GetExamUlid() != "exam-regression" || request.GetFilters().GetStatusType() != "EXAM" || request.GetCursor() != "transitions" || request.GetPageSize() != 9 || request.GetSortOrder() != gexampb.SortOrder(1) {
		t.Fatalf("exam transition list request = %+v", request)
	}

	examReminderData := assertAdminOpsReadOK(t, handler.ListExamReminderMails, httptest.NewRequest(http.MethodGet, "/api/exam-ops/reminder-mails?exam_ulid=exam-regression&task_status=SENT&delivery_status=DELIVERED&candidate_email=candidate@example.test&reminder_type=ONE_HOUR&cursor=reminders&page_size=8&sort=1", nil))
	assertAdminOpsJSONPath(t, examReminderData, "exam-reminder-response", "mails", 0, "mail_ulid")
	assertAdminOpsJSONPath(t, examReminderData, "exam-reminder-next", "next_cursor")
	if request := exam.reminderList; request.GetFilters().GetExamUlid() != "exam-regression" || request.GetFilters().GetTaskStatus() != "SENT" || request.GetFilters().GetDeliveryStatus() != "DELIVERED" || request.GetFilters().GetCandidateEmail() != "candidate@example.test" || request.GetFilters().GetReminderType() != "ONE_HOUR" || request.GetCursor() != "reminders" || request.GetPageSize() != 8 || request.GetSortOrder() != gexampb.SortOrder(1) {
		t.Fatalf("exam reminder list request = %+v", request)
	}
}

func TestAdminOpsReadDetailsForwardPathIDs(t *testing.T) {
	pay := &adminOpsPayReadStub{}
	mall := &adminOpsMallReadStub{}
	prog := &adminOpsProgReadStub{}
	exam := &adminOpsExamReadStub{}
	membership := &adminOpsMembershipReadStub{}
	handler := &Handler{Gpay: pay, Mall: mall, Gprog: prog, Gexam: exam, Gmbr: membership}

	tests := []struct {
		name       string
		handler    http.HandlerFunc
		param      string
		value      string
		jsonPath   []interface{}
		capturedID func() string
	}{
		{name: "pay webhook", handler: handler.GetPayWebhookEventDetail, param: "event_id", value: "evt-regression", jsonPath: []interface{}{"event", "event_id"}, capturedID: func() string { return pay.webhookDetail.GetEventId() }},
		{name: "mall mail summary", handler: handler.GetMallMailTaskSummary, param: "mail_task_ulid", value: "mall-mail-regression", jsonPath: []interface{}{"summary", "mail_task_ulid"}, capturedID: func() string { return mall.mailSummary.GetMailTaskUlid() }},
		{name: "mall mail detail", handler: handler.GetMallMailTaskDetail, param: "mail_task_ulid", value: "mall-mail-regression", jsonPath: []interface{}{"detail", "summary", "mail_task_ulid"}, capturedID: func() string { return mall.mailDetail.GetMailTaskUlid() }},
		{name: "mall NATS summary", handler: handler.GetMallNatsMessageSummary, param: "message_ulid", value: "mall-nats-regression", jsonPath: []interface{}{"summary", "message_ulid"}, capturedID: func() string { return mall.natsSummary.GetMessageUlid() }},
		{name: "mall NATS detail", handler: handler.GetMallNatsMessageDetail, param: "message_ulid", value: "mall-nats-regression", jsonPath: []interface{}{"detail", "summary", "message_ulid"}, capturedID: func() string { return mall.natsDetail.GetMessageUlid() }},
		{name: "membership mail", handler: handler.GetMembershipMailDetail, param: "mail_ulid", value: "membership-mail-regression", jsonPath: []interface{}{"mail", "mail_ulid"}, capturedID: func() string { return membership.mailDetail.GetMailUlid() }},
		{name: "prog mail", handler: handler.GetProgMailTaskDetail, param: "mail_task_ulid", value: "prog-mail-regression", jsonPath: []interface{}{"summary", "mail_task_ulid"}, capturedID: func() string { return prog.mailDetail.GetMailTaskUlid() }},
		{name: "prog stage", handler: handler.GetProgStageDetail, param: "stage_ulid", value: "stage-regression", jsonPath: []interface{}{"summary", "stage_ulid"}, capturedID: func() string { return prog.stageDetail.GetStageUlid() }},
		{name: "prog course unit", handler: handler.GetProgCourseUnitDetail, param: "course_unit_ulid", value: "unit-regression", jsonPath: []interface{}{"course_unit_ulid"}, capturedID: func() string { return prog.unitDetail.GetCourseUnitUlid() }},
		{name: "prog driver", handler: handler.GetProgDriverEventDetail, param: "event_ulid", value: "driver-regression", jsonPath: []interface{}{"summary", "event_ulid"}, capturedID: func() string { return prog.driverDetail.GetEventUlid() }},
		{name: "prog NATS", handler: handler.GetProgNatsMessageDetail, param: "message_ulid", value: "prog-nats-regression", jsonPath: []interface{}{"summary", "message_ulid"}, capturedID: func() string { return prog.natsDetail.GetMessageUlid() }},
		{name: "exam audit", handler: handler.GetExamAuditMessageDetail, param: "message_ulid", value: "exam-audit-regression", jsonPath: []interface{}{"message_ulid"}, capturedID: func() string { return exam.auditDetail.GetMessageUlid() }},
		{name: "exam reminder", handler: handler.GetExamReminderMailDetail, param: "mail_ulid", value: "exam-mail-regression", jsonPath: []interface{}{"mail", "mail_ulid"}, capturedID: func() string { return exam.reminderDetail.GetMailUlid() }},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			request := requestWithURLParam(http.MethodGet, "/detail/"+testCase.value, testCase.param, testCase.value)
			data := assertAdminOpsReadOK(t, testCase.handler, request)
			if got := testCase.capturedID(); got != testCase.value {
				t.Fatalf("forwarded ID = %q, want %q", got, testCase.value)
			}
			assertAdminOpsJSONPath(t, data, testCase.value, testCase.jsonPath...)
		})
	}
}

func TestAdminOpsReadHandlersMapDownstreamUnavailable(t *testing.T) {
	downstreamErr := status.Error(codes.Unavailable, "downstream unavailable")
	pay := &adminOpsPayReadStub{err: downstreamErr}
	mall := &adminOpsMallReadStub{err: downstreamErr}
	prog := &adminOpsProgReadStub{err: downstreamErr}
	exam := &adminOpsExamReadStub{err: downstreamErr}
	membership := &adminOpsMembershipReadStub{err: downstreamErr}
	handler := &Handler{Gpay: pay, Mall: mall, Gprog: prog, Gexam: exam, Gmbr: membership}

	tests := []struct {
		name    string
		handler http.HandlerFunc
		request func() *http.Request
	}{
		{name: "pay subscription list", handler: handler.ListPaySubscriptions, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/pay/subscriptions", nil) }},
		{name: "pay webhook list", handler: handler.ListPayWebhookEvents, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/pay/webhook-events", nil) }},
		{name: "pay webhook detail", handler: handler.GetPayWebhookEventDetail, request: func() *http.Request { return requestWithURLParam(http.MethodGet, "/detail/evt", "event_id", "evt") }},
		{name: "pay order items", handler: handler.ListPayOrderItems, request: func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/api/pay/order-items?order_ulid=order-regression", nil)
		}},
		{name: "mall mail list", handler: handler.ListMallMailTasks, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/mall/mail-tasks", nil) }},
		{name: "mall mail summary", handler: handler.GetMallMailTaskSummary, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/summary/mail", "mail_task_ulid", "mail")
		}},
		{name: "mall mail detail", handler: handler.GetMallMailTaskDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/mail", "mail_task_ulid", "mail")
		}},
		{name: "mall NATS list", handler: handler.ListMallNatsMessages, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/mall/nats-messages", nil) }},
		{name: "mall NATS summary", handler: handler.GetMallNatsMessageSummary, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/summary/nats", "message_ulid", "nats")
		}},
		{name: "mall NATS detail", handler: handler.GetMallNatsMessageDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/nats", "message_ulid", "nats")
		}},
		{name: "membership mail list", handler: handler.ListMembershipMails, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/memberships/mails", nil) }},
		{name: "membership mail detail", handler: handler.GetMembershipMailDetail, request: func() *http.Request { return requestWithURLParam(http.MethodGet, "/detail/mail", "mail_ulid", "mail") }},
		{name: "prog mail list", handler: handler.ListProgMailTasks, request: func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/api/prog/mail-tasks?candidate_ulid=candidate-regression", nil)
		}},
		{name: "prog mail detail", handler: handler.GetProgMailTaskDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/mail", "mail_task_ulid", "mail")
		}},
		{name: "prog stage list", handler: handler.ListProgStages, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/prog/stages", nil) }},
		{name: "prog stage detail", handler: handler.GetProgStageDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/stage", "stage_ulid", "stage")
		}},
		{name: "prog course unit list", handler: handler.ListProgCourseUnits, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/prog/course-units", nil) }},
		{name: "prog course unit detail", handler: handler.GetProgCourseUnitDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/unit", "course_unit_ulid", "unit")
		}},
		{name: "prog driver list", handler: handler.ListProgDriverEvents, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/prog/driver-events", nil) }},
		{name: "prog driver detail", handler: handler.GetProgDriverEventDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/driver", "event_ulid", "driver")
		}},
		{name: "prog NATS list", handler: handler.ListProgNatsMessages, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/prog/nats-messages", nil) }},
		{name: "prog NATS detail", handler: handler.GetProgNatsMessageDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/nats", "message_ulid", "nats")
		}},
		{name: "exam audit list", handler: handler.ListExamAuditMessages, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/exam-ops/audit-messages", nil) }},
		{name: "exam audit detail", handler: handler.GetExamAuditMessageDetail, request: func() *http.Request {
			return requestWithURLParam(http.MethodGet, "/detail/audit", "message_ulid", "audit")
		}},
		{name: "exam transition list", handler: handler.ListExamStatusTransitions, request: func() *http.Request {
			return httptest.NewRequest(http.MethodGet, "/api/exam-ops/status-transitions", nil)
		}},
		{name: "exam reminder list", handler: handler.ListExamReminderMails, request: func() *http.Request { return httptest.NewRequest(http.MethodGet, "/api/exam-ops/reminder-mails", nil) }},
		{name: "exam reminder detail", handler: handler.GetExamReminderMailDetail, request: func() *http.Request { return requestWithURLParam(http.MethodGet, "/detail/mail", "mail_ulid", "mail") }},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			testCase.handler(recorder, testCase.request())
			if recorder.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, want %d; body=%s", recorder.Code, http.StatusServiceUnavailable, recorder.Body.String())
			}
			var payload struct {
				Code      int       `json:"code"`
				ErrorCode ErrorCode `json:"error_code"`
				Message   string    `json:"message"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
				t.Fatalf("decode error response: %v; body=%s", err, recorder.Body.String())
			}
			if payload.Code != http.StatusServiceUnavailable || payload.ErrorCode != ErrServiceUnavailable || payload.Message != http.StatusText(http.StatusServiceUnavailable) {
				t.Fatalf("error response = %+v", payload)
			}
		})
	}
}
