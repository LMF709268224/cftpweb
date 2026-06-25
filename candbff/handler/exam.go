package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"strings"

	"candbff/config"

	"github.com/go-chi/chi/v5"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	mallpb "github.com/afnandelfin620-star/cftptest/cftp/gmall"
	"github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const examCallbackPath = "/api/public/webhooks/exams/callback"

const (
	retakeActionNone              = "NONE"
	retakeActionCreateRetakeOrder = "CREATE_RETAKE_ORDER"
	retakeActionContinuePayment   = "CONTINUE_PAYMENT"
	retakeActionApplyRetake       = "APPLY_RETAKE"
	retakeActionSignupExam        = "SIGNUP_EXAM"
)

type retakePaymentSnapshot struct {
	found                 bool
	paid                  bool
	message               string
	courseRetakeOrderUlid string
	orderStatus           string
	payOrderUlid          string
}

// SignupExam POST /api/exams/units/{courseUnitUlid}/signup
func (h *Handler) SignupExam(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseUnitUlid := strings.TrimSpace(chi.URLParam(r, "courseUnitUlid"))

	var input SignupExamInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	if courseUnitUlid != "" {
		input.CourseUnitULID = courseUnitUlid
	}
	if !requireRequestFields(w, candidateID, "candidate_id", input.CourseUnitULID, "course_unit_ulid") {
		return
	}

	resp, err := h.Gprog.CandidateSignupExam(r.Context(), &gprog.CandidateSignupExamReq{
		CourseUnitUlid:      input.CourseUnitULID,
		CandidateUlid:       candidateID,
		CandidateFirstName:  input.FirstName,
		CandidateMiddleName: input.MiddleName,
		CandidateLastName:   input.LastName,
		CandidateEmail:      input.Email,
		CandidateHomePhone:  input.HomePhone,
		CandidateWorkPhone:  input.WorkPhone,
		CandidateGender:     input.Gender,
		CandidateBirthdate:  input.Birthdate,
		CandidateCountry:    input.Country,
		CandidateProvince:   input.Province,
		CandidateCity:       input.City,
		CandidateAddress:    input.Address,
		CandidatePostalCode: input.PostalCode,
		SourceSystem:        "candbff",
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, CandidateSignupExamRsp{
		CourseUnitUlid:   resp.GetCourseUnitUlid(),
		CourseUnitStatus: resp.GetCourseUnitStatus(),
		Message:          resp.GetMessage(),
	})
}

// ApplyRetake POST /api/exams/units/{courseUnitUlid}/retake
func (h *Handler) ApplyRetake(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseUnitUlid := strings.TrimSpace(chi.URLParam(r, "courseUnitUlid"))
	if !requireRequestFields(w, candidateID, "candidate_id", courseUnitUlid, "course_unit_ulid") {
		return
	}

	resp, err := h.applyRetake(r.Context(), candidateID, courseUnitUlid)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, CandidateApplyRetakeRsp{
		CourseUnitUlid:   resp.GetCourseUnitUlid(),
		CourseUnitStatus: resp.GetCourseUnitStatus(),
		Message:          resp.GetMessage(),
	})
}

func (h *Handler) applyRetake(ctx context.Context, candidateID string, courseUnitUlid string) (*gprog.CandidateApplyRetakeRsp, error) {
	return h.Gprog.CandidateApplyRetake(ctx, &gprog.CandidateApplyRetakeReq{
		CourseUnitUlid: courseUnitUlid,
		CandidateUlid:  candidateID,
	})
}

// PrepareRetakePayment POST /api/exams/units/{courseUnitUlid}/retake-payment
func (h *Handler) PrepareRetakePayment(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseUnitUlid := strings.TrimSpace(chi.URLParam(r, "courseUnitUlid"))

	var input PrepareRetakePaymentInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}
	input.CourseUnitCcULID = strings.TrimSpace(input.CourseUnitCcULID)
	input.SuccessURL = strings.TrimSpace(input.SuccessURL)
	input.CancelURL = strings.TrimSpace(input.CancelURL)
	input.BundleOrderUlid = strings.TrimSpace(input.BundleOrderUlid)
	if !requireRequestFields(w, candidateID, "candidate_id", courseUnitUlid, "course_unit_ulid", input.CourseUnitCcULID, "course_unit_cc_ulid", input.BundleOrderUlid, "bundle_order_ulid") {
		return
	}

	payment, err := h.retakePaymentSnapshot(r.Context(), candidateID, courseUnitUlid, input.CourseUnitCcULID, input.BundleOrderUlid, input.RetriedCount)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	if payment.paid {
		retakeResp, err := h.applyRetake(r.Context(), candidateID, courseUnitUlid)
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		out := PrepareRetakePaymentRsp{
			CourseRetakeOrderUlid: payment.courseRetakeOrderUlid,
			OrderStatus:           payment.orderStatus,
			PayOrderUlid:          payment.payOrderUlid,
			PaymentRequired:       !isFreeRetakeMessage(payment.message),
			Paid:                  true,
			Message:               payment.message,
		}
		if retakeResp != nil {
			out.CourseUnitUlid = retakeResp.GetCourseUnitUlid()
			out.CourseUnitStatus = retakeResp.GetCourseUnitStatus().String()
			if retakeResp.GetMessage() != "" {
				out.Message = retakeResp.GetMessage()
			}
		}
		WriteJSON(w, http.StatusOK, out)
		return
	}

	orderResp, err := h.Mall.CreateCourseRetakeOrder(r.Context(), &mallpb.CreateCourseRetakeOrderRequest{
		CourseUnitUlid:   courseUnitUlid,
		CourseUnitCcUlid: input.CourseUnitCcULID,
		CandidateUlid:    candidateID,
		RetriedCount:     input.RetriedCount,
		BundleOrderUlid:  input.BundleOrderUlid,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	paymentKey := formatPaymentKey(orderResp.GetPaymentKey())
	payOrderUlid := orderResp.GetPayOrderUlid()
	if paymentKey == "" && orderResp.GetCourseRetakeOrderUlid() != "" {
		initResp, err := h.Mall.InitiatePayment(r.Context(), &mallpb.InitiatePaymentRequest{
			BizType:    orderBizCourseRetakePayment,
			BizRefUlid: orderResp.GetCourseRetakeOrderUlid(),
			SuccessUrl: input.SuccessURL,
			CancelUrl:  input.CancelURL,
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		paymentKey = formatPaymentKey(initResp.GetPaymentKey())
		if initResp.GetPayOrderUlid() != "" {
			payOrderUlid = initResp.GetPayOrderUlid()
		}
	}

	WriteJSON(w, http.StatusOK, PrepareRetakePaymentRsp{
		CourseRetakeOrderUlid: orderResp.GetCourseRetakeOrderUlid(),
		OrderStatus:           orderResp.GetOrderStatus(),
		PayOrderUlid:          payOrderUlid,
		PaymentKey:            paymentKey,
		PaymentRequired:       true,
		Paid:                  false,
		ReusedExisting:        orderResp.GetReusedExisting(),
		Message:               orderResp.GetMessage(),
	})
}

// GetScheduleURL GET /api/exams/{examId}/schedule-url
func (h *Handler) GetScheduleURL(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	examID := strings.TrimSpace(chi.URLParam(r, "examId"))
	pipelineULID := strings.TrimSpace(r.URL.Query().Get("pipeline_ulid"))
	courseULID := strings.TrimSpace(r.URL.Query().Get("course_ulid"))
	urlType, ok := parseExamURLType(w, r.URL.Query().Get("url_type"))
	if !ok {
		return
	}

	// 后端自行构建回调基准地址，不再依赖前端传入，提升安全性和整洁度
	termURLBase := examCallbackBaseURL(r)

	if pipelineULID == "" && courseULID == "" {
		if !requireRequestFields(w, candidateID, "candidate_id", examID, "exam_id") {
			return
		}
		resp, err := h.Gexam.GetScheduleURL(r.Context(), &gexampb.GetURLRequest{
			ExamUlid:    examID,
			TermUrlBase: termURLBase,
			UrlType:     examURLTypeForGexam(urlType),
		})
		if err != nil {
			HandleGrpcError(w, err)
			return
		}

		WriteJSON(w, http.StatusOK, GetScheduleURLRsp{URL: resp.GetUrl()})
		return
	}

	if !requireRequestFields(w, candidateID, "candidate_id", pipelineULID, "pipeline_ulid", courseULID, "course_ulid") {
		return
	}

	resp, err := h.Gprog.GetExamURL(r.Context(), &gprog.GetURLRequest{
		CandidateUlid: candidateID,
		PipelineUlid:  pipelineULID,
		CourseUlid:    courseULID,
		UrlType:       urlType,
		TermUrlBase:   termURLBase,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, GetScheduleURLRsp{URL: resp.GetUrl()})
}

func examCallbackBaseURL(r *http.Request) string {
	if configured := strings.TrimSpace(os.Getenv(config.EnvExamCallbackBaseURL)); configured != "" {
		configured = strings.TrimRight(configured, "/")
		if strings.HasSuffix(configured, examCallbackPath) {
			return configured
		}
		return configured + examCallbackPath
	}

	scheme := "https"
	if r.Header.Get("X-Forwarded-Proto") == "http" || (r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "") {
		scheme = "http"
	}
	return scheme + "://" + r.Host + examCallbackPath
}

// GetExamResult GET /api/exams/{examId}/result
func (h *Handler) GetExamResult(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	examID := strings.TrimSpace(chi.URLParam(r, "examId"))
	if !requireRequestFields(w, candidateID, "candidate_id", examID, "exam_id") {
		return
	}

	resp, err := h.Gexam.GetExamResultDetail(r.Context(), &gexampb.GetExamRequest{
		ExamUlid: examID,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			WriteJSON(w, http.StatusOK, ExamResultRsp{ExamID: examID, HasResult: false})
			return
		}
		HandleGrpcError(w, err)
		return
	}

	if resp == nil {
		WriteJSON(w, http.StatusOK, ExamResultRsp{ExamID: examID, HasResult: false})
		return
	}

	WriteJSON(w, http.StatusOK, ExamResultRsp{
		ExamID:          resp.GetExamUlid(),
		TotalScore:      resp.GetTotalScore(),
		IsPassed:        resp.GetIsPassed(),
		HasResult:       true,
		ScoreDetailsRaw: resp.GetScoreDetailsJson(),
	})
}

// TermUrlCallback POST /api/exams/{examId}/schedule-callback
func (h *Handler) TermUrlCallback(w http.ResponseWriter, r *http.Request) {
	examID := strings.TrimSpace(chi.URLParam(r, "examId"))
	var input TermUrlInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}
	if input.CallbackBody == "" {
		raw, err := json.Marshal(input)
		if err == nil {
			input.CallbackBody = string(raw)
		}
	}
	if !requireRequestFields(w, examID, "exam_id", input.URLType, "url_type") {
		return
	}

	resp, err := h.Gexam.TermUrlCallback(r.Context(), &gexampb.TermUrlCallbackRequest{
		ExamUlid:     examID,
		UrlType:      input.URLType,
		CallbackBody: input.CallbackBody,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, TermUrlCallbackRsp{
		ExamID:     resp.GetExamUlid(),
		ExamStatus: resp.GetExamStatus(),
	})
}

// TermUrlRedirectCallback GET /api/exams/{examId}/schedule-callback/{urlType}
func (h *Handler) TermUrlRedirectCallback(w http.ResponseWriter, r *http.Request) {
	examID := strings.TrimSpace(chi.URLParam(r, "examId"))
	urlType := strings.TrimSpace(chi.URLParam(r, "urlType"))
	if examID != "" && urlType != "" {
		callbackBody := r.URL.RawQuery
		if callbackBody == "" {
			callbackBody = "{}"
		}
		_, _ = h.Gexam.TermUrlCallback(r.Context(), &gexampb.TermUrlCallbackRequest{
			ExamUlid:     examID,
			UrlType:      urlType,
			CallbackBody: callbackBody,
		})
	}
	http.Redirect(w, r, "/exams?schedule_return=1", http.StatusFound)
}

// ApplyExemption POST /api/exams/units/{courseUnitUlid}/exemption
func (h *Handler) ApplyExemption(w http.ResponseWriter, r *http.Request) {
	// TODO(microservice-missing-api): gprog has not exposed candidate exemption application yet.
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction (waiting for gprog exemption API)")
}

// ListExams GET /api/exams
func (h *Handler) ListExams(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	if !requireRequestField(w, candidateID, "candidate_id") {
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	resultStatus := strings.TrimSpace(r.URL.Query().Get("result_status"))
	confirmationNumber := strings.TrimSpace(r.URL.Query().Get("confirmation_number"))
	courseUnitUlid := strings.TrimSpace(r.URL.Query().Get("course_unit_ulid"))
	page, pageSize := parsePagination(r, 20)

	req := &gexampb.ListExamsRequest{
		Page:     uint32(page),
		PageSize: uint32(pageSize),
	}
	if status != "" {
		req.Status = &status
	}
	if resultStatus != "" {
		req.ResultStatus = &resultStatus
	}
	if confirmationNumber != "" {
		req.ConfirmationNumber = &confirmationNumber
	}
	if courseUnitUlid != "" {
		req.CourseUnitUlid = &courseUnitUlid
	}
	if candidateID != "" {
		req.CandidateUlid = &candidateID
	}

	resp, err := h.Gexam.ListExams(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListExamsRsp{
		Exams: make([]ExamListItem, 0, len(resp.GetExams())),
		Total: resp.GetTotal(),
	}
	bundleOrdersByPipeline := h.completedBundleOrdersByPipeline(r, candidateID)
	for _, exam := range resp.GetExams() {
		if exam == nil {
			continue
		}

		examStatus := exam.GetExamStatus()
		if shouldShowWaitingExamConfirmation(exam) {
			examStatus = "WAITING_EXAM_CONFIRMATION"
		}

		item := ExamListItem{
			ExamUlid:             exam.GetExamUlid(),
			ProgramCode:          exam.GetProgramCode(),
			ExamCode:             exam.GetExamCode(),
			ExamStatus:           examStatus,
			ResultStatus:         exam.GetResultStatus(),
			TotalScore:           exam.GetTotalScore(),
			IsPassed:             exam.GetIsPassed(),
			CandidateFirstName:   exam.GetCandidateFirstName(),
			CandidateLastName:    exam.GetCandidateLastName(),
			CandidateEmail:       exam.GetCandidateEmail(),
			ConfirmationNumber:   exam.GetConfirmationNumber(),
			AppointmentStartTime: exam.GetAppointmentStartTime(),
			AppointmentEndTime:   exam.GetAppointmentEndTime(),
			SiteName:             exam.GetSiteName(),
			LastTermurlTimestamp: exam.GetLastTermurlTimestamp(),
			LastTermurlType:      exam.GetLastTermurlType(),
		}

		if detail, err := h.Gexam.GetExamDetail(r.Context(), &gexampb.GetExamRequest{ExamUlid: exam.GetExamUlid()}); err == nil && detail != nil {
			item.PipelineUlid = detail.GetPipelineUlid()
			item.CourseUnitUlid = detail.GetCourseUnitUlid()
		} else if err != nil {
			slog.Warn("ListExams get exam detail failed", "exam_id", exam.GetExamUlid(), "error", err)
		}

		if item.CourseUnitUlid != "" {
			unit, err := h.Gprog.GetCourseUnitDetail(r.Context(), &gprog.GetCourseUnitDetailReq{
				CourseUnitUlid: item.CourseUnitUlid,
			})
			if err != nil {
				slog.Warn("ListExams get course unit detail failed", "exam_id", exam.GetExamUlid(), "course_unit_ulid", item.CourseUnitUlid, "error", err)
			} else if unit != nil {
				item.CourseUnitCcUlid = unit.GetCourseUnitCcUlid()
				item.CourseUnitStatus = unit.GetStatus().String()
				item.RetriedCount = unit.GetRetriedCount()
				if item.PipelineUlid == "" {
					item.PipelineUlid = unit.GetPipelineUlid()
				}
				if item.BundleOrderUlid == "" && item.PipelineUlid != "" {
					if pipelineCcUlid := h.pipelineConfigIDFromRuntime(r, item.PipelineUlid); pipelineCcUlid != "" {
						item.BundleOrderUlid = bundleOrdersByPipeline[pipelineCcUlid]
					}
				}

				if unit.GetStatus() == gprog.CourseUnitStatus_COURSE_UNIT_STATUS_EXAM_FAILED && unit.GetCourseUnitCcUlid() != "" {
					eligibility, err := h.Gprog.ValidateRetakeEligibility(r.Context(), &gprog.ValidateRetakeEligibilityReq{
						CandidateUlid:    candidateID,
						CourseUnitUlid:   unit.GetCourseUnitUlid(),
						CourseUnitCcUlid: unit.GetCourseUnitCcUlid(),
					})
					if err != nil {
						slog.Warn("ListExams validate retake eligibility failed", "exam_id", exam.GetExamUlid(), "course_unit_ulid", item.CourseUnitUlid, "error", err)
					} else if eligibility != nil {
						item.RetakeEligible = eligibility.GetEligible()
						item.RetakeMessage = eligibility.GetMessage()
						item.NextRetriedCount = eligibility.GetNextRetriedCount()
					}
					item.Retake = h.buildExamRetakeState(r, candidateID, &item)
				}
			}
		}

		out.Exams = append(out.Exams, item)
	}
	suppressSupersededRetakeActions(out.Exams)

	WriteJSON(w, http.StatusOK, out)
}

func suppressSupersededRetakeActions(exams []ExamListItem) {
	latestByCourseUnit := make(map[string]int)
	for i := range exams {
		courseUnitUlid := strings.TrimSpace(exams[i].CourseUnitUlid)
		if courseUnitUlid == "" {
			continue
		}
		if !isRetakeRelevantExam(exams[i]) {
			continue
		}
		current, ok := latestByCourseUnit[courseUnitUlid]
		if !ok || compareExamRecency(exams[i], exams[current]) > 0 {
			latestByCourseUnit[courseUnitUlid] = i
		}
	}
	for i := range exams {
		courseUnitUlid := strings.TrimSpace(exams[i].CourseUnitUlid)
		latest, ok := latestByCourseUnit[courseUnitUlid]
		if courseUnitUlid == "" || !ok || latest == i {
			continue
		}
		exams[i].RetakeEligible = false
		if exams[i].Retake != nil {
			exams[i].Retake.Eligible = false
			exams[i].Retake.Action = retakeActionNone
		}
	}
}

func compareExamRecency(left ExamListItem, right ExamListItem) int {
	for _, pair := range [][2]string{
		{left.AppointmentStartTime, right.AppointmentStartTime},
		{left.AppointmentEndTime, right.AppointmentEndTime},
		{left.LastTermurlTimestamp, right.LastTermurlTimestamp},
	} {
		leftValue := strings.TrimSpace(pair[0])
		rightValue := strings.TrimSpace(pair[1])
		if leftValue == "" || rightValue == "" || leftValue == rightValue {
			continue
		}
		return strings.Compare(leftValue, rightValue)
	}
	return strings.Compare(strings.TrimSpace(left.ExamUlid), strings.TrimSpace(right.ExamUlid))
}

func isRetakeRelevantExam(item ExamListItem) bool {
	return isPendingExamAttempt(item) || isFinalFailedExamResult(item)
}

func isPendingExamAttempt(item ExamListItem) bool {
	examStatus := strings.ToUpper(strings.TrimSpace(item.ExamStatus))
	resultStatus := strings.ToUpper(strings.TrimSpace(item.ResultStatus))
	if examStatus != "DONE" {
		return true
	}
	return resultStatus == "" || resultStatus == "NONE"
}

func isFinalFailedExamResult(item ExamListItem) bool {
	examStatus := strings.ToUpper(strings.TrimSpace(item.ExamStatus))
	resultStatus := strings.ToUpper(strings.TrimSpace(item.ResultStatus))
	if examStatus != "DONE" {
		return false
	}
	if resultStatus == "" || resultStatus == "NONE" {
		return false
	}
	return !item.IsPassed
}

func (h *Handler) buildExamRetakeState(r *http.Request, candidateID string, item *ExamListItem) *ExamRetakeState {
	if item == nil {
		return nil
	}
	state := &ExamRetakeState{
		Eligible:         item.RetakeEligible,
		Message:          item.RetakeMessage,
		NextRetriedCount: item.NextRetriedCount,
		RequiresPayment:  true,
		Action:           retakeActionNone,
	}
	if !item.RetakeEligible {
		return state
	}
	if !isFinalFailedExamResult(*item) {
		state.Eligible = false
		return state
	}
	if strings.TrimSpace(item.BundleOrderUlid) == "" {
		if state.Message == "" {
			state.Message = "missing bundle order"
		}
		return state
	}
	state.Action = retakeActionCreateRetakeOrder
	retriedCount := item.NextRetriedCount
	if retriedCount == 0 {
		retriedCount = item.RetriedCount
	}
	payment, err := h.retakePaymentSnapshot(r.Context(), candidateID, item.CourseUnitUlid, item.CourseUnitCcUlid, item.BundleOrderUlid, retriedCount)
	if err != nil {
		slog.Warn("ListExams build retake state failed", "exam_id", item.ExamUlid, "course_unit_ulid", item.CourseUnitUlid, "error", err)
		return state
	}
	state.Message = firstNonEmpty(payment.message, state.Message)
	state.RequiresPayment = !isFreeRetakeMessage(payment.message)
	state.PaymentFound = payment.found
	state.PaymentPaid = payment.paid
	state.CourseRetakeOrderUlid = payment.courseRetakeOrderUlid
	state.OrderStatus = payment.orderStatus
	state.PayOrderUlid = payment.payOrderUlid
	switch {
	case !state.RequiresPayment || payment.paid:
		state.Action = retakeActionApplyRetake
	case payment.found:
		state.Action = retakeActionContinuePayment
	default:
		state.Action = retakeActionCreateRetakeOrder
	}
	return state
}

func (h *Handler) retakePaymentSnapshot(ctx context.Context, candidateID, courseUnitUlid, courseUnitCcUlid, bundleOrderUlid string, retriedCount uint32) (retakePaymentSnapshot, error) {
	out := retakePaymentSnapshot{}
	statusResp, err := h.Mall.GetCourseUnitRetakePaymentStatus(ctx, &mallpb.GetCourseUnitRetakePaymentStatusRequest{
		BundleOrderUlid:  bundleOrderUlid,
		CourseUnitCcUlid: courseUnitCcUlid,
		RetriedCount:     retriedCount,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			out.message = "course retake payment not found"
		} else {
			slog.Warn("retake payment snapshot status check failed", "candidate_id", candidateID, "course_unit_ulid", courseUnitUlid, "course_unit_cc_ulid", courseUnitCcUlid, "error", err)
		}
	} else if statusResp != nil {
		out.found = statusResp.GetFound()
		out.paid = statusResp.GetPaid()
		out.message = statusResp.GetMessage()
	}

	orders, err := h.Mall.ListCourseRetakeOrders(ctx, &mallpb.ListCourseRetakeOrdersRequest{
		CandidateUlid:    candidateID,
		CourseUnitUlid:   courseUnitUlid,
		CourseUnitCcUlid: courseUnitCcUlid,
		BundleOrderUlid:  bundleOrderUlid,
		Limit:            20,
	})
	if err != nil {
		slog.Warn("retake payment snapshot list orders failed", "candidate_id", candidateID, "course_unit_ulid", courseUnitUlid, "course_unit_cc_ulid", courseUnitCcUlid, "error", err)
		return out, nil
	}
	latestCreatedAt := ""
	for _, order := range orders.GetItems() {
		if order == nil || order.GetRetriedCount() != retriedCount {
			continue
		}
		if out.courseRetakeOrderUlid != "" && strings.Compare(order.GetCreatedAt(), latestCreatedAt) <= 0 {
			continue
		}
		out.found = true
		out.courseRetakeOrderUlid = order.GetCourseRetakeOrderUlid()
		out.orderStatus = order.GetOrderStatus()
		out.payOrderUlid = order.GetPayOrderUlid()
		latestCreatedAt = order.GetCreatedAt()
		if candidateOrderStatus(order.GetOrderStatus()) == "completed" {
			out.paid = true
		}
	}
	return out, nil
}

func isFreeRetakeMessage(message string) bool {
	normalized := strings.ToLower(strings.TrimSpace(message))
	return strings.Contains(normalized, "free") || strings.Contains(normalized, "免费")
}

func (h *Handler) completedBundleOrdersByPipeline(r *http.Request, candidateID string) map[string]string {
	out := make(map[string]string)
	createdAtByPipeline := make(map[string]string)
	if strings.TrimSpace(candidateID) == "" {
		return out
	}
	resp, err := h.Mall.ListBundleOrders(r.Context(), &mallpb.ListBundleOrdersRequest{
		CandidateUlid: candidateID,
		Limit:         100,
	})
	if err != nil {
		slog.Warn("ListExams list bundle orders failed", "candidate_id", candidateID, "error", err)
		return out
	}
	for _, order := range resp.GetItems() {
		if order == nil || strings.TrimSpace(order.GetBundleOrderUlid()) == "" {
			continue
		}
		if candidateOrderStatus(order.GetOrderStatus()) != "completed" {
			continue
		}
		bundle, err := h.Mall.GetBundle(r.Context(), &mallpb.GetBundleRequest{
			Query: &mallpb.GetBundleRequest_BundleUlid{BundleUlid: order.GetBundleUlid()},
		})
		if err != nil {
			slog.Warn("ListExams get bundle for order failed", "bundle_id", order.GetBundleUlid(), "bundle_order_ulid", order.GetBundleOrderUlid(), "error", err)
			continue
		}
		pipelineCcUlid := h.extractPipelineID(bundle.GetBundle())
		if pipelineCcUlid == "" {
			continue
		}
		if out[pipelineCcUlid] == "" || strings.Compare(order.GetCreatedAt(), createdAtByPipeline[pipelineCcUlid]) > 0 {
			out[pipelineCcUlid] = order.GetBundleOrderUlid()
			createdAtByPipeline[pipelineCcUlid] = order.GetCreatedAt()
		}
	}
	return out
}

func (h *Handler) pipelineConfigIDFromRuntime(r *http.Request, pipelineUlid string) string {
	pipelineUlid = strings.TrimSpace(pipelineUlid)
	if pipelineUlid == "" {
		return ""
	}
	resp, err := h.Gprog.GetPipelineDetail(r.Context(), &gprog.GetPipelineDetailReq{
		PipelineUlid: pipelineUlid,
	})
	if err != nil {
		slog.Warn("ListExams get pipeline detail failed", "pipeline_ulid", pipelineUlid, "error", err)
		return ""
	}
	return strings.TrimSpace(resp.GetPipeline().GetPipelineCcUlid())
}

func shouldShowWaitingExamConfirmation(exam *gexampb.ExamInfo) bool {
	if exam == nil {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(exam.GetExamStatus()), "OPEN") {
		return false
	}
	if strings.TrimSpace(exam.GetLastTermurlTimestamp()) == "" {
		return false
	}
	return true
}

// ListExamHistory GET /api/exams/history
func (h *Handler) ListExamHistory(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if query.Get("result_status") == "" {
		query.Set("result_status", "DONE")
	}
	r.URL.RawQuery = query.Encode()
	h.ListExams(w, r)
}

func parseExamURLType(w http.ResponseWriter, raw string) (gprog.ExamURLType, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "url_type is required")
		return gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN, false
	}
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		urlType := gprog.ExamURLType(n)
		if urlType != gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN {
			return urlType, true
		}
	}
	normalized := strings.ToUpper(strings.ReplaceAll(value, "-", "_"))
	aliases := map[string]gprog.ExamURLType{
		"SCHD":         gprog.ExamURLType_OFFLINE_SCHED,
		"RESCHD":       gprog.ExamURLType_OFFLINE_RESCH,
		"PROCTORSCH":   gprog.ExamURLType_ONLINE_SCHED,
		"PROCTORRESCH": gprog.ExamURLType_ONLINE_RESCH,
		"CANCEL":       gprog.ExamURLType_CANCEL,
	}
	if urlType, ok := aliases[normalized]; ok {
		return urlType, true
	}
	enumValue, ok := gprog.ExamURLType_value[normalized]
	if !ok || enumValue == int32(gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "url_type is invalid")
		return gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN, false
	}
	return gprog.ExamURLType(enumValue), true
}

func examURLTypeForGexam(urlType gprog.ExamURLType) string {
	switch urlType {
	case gprog.ExamURLType_OFFLINE_SCHED:
		return "schd"
	case gprog.ExamURLType_OFFLINE_RESCH:
		return "reschd"
	case gprog.ExamURLType_ONLINE_SCHED:
		return "proctorsch"
	case gprog.ExamURLType_ONLINE_RESCH:
		return "proctorresch"
	case gprog.ExamURLType_CANCEL:
		return "cancel"
	default:
		return ""
	}
}

func truncateForLog(value string, maxLen int) string {
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "...(truncated)"
}

func requestClientAddr(r *http.Request) string {
	if forwardedFor := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwardedFor != "" {
		parts := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(parts[0])
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	return r.RemoteAddr
}

// ThirdPartyExamCallback POST /api/public/webhooks/exams/callback/{urlType}/{examId}
func (h *Handler) ThirdPartyExamCallback(w http.ResponseWriter, r *http.Request) {
	urlType := chi.URLParam(r, "urlType")
	examId := chi.URLParam(r, "examId")
	logAttrs := []any{
		"exam_id", examId,
		"url_type", urlType,
		"method", r.Method,
		"path", r.URL.Path,
		"raw_query", truncateForLog(r.URL.RawQuery, 512),
		"remote_addr", requestClientAddr(r),
		"user_agent", truncateForLog(r.UserAgent(), 256),
		"content_type", r.Header.Get("Content-Type"),
		"content_length", r.ContentLength,
	}
	slog.Info("ThirdPartyExamCallback received", logAttrs...)

	// Helper function for rendering auto-closing HTML
	renderCloseHTML := func(success bool, errMsg string) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if success {
			w.Write([]byte(`
				<html><body>
					<h3>Appointment status synced successfully!</h3>
					<p>You can now close this window and return to your learning portal.</p>
					<script>
						if (window.opener) { window.opener.postMessage("schedule_success", "*"); }
						setTimeout(function() { window.close(); }, 3000);
					</script>
				</body></html>
			`))
		} else {
			w.Write([]byte(`
				<html><body>
					<h3 style="color:red;">Failed to sync appointment status</h3>
					<p>Error details: ` + errMsg + `</p>
					<p>Please close this window and try again.</p>
				</body></html>
			`))
		}
	}

	// 1. 解析 Form 数据
	if err := r.ParseForm(); err != nil {
		slog.Warn("ThirdPartyExamCallback parse form failed", append(logAttrs, "error", err)...)
		renderCloseHTML(false, "parse form error: "+err.Error())
		return
	}
	slog.Info("ThirdPartyExamCallback form parsed", append(logAttrs, "form_keys", len(r.Form))...)

	// 2. 提取 apptdata 字段
	apptDataRaw := r.FormValue("apptdata")
	if apptDataRaw == "" {
		slog.Warn("ThirdPartyExamCallback missing apptdata", logAttrs...)
		renderCloseHTML(false, "empty apptdata")
		return
	}
	slog.Info("ThirdPartyExamCallback apptdata extracted",
		append(logAttrs,
			"apptdata_length", len(apptDataRaw),
			"apptdata_preview", truncateForLog(apptDataRaw, 1024),
		)...,
	)

	// 3. 包装成 JSON 字符串，满足 gexam 对 callback_body 必须是合法 JSON 的要求
	bodyMap := map[string]string{"raw_xml": apptDataRaw}
	bodyJson, err := json.Marshal(bodyMap)
	if err != nil {
		slog.Error("ThirdPartyExamCallback marshal callback body failed",
			append(logAttrs,
				"apptdata_length", len(apptDataRaw),
				"error", err,
			)...,
		)
		renderCloseHTML(false, "json marshal error")
		return
	}
	slog.Info("ThirdPartyExamCallback calling gprog",
		append(logAttrs,
			"callback_body_length", len(bodyJson),
		)...,
	)

	// 4. 将结果发送给 gprog 的 ExamUrlCallback
	resp, err := h.Gprog.ExamUrlCallback(r.Context(), &gprog.ExamUrlCallbackReq{
		ExamUlid:     examId,
		UrlType:      urlType,
		CallbackBody: string(bodyJson),
	})
	if err != nil {
		slog.Error("ThirdPartyExamCallback gprog processing failed",
			append(logAttrs,
				"callback_body_length", len(bodyJson),
				"error", err,
			)...,
		)
		renderCloseHTML(false, "backend processing failed: "+err.Error())
		return
	}
	slog.Info("ThirdPartyExamCallback processed successfully",
		append(logAttrs,
			"callback_body_length", len(bodyJson),
			"gprog_response", resp,
		)...,
	)

	// 5. 成功后返回自动关闭窗口的 HTML
	renderCloseHTML(true, "")
}
