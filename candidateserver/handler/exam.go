package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/afnandelfin620-star/cftptest/cftp/gprog"
)

// SignupExam  POST /api/exams/{courseId}/signup
// 考生报名接口，填报个人资料并关联考试单元
func (h *Handler) SignupExam(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseUnitUlid := chi.URLParam(r, "courseUnitUlid")

	var input SignupExamInput
	if err := ReadJSON(r, &input); err != nil {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "invalid request body: "+err.Error())
		return
	}

	// 如果 URL 中有 courseId，优先使用 URL 中的
	if courseUnitUlid != "" {
		input.CourseUnitULID = courseUnitUlid
	}

	if input.CourseUnitULID == "" {
		WriteError(w, http.StatusBadRequest, ErrInvalidRequest, "course_unit_ulid is required")
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

	out := CandidateSignupExamRsp{
		CourseUnitUlid:   resp.GetCourseUnitUlid(),
		CourseUnitStatus: resp.GetCourseUnitStatus(),
		Message:          resp.GetMessage(),
	}

	WriteJSON(w, http.StatusOK, out)
}

// ApplyRetake  POST /api/exams/{courseId}/retake
// 申请补考资格
func (h *Handler) ApplyRetake(w http.ResponseWriter, r *http.Request) {
	candidateID := CandidateID(r)
	courseId := chi.URLParam(r, "courseId")

	resp, err := h.Gprog.CandidateApplyRetake(r.Context(), &gprog.CandidateApplyRetakeReq{
		CourseUnitUlid: courseId,
		CandidateUlid:  candidateID,
	})
	if err != nil {
		HandleGrpcError(w, err)
		return
	}

	out := CandidateApplyRetakeRsp{
		CourseUnitUlid:   resp.GetCourseUnitUlid(),
		CourseUnitStatus: resp.GetCourseUnitStatus(),
		Message:          resp.GetMessage(),
	}

	WriteJSON(w, http.StatusOK, out)
}

// GetScheduleURL  GET /api/exams/{examId}/schedule
// 获取前往第三方考试平台预约座位的 URL
func (h *Handler) GetScheduleURL(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要 gprog 暴露 GetCandidateScheduleUrl 接口，不能直接调用 gexam
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction due to architecture enforcement (waiting for gprog)")
}

// GetExamResult  GET /api/exams/{examId}/result
// 获取考试结果
func (h *Handler) GetExamResult(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要 gprog 暴露 GetCandidateExamResult 接口，不能直接调用 gexam
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction due to architecture enforcement (waiting for gprog)")
}

// TermUrlCallback  POST /api/exams/{examId}/termurl
// 前端在收到 GEE 回跳并提取参数后，调用此接口将结果同步给 BFF
func (h *Handler) TermUrlCallback(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要 gprog 新增 NotifyExamBookingResult gRPC 接口
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction due to architecture enforcement (waiting for gprog)")
}

// ApplyExemption  POST /api/exams/{examId}/exemption  申请免考
func (h *Handler) ApplyExemption(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要 gprog 新增 ApplyExemption gRPC 接口
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction (waiting for gprog)")
}

// ListExams  GET /api/exams  获取考生所有考试列表
func (h *Handler) ListExams(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要 gprog 新增 ListExams(candidate_id) gRPC 接口
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction (waiting for gprog)")
}

// ListExamHistory  GET /api/exams/history  历史考试成绩
func (h *Handler) ListExamHistory(w http.ResponseWriter, r *http.Request) {
	// TODO: 需要 gprog 新增 GetExamHistory(candidate_id) gRPC 接口
	WriteError(w, http.StatusNotImplemented, ErrInternal, "Under construction (waiting for gprog)")
}
