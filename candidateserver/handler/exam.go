package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	gexampb "github.com/afnandelfin620-star/cftptest/cftp/gexam"
	"github.com/afnandelfin620-star/cftptest/cftp/gprog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
		SourceSystem:        "candidateserver",
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

	resp, err := h.Gprog.CandidateApplyRetake(r.Context(), &gprog.CandidateApplyRetakeReq{
		CourseUnitUlid: courseUnitUlid,
		CandidateUlid:  candidateID,
	})
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
	scheme := "https"
	if r.Header.Get("X-Forwarded-Proto") == "http" || (r.TLS == nil && r.Header.Get("X-Forwarded-Proto") == "") {
		scheme = "http"
	}
	termURLBase := scheme + "://" + r.Host + "/api/public/webhooks/exams/callback"

	if pipelineULID == "" && courseULID == "" {
		if !requireRequestFields(w, candidateID, "candidate_id", examID, "exam_id") {
			return
		}
		resp, err := h.Gexam.GetScheduleURL(r.Context(), &gexampb.GetURLRequest{
			ExamId:      examID,
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
		CandidateId:  candidateID,
		PipelineUlid: pipelineULID,
		CourseUlid:   courseULID,
		UrlType:      urlType,
		TermUrlBase:  termURLBase,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, GetScheduleURLRsp{URL: resp.GetUrl()})
}

// GetExamResult GET /api/exams/{examId}/result
func (h *Handler) GetExamResult(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	examID := strings.TrimSpace(chi.URLParam(r, "examId"))
	if !requireRequestFields(w, candidateID, "candidate_id", examID, "exam_id") {
		return
	}

	resp, err := h.Gexam.GetExamResultDetail(r.Context(), &gexampb.GetExamRequest{
		ExamId: examID,
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
		ExamID:          resp.GetExamId(),
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
		ExamId:       examID,
		UrlType:      input.URLType,
		CallbackBody: input.CallbackBody,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, TermUrlCallbackRsp{
		ExamID:     resp.GetExamId(),
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
			ExamId:       examID,
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
	page := uint32(1)
	pageSize := uint32(20)
	if raw := strings.TrimSpace(r.URL.Query().Get("page")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, 32); err == nil && parsed > 0 {
			page = uint32(parsed)
		}
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("page_size")); raw != "" {
		if parsed, err := strconv.ParseUint(raw, 10, 32); err == nil && parsed > 0 {
			pageSize = uint32(parsed)
		}
	}

	req := &gexampb.ListExamsRequest{
		Page:     page,
		PageSize: pageSize,
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
		req.CandidateId = &candidateID
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
	for _, exam := range resp.GetExams() {
		if exam == nil {
			continue
		}

		examStatus := exam.GetExamStatus()
		if shouldShowWaitingExamConfirmation(exam) {
			examStatus = "WAITING_EXAM_CONFIRMATION"
		}

		out.Exams = append(out.Exams, ExamListItem{
			ExamId:               exam.GetExamId(),
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
		})
	}

	WriteJSON(w, http.StatusOK, out)
}

func shouldShowWaitingExamConfirmation(exam *gexampb.ExamInfo) bool {
	if exam == nil || strings.TrimSpace(exam.GetLastTermurlTimestamp()) == "" {
		return false
	}
	if strings.TrimSpace(exam.GetConfirmationNumber()) != "" ||
		strings.TrimSpace(exam.GetAppointmentStartTime()) != "" ||
		strings.TrimSpace(exam.GetAppointmentEndTime()) != "" ||
		strings.TrimSpace(exam.GetSiteName()) != "" {
		return false
	}
	if strings.TrimSpace(exam.GetResultStatus()) != "" ||
		exam.GetTotalScore() != 0 ||
		exam.GetIsPassed() {
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
		ExamId:       examId,
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
