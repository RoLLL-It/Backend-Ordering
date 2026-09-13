package errs

import (
	"encoding/json"
	"net/http"

	"github.com/RoLLL-It/Backend-Ordering/internal/http/middleware"
)

// AppError is the canonical error returned by all handlers.
type AppError struct {
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	HTTPStatus int            `json:"-"`
}

func (e *AppError) Error() string { return e.Message }

type envelope struct {
	Error *errorBody `json:"error"`
}

type errorBody struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Details   map[string]any `json:"details,omitempty"`
	RequestID string         `json:"request_id,omitempty"`
}

// WriteError writes a JSON error response.
func WriteError(w http.ResponseWriter, r *http.Request, err *AppError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.HTTPStatus)
	_ = json.NewEncoder(w).Encode(envelope{Error: &errorBody{
		Code:      err.Code,
		Message:   err.Message,
		Details:   err.Details,
		RequestID: middleware.GetRequestID(r.Context()),
	}})
}

// --- constructors ---

func Validation(details map[string]string) *AppError {
	d := make(map[string]any, len(details))
	for k, v := range details {
		d[k] = v
	}
	return &AppError{Code: "VALIDATION_ERROR", Message: "Validation failed.", Details: d, HTTPStatus: http.StatusBadRequest}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Code: "UNAUTHORIZED", Message: msg, HTTPStatus: http.StatusUnauthorized}
}

func TokenExpired() *AppError {
	return &AppError{Code: "TOKEN_EXPIRED", Message: "Your session has expired. Please log in again.", HTTPStatus: http.StatusUnauthorized}
}

func InvalidCredentials() *AppError {
	return &AppError{Code: "INVALID_CREDENTIALS", Message: "Invalid credentials.", HTTPStatus: http.StatusUnauthorized}
}

func Forbidden(msg string) *AppError {
	return &AppError{Code: "FORBIDDEN", Message: msg, HTTPStatus: http.StatusForbidden}
}

func NotFound(resource string) *AppError {
	return &AppError{Code: "NOT_FOUND", Message: resource + " not found.", HTTPStatus: http.StatusNotFound}
}

func EmailTaken() *AppError {
	return &AppError{Code: "EMAIL_TAKEN", Message: "This email is already registered.", HTTPStatus: http.StatusConflict}
}

func PhoneTaken() *AppError {
	return &AppError{Code: "PHONE_TAKEN", Message: "This phone number is already registered.", HTTPStatus: http.StatusConflict}
}

func ItemsUnavailable(items []string) *AppError {
	return &AppError{
		Code:       "ITEMS_UNAVAILABLE",
		Message:    "One or more items are no longer available.",
		Details:    map[string]any{"items": items},
		HTTPStatus: http.StatusConflict,
	}
}

func SlotFull() *AppError {
	return &AppError{Code: "SLOT_FULL", Message: "This delivery slot is now full. Please choose another.", HTTPStatus: http.StatusConflict}
}

func SlotExpired() *AppError {
	return &AppError{Code: "SLOT_EXPIRED", Message: "The order cutoff for this slot has passed.", HTTPStatus: http.StatusConflict}
}

func DeliveryDisabled() *AppError {
	return &AppError{Code: "DELIVERY_DISABLED", Message: "Delivery is currently unavailable. Please try again later.", HTTPStatus: http.StatusConflict}
}

func InvalidTransition() *AppError {
	return &AppError{Code: "INVALID_TRANSITION", Message: "This status change is not allowed.", HTTPStatus: http.StatusConflict}
}

func CancelWindowPassed() *AppError {
	return &AppError{Code: "CANCEL_WINDOW_PASSED", Message: "The cancellation window has passed.", HTTPStatus: http.StatusConflict}
}

func ReviewExists() *AppError {
	return &AppError{Code: "REVIEW_EXISTS", Message: "You have already reviewed this order.", HTTPStatus: http.StatusConflict}
}

func ReviewNotAllowed() *AppError {
	return &AppError{Code: "REVIEW_NOT_ALLOWED", Message: "You cannot review this order.", HTTPStatus: http.StatusConflict}
}

func ReviewLocked() *AppError {
	return &AppError{Code: "REVIEW_LOCKED", Message: "The edit window for this review has passed.", HTTPStatus: http.StatusConflict}
}

func RateLimited() *AppError {
	return &AppError{Code: "RATE_LIMITED", Message: "Too many attempts. Please try again later.", HTTPStatus: http.StatusTooManyRequests}
}

func Internal(msg string) *AppError {
	return &AppError{Code: "INTERNAL", Message: msg, HTTPStatus: http.StatusInternalServerError}
}
