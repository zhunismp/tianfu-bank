package apperror

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Err     error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%v - %v | original_err: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%v - %v", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func New(code, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

const (
	ErrCodeAccountNotFound   = "ACCOUNT_NOT_FOUND"
	ErrCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrCodeInvalidAmount     = "INVALID_AMOUNT"
	ErrCodeInternal          = "INTERNAL_ERROR"
)

func MapToHTTPStatus(code string) int {
	switch code {
	case ErrCodeAccountNotFound:
		return http.StatusNotFound
	case ErrCodeInsufficientFunds, ErrCodeInvalidAmount:
		return http.StatusUnprocessableEntity
	case ErrCodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
