package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func newCandidateHandlerRequest(
	method string,
	target string,
	body string,
	candidateID string,
	routeParams map[string]string,
) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	ctx := request.Context()
	if len(routeParams) > 0 {
		routeContext := chi.NewRouteContext()
		for key, value := range routeParams {
			routeContext.URLParams.Add(key, value)
		}
		ctx = context.WithValue(ctx, chi.RouteCtxKey, routeContext)
	}
	return request.WithContext(WithCandidate(ctx, candidateID, "", "", ""))
}

func decodeHandlerAPIResponse(t *testing.T, recorder *httptest.ResponseRecorder) apiResponse {
	t.Helper()

	var response apiResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	return response
}

func assertHandlerAPIError(
	t *testing.T,
	recorder *httptest.ResponseRecorder,
	wantStatus int,
	wantErrorCode ErrorCode,
) apiResponse {
	t.Helper()

	if recorder.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%q", recorder.Code, wantStatus, recorder.Body.String())
	}
	response := decodeHandlerAPIResponse(t, recorder)
	if response.ErrorCode != wantErrorCode {
		t.Fatalf("error_code = %q, want %q; body=%q", response.ErrorCode, wantErrorCode, recorder.Body.String())
	}
	return response
}
