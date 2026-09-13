package domain

import "errors"

// Sentinel errors used by services; handlers map these to HTTP responses.
var (
	ErrNotFound            = errors.New("not found")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrForbidden           = errors.New("forbidden")
	ErrEmailTaken          = errors.New("email already registered")
	ErrPhoneTaken          = errors.New("phone already registered")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrTokenExpired        = errors.New("token expired")
	ErrSlotFull            = errors.New("slot is full")
	ErrSlotExpired         = errors.New("slot cutoff has passed")
	ErrDeliveryDisabled    = errors.New("delivery is disabled")
	ErrItemsUnavailable    = errors.New("one or more items are unavailable")
	ErrInvalidTransition   = errors.New("invalid order status transition")
	ErrCancelWindowPassed  = errors.New("cancel window has passed")
	ErrReviewExists        = errors.New("order already has a review")
	ErrReviewNotAllowed    = errors.New("review not allowed for this order")
	ErrReviewLocked        = errors.New("review edit window has passed")
	ErrValidation          = errors.New("validation error")
)
