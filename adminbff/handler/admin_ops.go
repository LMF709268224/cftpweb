package handler

import (
	"net/http"
	"strconv"
	"strings"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	gpaypb "github.com/afnandelfin620-star/cftptest/cftp/gpay"
	gprogpb "github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

type retryMailTaskBody struct {
	UseNewEmail                bool   `json:"use_new_email"`
	OverrideRecipientEmail     string `json:"override_recipient_email"`
	OverrideTemplateParamsJson string `json:"override_template_params_json"`
}

func queryText(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

func queryLimit(r *http.Request) int32 {
	limit := parsePositiveIntQuery(r, "limit", 20)
	if limit > 200 {
		limit = 200
	}
	return int32(limit)
}

func queryOffset(r *http.Request) int32 {
	return int32(parseUint32Query(r, "offset"))
}

func queryPage(r *http.Request) uint32 {
	return uint32(parsePositiveIntQuery(r, "page", 1))
}

func queryPageSize(r *http.Request) uint32 {
	pageSize := parsePositiveIntQuery(r, "page_size", 20)
	if pageSize > 200 {
		pageSize = 200
	}
	return uint32(pageSize)
}

func queryInt64(r *http.Request, key string) int64 {
	value := queryText(r, key)
	if value == "" {
		return 0
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func optionalQueryString(r *http.Request, key string) *string {
	return optionalString(queryText(r, key))
}

func adminActorID(r *http.Request) string {
	if adminID := strings.TrimSpace(AdminID(r)); adminID != "" {
		return adminID
	}
	return "admin"
}

func parsePayOrderStatus(r *http.Request) gpaypb.OrderStatus {
	value := queryText(r, "status")
	if value == "" {
		return gpaypb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
	if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
		return gpaypb.OrderStatus(parsed)
	}
	key := strings.ToUpper(value)
	if !strings.HasPrefix(key, "ORDER_STATUS_") {
		key = "ORDER_STATUS_" + key
	}
	if enumValue, ok := gpaypb.OrderStatus_value[key]; ok {
		return gpaypb.OrderStatus(enumValue)
	}
	return gpaypb.OrderStatus_ORDER_STATUS_UNSPECIFIED
}

func parseCourseUnitStatus(r *http.Request) gprogpb.CourseUnitStatus {
	value := queryText(r, "status")
	if value == "" {
		return gprogpb.CourseUnitStatus_COURSE_UNIT_STATUS_UNSPECIFIED
	}
	if parsed, err := strconv.ParseInt(value, 10, 32); err == nil {
		return gprogpb.CourseUnitStatus(parsed)
	}
	key := strings.ToUpper(value)
	if !strings.HasPrefix(key, "COURSE_UNIT_STATUS_") {
		key = "COURSE_UNIT_STATUS_" + key
	}
	if enumValue, ok := gprogpb.CourseUnitStatus_value[key]; ok {
		return gprogpb.CourseUnitStatus(enumValue)
	}
	return gprogpb.CourseUnitStatus_COURSE_UNIT_STATUS_UNSPECIFIED
}

func readRetryMailTaskBody(w http.ResponseWriter, r *http.Request) (retryMailTaskBody, bool) {
	var body retryMailTaskBody
	if r.Body == nil || r.ContentLength == 0 {
		return body, true
	}
	if err := ReadJSON(r, &body); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid body")
		return body, false
	}
	return body, true
}

func (h *Handler) ListPaySubscriptions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gpay.ListSubscriptions(r.Context(), &gpaypb.ListSubscriptionsRequest{
		CustomerUlid: queryText(r, "customer_ulid"),
		Status:       parsePayOrderStatus(r),
		Page:         queryPage(r),
		PageSize:     queryPageSize(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListPayWebhookEvents(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gpay.ListWebhookEvents(r.Context(), &gpaypb.ListWebhookEventsRequest{
		EventType:       queryText(r, "event_type"),
		ProcessedStatus: queryText(r, "processed_status"),
		StartTime:       queryInt64(r, "start_time"),
		EndTime:         queryInt64(r, "end_time"),
		Page:            queryPage(r),
		PageSize:        queryPageSize(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetPayWebhookEventDetail(w http.ResponseWriter, r *http.Request) {
	eventID, ok := requiredURLParam(w, r, "event_id")
	if !ok {
		return
	}
	resp, err := h.Gpay.GetWebhookEventDetail(r.Context(), &gpaypb.GetWebhookEventDetailRequest{EventId: eventID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListPayOrderItems(w http.ResponseWriter, r *http.Request) {
	orderULID := queryText(r, "order_ulid")
	if !requireRequestField(w, orderULID, "order_ulid") {
		return
	}
	resp, err := h.Gpay.ListOrderItems(r.Context(), &gpaypb.ListOrderItemsRequest{OrderUlid: orderULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListMallMailTasks(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Mall.ListMailTasks(r.Context(), &mallpb.ListMailTasksRequest{
		CandidateUlid: queryText(r, "candidate_ulid"),
		OrderUlid:     queryText(r, "order_ulid"),
		TaskStatus:    queryText(r, "task_status"),
		MailType:      queryText(r, "mail_type"),
		Limit:         queryLimit(r),
		Offset:        queryOffset(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMallMailTaskSummary(w http.ResponseWriter, r *http.Request) {
	mailTaskULID, ok := requiredURLParam(w, r, "mail_task_ulid")
	if !ok {
		return
	}
	resp, err := h.Mall.GetMailTaskSummary(r.Context(), &mallpb.GetMailTaskSummaryRequest{MailTaskUlid: mailTaskULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMallMailTaskDetail(w http.ResponseWriter, r *http.Request) {
	mailTaskULID, ok := requiredURLParam(w, r, "mail_task_ulid")
	if !ok {
		return
	}
	resp, err := h.Mall.GetMailTaskDetail(r.Context(), &mallpb.GetMailTaskDetailRequest{MailTaskUlid: mailTaskULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RetryMallMailTask(w http.ResponseWriter, r *http.Request) {
	mailTaskULID, ok := requiredURLParam(w, r, "mail_task_ulid")
	if !ok {
		return
	}
	body, ok := readRetryMailTaskBody(w, r)
	if !ok {
		return
	}
	resp, err := h.Mall.ResendMailTask(r.Context(), &mallpb.ResendMailTaskRequest{
		MailTaskUlid:               mailTaskULID,
		AdminUlid:                  adminActorID(r),
		UseNewEmail:                body.UseNewEmail,
		OverrideRecipientEmail:     optionalString(body.OverrideRecipientEmail),
		OverrideTemplateParamsJson: optionalString(body.OverrideTemplateParamsJson),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) IgnoreMallMailTask(w http.ResponseWriter, r *http.Request) {
	mailTaskULID, ok := requiredURLParam(w, r, "mail_task_ulid")
	if !ok {
		return
	}
	resp, err := h.Mall.IgnoreMailTask(r.Context(), &mallpb.IgnoreMailTaskRequest{
		MailTaskUlid: mailTaskULID,
		AdminUlid:    adminActorID(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListMallNatsMessages(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Mall.ListNatsMessages(r.Context(), &mallpb.ListNatsMessagesRequest{
		ReceiveStatus: queryText(r, "receive_status"),
		SourceService: queryText(r, "source_service"),
		Subject:       queryText(r, "subject"),
		MessageType:   queryText(r, "message_type"),
		Limit:         queryLimit(r),
		Offset:        queryOffset(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMallNatsMessageSummary(w http.ResponseWriter, r *http.Request) {
	messageULID, ok := requiredURLParam(w, r, "message_ulid")
	if !ok {
		return
	}
	resp, err := h.Mall.GetNatsMessageSummary(r.Context(), &mallpb.GetNatsMessageSummaryRequest{MessageUlid: messageULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetMallNatsMessageDetail(w http.ResponseWriter, r *http.Request) {
	messageULID, ok := requiredURLParam(w, r, "message_ulid")
	if !ok {
		return
	}
	resp, err := h.Mall.GetNatsMessageDetail(r.Context(), &mallpb.GetNatsMessageDetailRequest{MessageUlid: messageULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListProgMailTasks(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gprog.ListMailTasks(r.Context(), &gprogpb.ListMailTasksReq{
		CandidateUlid: queryText(r, "candidate_ulid"),
		PipelineUlid:  queryText(r, "pipeline_ulid"),
		Limit:         queryLimit(r),
		Offset:        queryOffset(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetProgMailTaskDetail(w http.ResponseWriter, r *http.Request) {
	mailTaskULID, ok := requiredURLParam(w, r, "mail_task_ulid")
	if !ok {
		return
	}
	resp, err := h.Gprog.GetMailTaskDetail(r.Context(), &gprogpb.GetMailTaskDetailReq{MailTaskUlid: mailTaskULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RetryProgMailTask(w http.ResponseWriter, r *http.Request) {
	mailTaskULID, ok := requiredURLParam(w, r, "mail_task_ulid")
	if !ok {
		return
	}
	body, ok := readRetryMailTaskBody(w, r)
	if !ok {
		return
	}
	resp, err := h.Gprog.ResendMailTask(r.Context(), &gprogpb.ResendMailTaskReq{
		MailTaskUlid:               mailTaskULID,
		AdminUlid:                  adminActorID(r),
		UseNewEmail:                body.UseNewEmail,
		OverrideRecipientEmail:     optionalString(body.OverrideRecipientEmail),
		OverrideTemplateParamsJson: optionalString(body.OverrideTemplateParamsJson),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) IgnoreProgMailTask(w http.ResponseWriter, r *http.Request) {
	mailTaskULID, ok := requiredURLParam(w, r, "mail_task_ulid")
	if !ok {
		return
	}
	resp, err := h.Gprog.IgnoreMailTask(r.Context(), &gprogpb.IgnoreMailTaskReq{
		MailTaskUlid: mailTaskULID,
		AdminUlid:    adminActorID(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListProgStages(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gprog.ListStages(r.Context(), &gprogpb.ListStagesReq{
		PipelineUlid: queryText(r, "pipeline_ulid"),
		Limit:        queryLimit(r),
		Offset:       queryOffset(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetProgStageDetail(w http.ResponseWriter, r *http.Request) {
	stageULID, ok := requiredURLParam(w, r, "stage_ulid")
	if !ok {
		return
	}
	resp, err := h.Gprog.GetStageDetail(r.Context(), &gprogpb.GetStageDetailReq{StageUlid: stageULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListProgCourseUnits(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gprog.ListCourseUnits(r.Context(), &gprogpb.ListCourseUnitsReq{
		PipelineUlid: queryText(r, "pipeline_ulid"),
		StageUlid:    queryText(r, "stage_ulid"),
		Status:       parseCourseUnitStatus(r),
		Limit:        queryLimit(r),
		Offset:       queryOffset(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListProgDriverEvents(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gprog.ListDriverEvents(r.Context(), &gprogpb.ListDriverEventsReq{
		EntityType:  queryText(r, "entity_type"),
		EntityUlid:  queryText(r, "entity_ulid"),
		EventStatus: queryText(r, "event_status"),
		EventType:   queryText(r, "event_type"),
		Limit:       queryLimit(r),
		Offset:      queryOffset(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetProgDriverEventDetail(w http.ResponseWriter, r *http.Request) {
	eventULID, ok := requiredURLParam(w, r, "event_ulid")
	if !ok {
		return
	}
	resp, err := h.Gprog.GetDriverEventDetail(r.Context(), &gprogpb.GetDriverEventDetailReq{EventUlid: eventULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListProgNatsMessages(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gprog.ListNatsMessages(r.Context(), &gprogpb.ListNatsMessagesReq{
		ReceiveStatus: queryText(r, "receive_status"),
		SourceService: queryText(r, "source_service"),
		Limit:         queryLimit(r),
		Offset:        queryOffset(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetProgNatsMessageDetail(w http.ResponseWriter, r *http.Request) {
	messageULID, ok := requiredURLParam(w, r, "message_ulid")
	if !ok {
		return
	}
	resp, err := h.Gprog.GetNatsMessageDetail(r.Context(), &gprogpb.GetNatsMessageDetailReq{MessageUlid: messageULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListExamAuditMessages(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gexam.ListAuditMessages(r.Context(), &gexampb.ListAuditMessagesRequest{
		ProcessedStatus: optionalQueryString(r, "processed_status"),
		EventType:       optionalQueryString(r, "event_type"),
		StartTime:       optionalQueryString(r, "start_time"),
		EndTime:         optionalQueryString(r, "end_time"),
		Page:            queryPage(r),
		PageSize:        queryPageSize(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetExamAuditMessageDetail(w http.ResponseWriter, r *http.Request) {
	messageULID, ok := requiredURLParam(w, r, "message_ulid")
	if !ok {
		return
	}
	resp, err := h.Gexam.GetAuditMessageDetail(r.Context(), &gexampb.GetAuditMessageDetailRequest{MessageUlid: messageULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListExamStatusTransitions(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gexam.ListExamStatusTransitions(r.Context(), &gexampb.ListExamStatusTransitionsRequest{
		ExamUlid:   optionalQueryString(r, "exam_ulid"),
		StatusType: optionalQueryString(r, "status_type"),
		Page:       queryPage(r),
		PageSize:   queryPageSize(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) ListExamReminderMails(w http.ResponseWriter, r *http.Request) {
	resp, err := h.Gexam.ListReminderMails(r.Context(), &gexampb.ListReminderMailsRequest{
		ExamUlid:       optionalQueryString(r, "exam_ulid"),
		TaskStatus:     optionalQueryString(r, "task_status"),
		DeliveryStatus: optionalQueryString(r, "delivery_status"),
		CandidateEmail: optionalQueryString(r, "candidate_email"),
		ReminderType:   optionalQueryString(r, "reminder_type"),
		Page:           queryPage(r),
		PageSize:       queryPageSize(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) GetExamReminderMailDetail(w http.ResponseWriter, r *http.Request) {
	mailULID, ok := requiredURLParam(w, r, "mail_ulid")
	if !ok {
		return
	}
	resp, err := h.Gexam.GetReminderMailDetail(r.Context(), &gexampb.GetReminderMailDetailRequest{MailUlid: mailULID})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) RetryExamReminderMail(w http.ResponseWriter, r *http.Request) {
	mailULID, ok := requiredURLParam(w, r, "mail_ulid")
	if !ok {
		return
	}
	body, ok := readRetryMailTaskBody(w, r)
	if !ok {
		return
	}
	resp, err := h.Gexam.RetryReminderMail(r.Context(), &gexampb.RetryReminderMailRequest{
		MailUlid:                   mailULID,
		AdminUlid:                  adminActorID(r),
		OverrideRecipientEmail:     optionalString(body.OverrideRecipientEmail),
		OverrideTemplateParamsJson: optionalString(body.OverrideTemplateParamsJson),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) IgnoreExamReminderMail(w http.ResponseWriter, r *http.Request) {
	mailULID, ok := requiredURLParam(w, r, "mail_ulid")
	if !ok {
		return
	}
	resp, err := h.Gexam.IgnoreReminderMail(r.Context(), &gexampb.IgnoreReminderMailRequest{
		MailUlid:  mailULID,
		AdminUlid: adminActorID(r),
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, resp)
}
