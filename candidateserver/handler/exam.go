package handler

import (
	"encoding/json"
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
	pipelineULID := strings.TrimSpace(r.URL.Query().Get("pipeline_ulid"))
	courseULID := strings.TrimSpace(r.URL.Query().Get("course_ulid"))
	termURLBase := strings.TrimSpace(r.URL.Query().Get("term_url_base"))
	urlType, ok := parseExamURLType(w, r.URL.Query().Get("url_type"))
	if !ok {
		return
	}
	if !requireRequestFields(w, candidateID, "candidate_id", pipelineULID, "pipeline_ulid", courseULID, "course_ulid", termURLBase, "term_url_base") {
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

	resp, err := h.Gprog.ExamUrlCallback(r.Context(), &gprog.ExamUrlCallbackReq{
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
		Page:      page,
		PageSize:  pageSize,
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
		out.Exams = append(out.Exams, ExamListItem{
			ExamId:               exam.GetExamId(),
			ProgramCode:          exam.GetProgramCode(),
			ExamCode:             exam.GetExamCode(),
			ExamStatus:           exam.GetExamStatus(),
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
	enumValue, ok := gprog.ExamURLType_value[normalized]
	if !ok || enumValue == int32(gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN) {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "url_type is invalid")
		return gprog.ExamURLType_EXAM_URL_TYPE_UNKNOWN, false
	}
	return gprog.ExamURLType(enumValue), true
}
