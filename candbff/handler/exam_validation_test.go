package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestExamHandlersRejectInvalidRequestsBeforeCallingServices(t *testing.T) {
	tests := []struct {
		name        string
		method      string
		target      string
		body        string
		routeParams map[string]string
		handle      func(*Handler, http.ResponseWriter, *http.Request)
	}{
		{
			name:        "signup profile is required",
			method:      http.MethodPost,
			target:      "/api/exams/units/unit-1/signup",
			body:        `{}`,
			routeParams: map[string]string{"courseUnitUlid": "unit-1"},
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.SignupExam(w, r)
			},
		},
		{
			name:   "retake course unit is required",
			method: http.MethodPost,
			target: "/api/exams/units//retake",
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.ApplyRetake(w, r)
			},
		},
		{
			name:        "retake payment context is required",
			method:      http.MethodPost,
			target:      "/api/exams/units/unit-1/retake-payment",
			body:        `{}`,
			routeParams: map[string]string{"courseUnitUlid": "unit-1"},
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.PrepareRetakePayment(w, r)
			},
		},
		{
			name:        "schedule URL type is required",
			method:      http.MethodGet,
			target:      "/api/exams/exam-1/schedule-url",
			routeParams: map[string]string{"examId": "exam-1"},
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.GetScheduleURL(w, r)
			},
		},
		{
			name:   "exam result ID is required",
			method: http.MethodGet,
			target: "/api/exams//result",
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.GetExamResult(w, r)
			},
		},
		{
			name:        "schedule callback URL type is required",
			method:      http.MethodPost,
			target:      "/api/exams/exam-1/schedule-callback",
			body:        `{}`,
			routeParams: map[string]string{"examId": "exam-1"},
			handle: func(h *Handler, w http.ResponseWriter, r *http.Request) {
				h.TermUrlCallback(w, r)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := newCandidateHandlerRequest(
				test.method,
				test.target,
				test.body,
				"candidate-1",
				test.routeParams,
			)
			recorder := httptest.NewRecorder()

			test.handle(&Handler{}, recorder, request)

			assertHandlerAPIError(t, recorder, http.StatusBadRequest, ErrInvalidRequest)
		})
	}
}

func TestApplyExemptionReportsUnavailableContract(t *testing.T) {
	request := newCandidateHandlerRequest(
		http.MethodPost,
		"/api/exams/units/unit-1/exemption",
		`{}`,
		"candidate-1",
		map[string]string{"courseUnitUlid": "unit-1"},
	)
	recorder := httptest.NewRecorder()

	(&Handler{}).ApplyExemption(recorder, request)

	assertHandlerAPIError(t, recorder, http.StatusNotImplemented, ErrNotImplemented)
}
