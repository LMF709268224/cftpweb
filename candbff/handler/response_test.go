package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestHandleGrpcErrorHidesInternalDetails(t *testing.T) {
	rec := httptest.NewRecorder()

	HandleGrpcError(rec, status.Error(codes.Internal, "database password leaked"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	resp := decodeAPIResponse(t, rec)
	if resp.Message != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("message = %q, want %q", resp.Message, http.StatusText(http.StatusInternalServerError))
	}
}

func TestHandleGrpcErrorKeepsBusinessValidationMessage(t *testing.T) {
	rec := httptest.NewRecorder()

	HandleGrpcError(rec, status.Error(codes.FailedPrecondition, "stage is not waiting for candidate action"))

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
	resp := decodeAPIResponse(t, rec)
	if resp.Message != "stage is not waiting for candidate action" {
		t.Fatalf("message = %q", resp.Message)
	}
}

func TestHandleGrpcErrorMapsCanceledRequestToClientClosed(t *testing.T) {
	rec := httptest.NewRecorder()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	HandleGrpcErrorWithContext(rec, ctx, status.Error(codes.Canceled, "context canceled"))

	if rec.Code != statusClientClosedRequest {
		t.Fatalf("status = %d, want %d", rec.Code, statusClientClosedRequest)
	}
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestHandleGrpcErrorDoesNotTreatDownstreamCanceledAsClientClosed(t *testing.T) {
	rec := httptest.NewRecorder()

	HandleGrpcError(rec, status.Error(codes.Canceled, "downstream operation canceled"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	resp := decodeAPIResponse(t, rec)
	if resp.Message != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("message = %q, want %q", resp.Message, http.StatusText(http.StatusInternalServerError))
	}
}

func TestHandleGrpcErrorHidesNonGRPCDetails(t *testing.T) {
	rec := httptest.NewRecorder()

	HandleGrpcError(rec, errors.New("connection string leaked"))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	resp := decodeAPIResponse(t, rec)
	if resp.Message != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("message = %q, want %q", resp.Message, http.StatusText(http.StatusInternalServerError))
	}
}

func decodeAPIResponse(t *testing.T, rec *httptest.ResponseRecorder) apiResponse {
	t.Helper()
	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v; body = %s", err, rec.Body.String())
	}
	return resp
}
