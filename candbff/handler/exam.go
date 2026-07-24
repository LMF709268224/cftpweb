package handler

import (
	"context"
	"encoding/json"
	"html"
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

const (
	examCallbackPath         = "/api/public/webhooks/exams/callback"
	maxExamCallbackBodyBytes = 1 << 20
)

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
	isFree                bool
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
	if err := normalizeAndValidateSignupExamInput(&input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, err.Error())
		return
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
		CandidateWorkPhone:  input.Phone,
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
		CourseUnitStatus: resp.GetCourseUnitStatus().String(),
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
		CourseUnitStatus: resp.GetCourseUnitStatus().String(),
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

	pipelineUlid, err := h.retakePipelineUlid(r.Context(), courseUnitUlid, input.CourseUnitCcULID)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	payment, err := h.retakePaymentSnapshot(r.Context(), courseUnitUlid, input.CourseUnitCcULID, input.BundleOrderUlid, pipelineUlid, input.RetriedCount)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	reusedExisting := payment.found
	if !payment.found {
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
		payment.found = true
		payment.courseRetakeOrderUlid = orderResp.GetCourseRetakeOrderUlid()
		payment.orderStatus = orderResp.GetOrderStatus()
		payment.payOrderUlid = orderResp.GetPayOrderUlid()
		payment.message = firstNonEmpty(orderResp.GetMessage(), payment.message)
		reusedExisting = orderResp.GetReusedExisting()
	}

	if payment.isFree && !isOrderCompleted(payment.orderStatus) {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "free retake order was not completed")
		return
	}

	if payment.isFree || payment.paid {
		retakeResp, err := h.applyRetake(r.Context(), candidateID, courseUnitUlid)
		if err != nil {
			HandleGrpcError(w, err)
			return
		}
		out := PrepareRetakePaymentRsp{
			CourseRetakeOrderUlid: payment.courseRetakeOrderUlid,
			OrderStatus:           payment.orderStatus,
			PayOrderUlid:          payment.payOrderUlid,
			PaymentRequired:       !payment.isFree,
			Paid:                  true,
			ReusedExisting:        reusedExisting,
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

	if payment.courseRetakeOrderUlid == "" {
		WriteError(w, http.StatusBadGateway, ErrServiceUnavailable, "retake payment status did not include an order ID")
		return
	}

	initResp, err := h.Mall.InitiatePayment(r.Context(), &mallpb.InitiatePaymentRequest{
		BizType:    orderBizCourseRetakePayment,
		BizRefUlid: payment.courseRetakeOrderUlid,
		SuccessUrl: input.SuccessURL,
		CancelUrl:  input.CancelURL,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	paymentKey := formatPaymentKey(initResp.GetPaymentKey())
	if initResp.GetPayOrderUlid() != "" {
		payment.payOrderUlid = initResp.GetPayOrderUlid()
	}

	WriteJSON(w, http.StatusOK, PrepareRetakePaymentRsp{
		CourseRetakeOrderUlid: payment.courseRetakeOrderUlid,
		OrderStatus:           payment.orderStatus,
		PayOrderUlid:          payment.payOrderUlid,
		PaymentKey:            paymentKey,
		PaymentRequired:       true,
		Paid:                  false,
		ReusedExisting:        reusedExisting,
		Message:               payment.message,
	})
}

func (h *Handler) retakePipelineUlid(ctx context.Context, courseUnitUlid, expectedCourseUnitCcUlid string) (string, error) {
	unit, err := h.Gprog.GetCourseUnitDetail(ctx, &gprog.GetCourseUnitDetailReq{CourseUnitUlid: courseUnitUlid})
	if err != nil {
		return "", err
	}
	if unit == nil {
		return "", status.Error(codes.NotFound, "course unit not found")
	}
	if actual := strings.TrimSpace(unit.GetCourseUnitCcUlid()); actual == "" || actual != expectedCourseUnitCcUlid {
		return "", status.Error(codes.InvalidArgument, "course_unit_cc_ulid does not match the course unit")
	}
	pipelineUlid := strings.TrimSpace(unit.GetPipelineUlid())
	if pipelineUlid == "" {
		return "", status.Error(codes.FailedPrecondition, "course unit does not have an associated pipeline")
	}
	return pipelineUlid, nil
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
		if _, err := h.candidateExamDetail(r.Context(), candidateID, examID); err != nil {
			writeCandidateAccessError(w, err)
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

	if _, err := h.candidateExamDetail(r.Context(), candidateID, examID); err != nil {
		writeCandidateAccessError(w, err)
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
	candidateID := CandidateID(r)
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
	if _, err := h.candidateExamDetail(r.Context(), candidateID, examID); err != nil {
		writeCandidateAccessError(w, err)
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
	candidateID := CandidateID(r)
	examID := strings.TrimSpace(chi.URLParam(r, "examId"))
	urlType := strings.TrimSpace(chi.URLParam(r, "urlType"))
	if examID != "" && urlType != "" {
		if _, err := h.candidateExamDetail(r.Context(), candidateID, examID); err == nil {
			callbackBody := r.URL.RawQuery
			if callbackBody == "" {
				callbackBody = "{}"
			}
			_, err := h.Gexam.TermUrlCallback(r.Context(), &gexampb.TermUrlCallbackRequest{
				ExamUlid:     examID,
				UrlType:      urlType,
				CallbackBody: callbackBody,
			})
			if err != nil {
				slog.Warn("Failed to process exam term URL redirect callback", "candidate_id", candidateID, "exam_id", examID, "url_type", urlType, "error", err)
			}
		} else {
			slog.Warn("TermUrlRedirectCallback denied for non-owned exam", "candidate_id", candidateID, "exam_id", examID, "error", err)
		}
	}
	http.Redirect(w, r, "/exams?schedule_return=1", http.StatusFound)
}

func (h *Handler) candidateExamDetail(ctx context.Context, candidateID, examID string) (*gexampb.ExamDetail, error) {
	candidateID = strings.TrimSpace(candidateID)
	examID = strings.TrimSpace(examID)
	if candidateID == "" || examID == "" {
		return nil, NewError(http.StatusBadRequest, ErrInvalidRequest, "candidate_id and exam_id are required")
	}

	detail, err := h.Gexam.GetExamDetail(ctx, &gexampb.GetExamRequest{ExamUlid: examID})
	if err != nil {
		return nil, err
	}
	if detail == nil || strings.TrimSpace(detail.GetExamUlid()) == "" {
		return nil, status.Error(codes.NotFound, "exam not found")
	}
	if strings.TrimSpace(detail.GetCandidateUlid()) == candidateID {
		return detail, nil
	}
	if strings.TrimSpace(detail.GetCandidateUlid()) != "" {
		return nil, NewError(http.StatusForbidden, ErrForbidden, "exam is not available for current candidate")
	}

	cursor := ""
	for {
		req := &gexampb.ListExamsRequest{
			Filters: &gexampb.ExamFilters{
				CandidateUlid: &candidateID,
			},
			Cursor:   cursor,
			PageSize: 100,
		}
		resp, err := h.Gexam.ListExams(ctx, req)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.GetExams() {
			if item != nil && strings.TrimSpace(item.GetExamUlid()) == examID {
				return detail, nil
			}
		}
		if !resp.GetHasMore() || resp.GetNextCursor() == "" {
			break
		}
		cursor = resp.GetNextCursor()
	}
	return nil, NewError(http.StatusForbidden, ErrForbidden, "exam is not available for current candidate")
}

func writeCandidateAccessError(w http.ResponseWriter, err error) {
	if appErr, ok := err.(*AppError); ok {
		HandleAppError(w, appErr)
		return
	}
	HandleGrpcError(w, err)
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
	page := parseCursorPage(r, 20)

	req := &gexampb.ListExamsRequest{
		Filters:   &gexampb.ExamFilters{},
		Cursor:    page.Cursor,
		PageSize:  page.PageSize,
		SortOrder: gexampb.SortOrder(page.Sort),
	}
	if status != "" {
		req.Filters.Status = &status
	}
	if resultStatus != "" {
		req.Filters.ResultStatus = &resultStatus
	}
	if confirmationNumber != "" {
		req.Filters.ConfirmationNumber = &confirmationNumber
	}
	if courseUnitUlid != "" {
		req.Filters.CourseUnitUlid = &courseUnitUlid
	}
	if candidateID != "" {
		req.Filters.CandidateUlid = &candidateID
	}

	resp, err := h.Gexam.ListExams(r.Context(), req)
	if err != nil {
		HandleGrpcError(w, err)
		return
	}
	total, err := countCursorAll(r.Context(), func(ctx context.Context, cursor string, limit uint32) (uint32, string, error) {
		resp, err := h.Gexam.GetExamCount(ctx, &gexampb.GetExamCountRequest{
			Filters: req.GetFilters(),
			Limit:   limit,
			Cursor:  cursor,
		})
		if err != nil {
			return 0, "", err
		}
		return resp.GetCount(), resp.GetNextCursor(), nil
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := ListExamsRsp{
		Exams:      make([]ExamListItem, 0, len(resp.GetExams())),
		Total:      total.Total,
		TotalLabel: total.Label(),
		TotalExact: total.Exact,
		NextCursor: resp.GetNextCursor(),
		HasMore:    resp.GetHasMore(),
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

				if unit.GetStatus() == gprog.CourseUnitStatus_COURSE_UNIT_STATUS_EXAM_FAILED &&
					isCurrentCourseUnitExam(item.ExamUlid, unit.GetExamUlid()) &&
					unit.GetCourseUnitCcUlid() != "" {
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

	WriteJSON(w, http.StatusOK, out)
}

func isCurrentCourseUnitExam(examUlid, currentExamUlid string) bool {
	examUlid = strings.TrimSpace(examUlid)
	return examUlid != "" && examUlid == strings.TrimSpace(currentExamUlid)
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
	if strings.TrimSpace(item.PipelineUlid) == "" {
		state.Message = "missing course unit pipeline"
		return state
	}
	state.Action = retakeActionCreateRetakeOrder
	retriedCount := item.NextRetriedCount
	if retriedCount == 0 {
		retriedCount = item.RetriedCount
	}
	payment, err := h.retakePaymentSnapshot(r.Context(), item.CourseUnitUlid, item.CourseUnitCcUlid, item.BundleOrderUlid, item.PipelineUlid, retriedCount)
	if err != nil {
		slog.Warn("ListExams build retake state failed", "exam_id", item.ExamUlid, "course_unit_ulid", item.CourseUnitUlid, "error", err)
		state.Action = retakeActionNone
		state.Message = "retake payment status is unavailable"
		return state
	}
	state.Message = firstNonEmpty(payment.message, state.Message)
	state.RequiresPayment = !payment.isFree
	state.PaymentFound = payment.found
	state.PaymentPaid = payment.paid
	state.CourseRetakeOrderUlid = payment.courseRetakeOrderUlid
	state.OrderStatus = payment.orderStatus
	state.PayOrderUlid = payment.payOrderUlid
	state.Action = retakeActionForPayment(payment)
	if payment.isFree && payment.found && !isOrderCompleted(payment.orderStatus) {
		state.Message = "free retake order is not completed"
	}
	return state
}

func retakeActionForPayment(payment retakePaymentSnapshot) string {
	switch {
	case payment.isFree && !payment.found:
		return retakeActionCreateRetakeOrder
	case payment.isFree && isOrderCompleted(payment.orderStatus):
		return retakeActionApplyRetake
	case payment.isFree:
		return retakeActionNone
	case payment.paid:
		return retakeActionApplyRetake
	case payment.found:
		return retakeActionContinuePayment
	default:
		return retakeActionCreateRetakeOrder
	}
}

func (h *Handler) retakePaymentSnapshot(ctx context.Context, courseUnitUlid, courseUnitCcUlid, bundleOrderUlid, pipelineUlid string, retriedCount uint32) (retakePaymentSnapshot, error) {
	statusResp, err := h.Mall.GetCourseUnitRetakePaymentStatus(ctx, &mallpb.GetCourseUnitRetakePaymentStatusRequest{
		BundleOrderUlid:  bundleOrderUlid,
		CourseUnitCcUlid: courseUnitCcUlid,
		RetriedCount:     retriedCount,
		PipelineUlid:     pipelineUlid,
		CourseUnitUlid:   courseUnitUlid,
	})
	if err != nil {
		return retakePaymentSnapshot{}, err
	}
	if statusResp == nil {
		return retakePaymentSnapshot{}, status.Error(codes.Internal, "empty retake payment status response")
	}

	return retakePaymentSnapshot{
		found:                 statusResp.GetFound(),
		paid:                  statusResp.GetPaid(),
		isFree:                statusResp.GetIsFree(),
		message:               statusResp.GetMessage(),
		courseRetakeOrderUlid: statusResp.GetCourseRetakeOrderUlid(),
		orderStatus:           statusResp.GetOrderStatus(),
		payOrderUlid:          statusResp.GetPayOrderUlid(),
	}, nil
}

func (h *Handler) completedBundleOrdersByPipeline(r *http.Request, candidateID string) map[string]string {
	out := make(map[string]string)
	createdAtByPipeline := make(map[string]string)
	if strings.TrimSpace(candidateID) == "" {
		return out
	}
	resp, err := h.Mall.ListBundleOrders(r.Context(), &mallpb.ListBundleOrdersRequest{
		Filters: &mallpb.BundleOrderFilters{
			CandidateUlid: candidateID,
		},
		PageSize: 100,
	})
	if err != nil {
		slog.Warn("ListExams list bundle orders failed", "candidate_id", candidateID, "error", err)
		return out
	}
	for _, order := range resp.GetItems() {
		if order == nil || strings.TrimSpace(order.GetBundleOrderUlid()) == "" {
			continue
		}
		if !isOrderCompleted(order.GetOrderStatus()) {
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
	applyExamHistoryDefaults(r)
	h.ListExams(w, r)
}

func applyExamHistoryDefaults(r *http.Request) {
	query := r.URL.Query()
	if strings.TrimSpace(query.Get("status")) == "" {
		query.Set("status", "DONE")
	}
	r.URL.RawQuery = query.Encode()
}

func parseExamURLType(w http.ResponseWriter, raw string) (gprog.ExamURLType, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "url_type is required")
		return gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN, false
	}
	urlType, ok := parseExamURLTypeValue(value)
	if !ok {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "url_type is invalid")
		return gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN, false
	}
	return urlType, true
}

func parseExamURLTypeValue(raw string) (gprog.ExamURLType, bool) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN, false
	}
	if n, err := strconv.ParseInt(value, 10, 32); err == nil {
		urlType := gprog.ExamURLType(n)
		if _, exists := gprog.ExamURLType_name[int32(urlType)]; exists &&
			urlType != gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN {
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

func renderExamCallbackHTML(w http.ResponseWriter, success bool, message string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if success {
		_, _ = w.Write([]byte(`
			<html><body>
				<h3>Appointment status synced successfully!</h3>
				<p>You can now close this window and return to your learning portal.</p>
				<script>
					if (window.opener) { window.opener.postMessage("schedule_success", "*"); }
					setTimeout(function() { window.close(); }, 3000);
				</script>
			</body></html>
		`))
		return
	}
	_, _ = w.Write([]byte(`
		<html><body>
			<h3 style="color:red;">Failed to sync appointment status</h3>
			<p>Error details: ` + html.EscapeString(message) + `</p>
			<p>Please close this window and try again.</p>
		</body></html>
	`))
}

// ThirdPartyExamCallback POST /api/public/webhooks/exams/callback/{urlType}/{examId}
func (h *Handler) ThirdPartyExamCallback(w http.ResponseWriter, r *http.Request) {
	rawURLType := strings.TrimSpace(chi.URLParam(r, "urlType"))
	examID := strings.TrimSpace(chi.URLParam(r, "examId"))
	parsedURLType, validURLType := parseExamURLTypeValue(rawURLType)
	urlType := examURLTypeForGexam(parsedURLType)
	logAttrs := []any{
		"exam_id", examID,
		"url_type", urlType,
		"method", r.Method,
		"path", r.URL.Path,
		"query_length", len(r.URL.RawQuery),
		"remote_addr", requestClientAddr(r),
		"user_agent", truncateForLog(r.UserAgent(), 256),
		"content_type", r.Header.Get("Content-Type"),
		"content_length", r.ContentLength,
	}
	slog.Info("ThirdPartyExamCallback received", logAttrs...)

	if examID == "" || !validURLType || urlType == "" {
		slog.Warn("ThirdPartyExamCallback rejected invalid callback path", logAttrs...)
		renderExamCallbackHTML(w, false, "invalid callback URL")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxExamCallbackBodyBytes)
	if err := r.ParseForm(); err != nil {
		slog.Warn("ThirdPartyExamCallback parse form failed", append(logAttrs, "error", err)...)
		renderExamCallbackHTML(w, false, "invalid callback data")
		return
	}
	slog.Info("ThirdPartyExamCallback form parsed", append(logAttrs, "form_keys", len(r.Form))...)

	apptDataRaw := r.FormValue("apptdata")
	if strings.TrimSpace(apptDataRaw) == "" {
		slog.Warn("ThirdPartyExamCallback missing apptdata", logAttrs...)
		renderExamCallbackHTML(w, false, "empty callback data")
		return
	}
	slog.Info("ThirdPartyExamCallback apptdata extracted",
		append(logAttrs,
			"apptdata_length", len(apptDataRaw),
		)...,
	)

	// gprog expects the provider payload wrapped in valid JSON.
	bodyMap := map[string]string{"raw_xml": apptDataRaw}
	bodyJson, err := json.Marshal(bodyMap)
	if err != nil {
		slog.Error("ThirdPartyExamCallback marshal callback body failed",
			append(logAttrs,
				"apptdata_length", len(apptDataRaw),
				"error", err,
			)...,
		)
		renderExamCallbackHTML(w, false, "invalid callback data")
		return
	}
	slog.Info("ThirdPartyExamCallback calling gprog",
		append(logAttrs,
			"callback_body_length", len(bodyJson),
		)...,
	)

	_, err = h.Gprog.ExamUrlCallback(r.Context(), &gprog.ExamUrlCallbackReq{
		ExamUlid:     examID,
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
		renderExamCallbackHTML(w, false, "backend processing failed")
		return
	}
	slog.Info("ThirdPartyExamCallback processed successfully",
		append(logAttrs,
			"callback_body_length", len(bodyJson),
		)...,
	)

	renderExamCallbackHTML(w, true, "")
}
