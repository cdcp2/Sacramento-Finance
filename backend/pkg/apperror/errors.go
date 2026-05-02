package apperror

import (
	"errors"
	"net/http"
)

// AppError is a typed application error with an associated HTTP status.
type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

// Common errors
var (
	ErrNotFound         = New(http.StatusNotFound, "NOT_FOUND", "resource not found")
	ErrUnauthorized     = New(http.StatusUnauthorized, "UNAUTHORIZED", "authentication required")
	ErrForbidden        = New(http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
	ErrBadRequest       = New(http.StatusBadRequest, "BAD_REQUEST", "invalid request")
	ErrConflict         = New(http.StatusConflict, "CONFLICT", "resource already exists")
	ErrInternal         = New(http.StatusInternalServerError, "INTERNAL_ERROR", "internal server error")
	ErrInvalidToken     = New(http.StatusUnauthorized, "INVALID_TOKEN", "invalid or expired token")

	// Domain-specific
	ErrDuplicateCedula  = New(http.StatusConflict, "DUPLICATE_CEDULA", "a user with this cedula already exists")
	ErrDuplicateEmail   = New(http.StatusConflict, "DUPLICATE_EMAIL", "a user with this email already exists")
	ErrInvalidPassword  = New(http.StatusUnauthorized, "INVALID_PASSWORD", "invalid credentials")
	ErrFundNotFound     = New(http.StatusNotFound, "FUND_NOT_FOUND", "fund not found")
	ErrNotFundMember    = New(http.StatusForbidden, "NOT_FUND_MEMBER", "you are not a member of this fund")
	ErrNotFundAdmin     = New(http.StatusForbidden, "NOT_FUND_ADMIN", "admin role required")
	ErrFundNotActive    = New(http.StatusBadRequest, "FUND_NOT_ACTIVE", "fund is not active")
	ErrAlreadyMember    = New(http.StatusConflict, "ALREADY_MEMBER", "user is already a member of this fund")
	ErrInvalidFundState = New(http.StatusBadRequest, "INVALID_FUND_STATE", "invalid fund state transition")
	ErrPaymentNotFound  = New(http.StatusNotFound, "PAYMENT_NOT_FOUND", "payment not found")
	ErrPaymentAlreadyPaid = New(http.StatusConflict, "PAYMENT_ALREADY_PAID", "payment has already been recorded")
)

// HTTPStatus returns the HTTP status for any error.
// Falls back to 500 for unknown errors.
func HTTPStatus(err error) int {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Status
	}
	return http.StatusInternalServerError
}

// As unwraps to AppError if possible.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	return appErr, errors.As(err, &appErr)
}
