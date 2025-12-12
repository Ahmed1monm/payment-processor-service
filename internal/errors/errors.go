package errors

import (
	"errors"
	"net/http"
)

var (
	// ErrAccountNotFound is returned when an account is not found.
	ErrAccountNotFound = errors.New("account not found")
	// ErrInsufficientBalance is returned when account has insufficient balance.
	ErrInsufficientBalance = errors.New("insufficient balance")
	// ErrInvalidCard is returned when card validation fails.
	ErrInvalidCard = errors.New("invalid card")
	// ErrAccountInactive is returned when account is not active.
	ErrAccountInactive = errors.New("account is not active")
	// ErrInvalidAmount is returned when amount is invalid.
	ErrInvalidAmount = errors.New("invalid amount")
)

// ErrorResponse represents a standardized error response.
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

// HTTPError represents an HTTP error with status code.
type HTTPError struct {
	StatusCode int
	Message    string
	Code       string
}

func (e *HTTPError) Error() string {
	return e.Message
}

// NewHTTPError creates a new HTTP error.
func NewHTTPError(statusCode int, message, code string) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    message,
		Code:       code,
	}
}

// ToErrorResponse converts an HTTPError to ErrorResponse.
func (e *HTTPError) ToErrorResponse() ErrorResponse {
	return ErrorResponse{
		Error: e.Message,
		Code:  e.Code,
	}
}

// MapErrorToHTTP maps domain errors to HTTP errors.
func MapErrorToHTTP(err error) *HTTPError {
	switch err {
	case ErrAccountNotFound:
		return NewHTTPError(http.StatusNotFound, err.Error(), "ACCOUNT_NOT_FOUND")
	case ErrInsufficientBalance:
		return NewHTTPError(http.StatusBadRequest, err.Error(), "INSUFFICIENT_BALANCE")
	case ErrInvalidCard:
		return NewHTTPError(http.StatusBadRequest, err.Error(), "INVALID_CARD")
	case ErrAccountInactive:
		return NewHTTPError(http.StatusBadRequest, err.Error(), "ACCOUNT_INACTIVE")
	case ErrInvalidAmount:
		return NewHTTPError(http.StatusBadRequest, err.Error(), "INVALID_AMOUNT")
	default:
		return NewHTTPError(http.StatusInternalServerError, "internal server error", "INTERNAL_ERROR")
	}
}
