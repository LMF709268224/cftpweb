package handler

import (
	"fmt"
	"net/http"
)

// ErrorCode 涓氬姟閿欒鐮侊紝鎺ㄨ崘浣跨敤瀛楃涓插舰寮忥紝鏇存槗璇?
type ErrorCode string

const (
	// 閫氱敤閿欒
	ErrInternal           ErrorCode = "INTERNAL_ERROR"
	ErrInvalidRequest     ErrorCode = "INVALID_REQUEST"
	ErrUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrForbidden          ErrorCode = "FORBIDDEN"
	ErrNotFound           ErrorCode = "NOT_FOUND"
	ErrNotImplemented     ErrorCode = "NOT_IMPLEMENTED"
	ErrPrecondition       ErrorCode = "PRECONDITION_FAILED"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	// 璁よ瘉妯″潡
	ErrAuthFailed          ErrorCode = "AUTH_FAILED"
	ErrTokenExpired        ErrorCode = "TOKEN_EXPIRED"
	ErrInvalidToken        ErrorCode = "INVALID_TOKEN"
	ErrPasswordIncorrect   ErrorCode = "PASSWORD_INCORRECT"
	ErrProfileUpdateFailed ErrorCode = "PROFILE_UPDATE_FAILED"

	// 璇剧▼/绠＄嚎妯″潡
	ErrPipelineNotFound ErrorCode = "PIPELINE_NOT_FOUND"
	ErrAlreadyPurchased ErrorCode = "ALREADY_PURCHASED"
	ErrInvalidPipeline  ErrorCode = "INVALID_PIPELINE"

	// 鑰冭瘯妯″潡
	ErrExamNotFound ErrorCode = "EXAM_NOT_FOUND"
	ErrNotEligible  ErrorCode = "NOT_ELIGIBLE"
	ErrSignupFailed ErrorCode = "SIGNUP_FAILED"
	ErrRetakeDenied ErrorCode = "RETAKE_DENIED"

	// 鏀粯/璁㈠崟妯″潡
	ErrPaymentFailed ErrorCode = "PAYMENT_FAILED"
	ErrOrderNotFound ErrorCode = "ORDER_NOT_FOUND"
	ErrInvalidAmount ErrorCode = "INVALID_AMOUNT"

	// 妗ｆ/浼氬憳妯″潡
	ErrMembershipExpired ErrorCode = "MEMBERSHIP_EXPIRED"
	ErrRecordRejected    ErrorCode = "RECORD_REJECTED"
)

// AppError 鑷畾涔変笟鍔￠敊璇?
type AppError struct {
	HttpStatus int
	Code       ErrorCode
	Message    string
}

func (e *AppError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func NewError(httpStatus int, code ErrorCode, message string) *AppError {
	return &AppError{
		HttpStatus: httpStatus,
		Code:       code,
		Message:    message,
	}
}

// 甯哥敤閿欒蹇嵎鍒涘缓
func BadRequest(code ErrorCode, message string) *AppError {
	return NewError(http.StatusBadRequest, code, message)
}

func Unauthorized(message string) *AppError {
	return NewError(http.StatusUnauthorized, ErrUnauthorized, message)
}

func Forbidden(message string) *AppError {
	return NewError(http.StatusForbidden, ErrForbidden, message)
}

func NotFound(code ErrorCode, message string) *AppError {
	return NewError(http.StatusNotFound, code, message)
}

func InternalError(message string) *AppError {
	return NewError(http.StatusInternalServerError, ErrInternal, message)
}
