package handler

import (
	"fmt"
	"net/http"
)

// ErrorCode 业务错误码，推荐使用字符串形式，更易读
type ErrorCode string

const (
	// 通用错误
	ErrInternal           ErrorCode = "INTERNAL_ERROR"
	ErrInvalidRequest     ErrorCode = "INVALID_REQUEST"
	ErrUnauthorized       ErrorCode = "UNAUTHORIZED"
	ErrForbidden          ErrorCode = "FORBIDDEN"
	ErrNotFound           ErrorCode = "NOT_FOUND"
	ErrNotImplemented     ErrorCode = "NOT_IMPLEMENTED"
	ErrPrecondition       ErrorCode = "PRECONDITION_FAILED"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"

	// 认证模块
	ErrAuthFailed          ErrorCode = "AUTH_FAILED"
	ErrNotStudent          ErrorCode = "NON_STUDENT_LOGIN_DENIED"
	ErrTokenExpired        ErrorCode = "TOKEN_EXPIRED"
	ErrInvalidToken        ErrorCode = "INVALID_TOKEN"
	ErrPasswordIncorrect   ErrorCode = "PASSWORD_INCORRECT"
	ErrProfileUpdateFailed ErrorCode = "PROFILE_UPDATE_FAILED"

	// 课程/管线模块
	ErrPipelineNotFound ErrorCode = "PIPELINE_NOT_FOUND"
	ErrAlreadyPurchased ErrorCode = "ALREADY_PURCHASED"
	ErrInvalidPipeline  ErrorCode = "INVALID_PIPELINE"

	// 考试模块
	ErrExamNotFound ErrorCode = "EXAM_NOT_FOUND"
	ErrNotEligible  ErrorCode = "NOT_ELIGIBLE"
	ErrSignupFailed ErrorCode = "SIGNUP_FAILED"
	ErrRetakeDenied ErrorCode = "RETAKE_DENIED"

	// 支付/订单模块
	ErrPaymentFailed ErrorCode = "PAYMENT_FAILED"
	ErrOrderNotFound ErrorCode = "ORDER_NOT_FOUND"
	ErrInvalidAmount ErrorCode = "INVALID_AMOUNT"

	// 档案/会员模块
	ErrMembershipExpired ErrorCode = "MEMBERSHIP_EXPIRED"
	ErrRecordRejected    ErrorCode = "RECORD_REJECTED"
)

// AppError 自定义业务错误
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

// 常用错误快捷创建
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
