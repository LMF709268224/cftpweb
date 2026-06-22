package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type apiResponse struct {
	Code      int         `json:"code"`
	ErrorCode ErrorCode   `json:"error_code,omitempty"` // 涓氬姟閿欒鐮?
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
}

func WriteJSON(w http.ResponseWriter, httpStatus int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)

	resp := apiResponse{
		Code:      httpStatus,
		ErrorCode: "OK", // 鎴愬姛鏃跺浐瀹氳繑鍥?OK 鎴栦笉杩斿洖
		Message:   http.StatusText(httpStatus),
		Data:      data,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode JSON response", "error", err)
	}
}

func WriteError(w http.ResponseWriter, httpStatus int, errorCode ErrorCode, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(httpStatus)

	if message != "" {
		slog.Warn("API error", "status", httpStatus, "error_code", errorCode, "detail", message)
	}

	publicMessage := http.StatusText(httpStatus)
	if publicMessage == "" {
		publicMessage = "Error"
	}
	if (errorCode == ErrInvalidRequest || errorCode == ErrPrecondition) && message != "" {
		publicMessage = message
	}

	resp := apiResponse{
		Code:      httpStatus,
		ErrorCode: errorCode,
		Message:   publicMessage,
	}

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		slog.Error("Failed to encode JSON error response", "error", err)
	}
}

// HandleAppError 缁熶竴澶勭悊鑷畾涔変笟鍔￠敊璇?
func HandleAppError(w http.ResponseWriter, err *AppError) {
	WriteError(w, err.HttpStatus, err.Code, err.Message)
}

func HandleGrpcError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}

	st, ok := status.FromError(err)
	if !ok {
		WriteError(w, http.StatusInternalServerError, ErrInternal, err.Error())
		return
	}

	var httpStatus int
	var errorCode ErrorCode

	switch st.Code() {
	case codes.NotFound:
		httpStatus = http.StatusNotFound
		errorCode = ErrNotFound
	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
		errorCode = ErrInvalidRequest
	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
		errorCode = ErrInvalidRequest
	case codes.FailedPrecondition:
		httpStatus = http.StatusConflict
		errorCode = ErrPrecondition
	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
		errorCode = ErrForbidden
	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
		errorCode = ErrUnauthorized
	case codes.ResourceExhausted:
		httpStatus = http.StatusTooManyRequests
		errorCode = ErrInternal
	case codes.Unavailable:
		httpStatus = http.StatusServiceUnavailable
		errorCode = ErrServiceUnavailable
	case codes.DeadlineExceeded:
		httpStatus = http.StatusGatewayTimeout
		errorCode = ErrServiceUnavailable
	default:
		httpStatus = http.StatusInternalServerError
		errorCode = ErrInternal
	}

	WriteError(w, httpStatus, errorCode, st.Message())
}

func ReadJSON(r *http.Request, dest interface{}) error {
	r.Body = http.MaxBytesReader(nil, r.Body, 1<<20)
	return json.NewDecoder(r.Body).Decode(dest)
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, `"`, "")
	s = strings.ReplaceAll(s, `\`, "")
	s = strings.ReplaceAll(s, "/", "")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", "")
	s = strings.ReplaceAll(s, "\x00", "")
	return s
}
