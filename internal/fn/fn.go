package fn

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/token"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

// ParseBody parses args["__ow_body"] (JSON string) into dst.
func ParseBody(args map[string]interface{}, dst interface{}) error {
	raw, _ := args["__ow_body"].(string)
	if raw == "" {
		if b, ok := args["body"].(string); ok {
			raw = b
		}
	}
	if raw == "" {
		return errors.New("empty body")
	}
	return json.Unmarshal([]byte(raw), dst)
}

// ExtractBearer gets the Bearer token from args["__ow_headers"].
func ExtractBearer(args map[string]interface{}) string {
	headers, _ := args["__ow_headers"].(map[string]interface{})
	auth, _ := headers["authorization"].(string)
	if auth == "" {
		// Also try lowercase
		auth, _ = headers["Authorization"].(string)
	}
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// RequireAuth validates the Bearer JWT; returns claims or an error response.
func RequireAuth(args map[string]interface{}, mgr *token.Manager) (*token.Claims, map[string]interface{}) {
	raw := ExtractBearer(args)
	if raw == "" {
		return nil, Err(401, "UNAUTHORIZED", "missing or invalid authorization header")
	}
	claims, err := mgr.Parse(raw)
	if err != nil {
		return nil, Err(401, "TOKEN_EXPIRED", "your session has expired")
	}
	return claims, nil
}

// RequireRole checks that claims.Role is in the allowed set.
func RequireRole(claims *token.Claims, roles ...domain.Role) map[string]interface{} {
	for _, r := range roles {
		if claims.Role == r {
			return nil
		}
	}
	return Err(403, "FORBIDDEN", "insufficient permissions")
}

// ParseUUID reads a string from args[key] and returns a UUID.
func ParseUUID(args map[string]interface{}, key string) (uuid.UUID, map[string]interface{}) {
	s, _ := args[key].(string)
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, Err(400, "INVALID_ID", key+" must be a valid UUID")
	}
	return id, nil
}

// QueryString reads args[key] as a string.
func QueryString(args map[string]interface{}, key string) string {
	s, _ := args[key].(string)
	return s
}

// QueryBool reads args[key] as a bool.
func QueryBool(args map[string]interface{}, key string, def bool) bool {
	switch v := args[key].(type) {
	case bool:
		return v
	case string:
		return v == "true" || v == "1"
	}
	return def
}

// QueryInt reads args[key] as an int.
func QueryInt(args map[string]interface{}, key string, def int) int {
	switch v := args[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return def
}

// OK returns a 200 response.
func OK(body interface{}) map[string]interface{} { return BuildResponse(200, body) }

// Created returns a 201 response.
func Created(body interface{}) map[string]interface{} { return BuildResponse(201, body) }

// NoContent returns a 204.
func NoContent() map[string]interface{} {
	return map[string]interface{}{"statusCode": 204, "body": ""}
}

// BuildResponse builds a DO Functions response map.
func BuildResponse(statusCode int, body interface{}) map[string]interface{} {
	b, _ := json.Marshal(body)
	return map[string]interface{}{
		"statusCode": statusCode,
		"headers":    map[string]interface{}{"Content-Type": "application/json"},
		"body":       string(b),
	}
}

// DomainError maps a domain sentinel error to an HTTP-status response.
func DomainError(err error) map[string]interface{} {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return Err(404, "NOT_FOUND", "resource not found")
	case errors.Is(err, domain.ErrUnauthorized):
		return Err(401, "UNAUTHORIZED", "unauthorized")
	case errors.Is(err, domain.ErrForbidden):
		return Err(403, "FORBIDDEN", "forbidden")
	case errors.Is(err, domain.ErrEmailTaken):
		return Err(409, "EMAIL_TAKEN", "this email is already registered")
	case errors.Is(err, domain.ErrPhoneTaken):
		return Err(409, "PHONE_TAKEN", "this phone number is already registered")
	case errors.Is(err, domain.ErrInvalidCredentials):
		return Err(401, "INVALID_CREDENTIALS", "invalid credentials")
	case errors.Is(err, domain.ErrSlotFull):
		return Err(409, "SLOT_FULL", "this delivery slot is now full")
	case errors.Is(err, domain.ErrSlotExpired):
		return Err(409, "SLOT_EXPIRED", "the order cutoff for this slot has passed")
	case errors.Is(err, domain.ErrDeliveryDisabled):
		return Err(409, "DELIVERY_DISABLED", "delivery is currently unavailable")
	case errors.Is(err, domain.ErrInvalidTransition):
		return Err(409, "INVALID_TRANSITION", "this status change is not allowed")
	case errors.Is(err, domain.ErrCancelWindowPassed):
		return Err(409, "CANCEL_WINDOW_PASSED", "the cancellation window has passed")
	case errors.Is(err, domain.ErrReviewExists):
		return Err(409, "REVIEW_EXISTS", "you have already reviewed this order")
	case errors.Is(err, domain.ErrReviewNotAllowed):
		return Err(409, "REVIEW_NOT_ALLOWED", "you cannot review this order")
	case errors.Is(err, domain.ErrReviewLocked):
		return Err(409, "REVIEW_LOCKED", "the edit window for this review has passed")
	}
	// Check for service-layer typed errors
	if fields, ok := service.IsValidationError(err); ok {
		details := map[string]interface{}{}
		for k, v := range fields {
			details[k] = v
		}
		b, _ := json.Marshal(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "VALIDATION_ERROR",
				"message": "validation failed",
				"details": details,
			},
		})
		return map[string]interface{}{
			"statusCode": 400,
			"headers":    map[string]interface{}{"Content-Type": "application/json"},
			"body":       string(b),
		}
	}
	if items, ok := service.IsItemsUnavailable(err); ok {
		b, _ := json.Marshal(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "ITEMS_UNAVAILABLE",
				"message": "one or more items are no longer available",
				"details": map[string]interface{}{"items": items},
			},
		})
		return map[string]interface{}{
			"statusCode": 409,
			"headers":    map[string]interface{}{"Content-Type": "application/json"},
			"body":       string(b),
		}
	}
	return Err(500, "INTERNAL", "an internal error occurred")
}

// Err builds a standard error response.
func Err(statusCode int, code, message string) map[string]interface{} {
	b, _ := json.Marshal(map[string]interface{}{
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	})
	return map[string]interface{}{
		"statusCode": statusCode,
		"headers":    map[string]interface{}{"Content-Type": "application/json"},
		"body":       string(b),
	}
}

// WithCORS adds CORS headers to any response.
func WithCORS(resp map[string]interface{}, origin string) map[string]interface{} {
	headers, _ := resp["headers"].(map[string]interface{})
	if headers == nil {
		headers = map[string]interface{}{}
	}
	if origin == "" {
		origin = "*"
	}
	headers["Access-Control-Allow-Origin"] = origin
	headers["Access-Control-Allow-Headers"] = "Authorization, Content-Type"
	headers["Access-Control-Allow-Methods"] = "GET, POST, PUT, PATCH, DELETE, OPTIONS"
	resp["headers"] = headers
	return resp
}

// SetRefreshCookie adds a Set-Cookie header for the refresh token.
func SetRefreshCookie(resp map[string]interface{}, raw string, maxAge int, secure bool) map[string]interface{} {
	headers, _ := resp["headers"].(map[string]interface{})
	if headers == nil {
		headers = map[string]interface{}{}
	}
	sameSite := "Lax"
	secureFlag := ""
	if secure {
		secureFlag = "; Secure"
	}
	cookie := fmt.Sprintf("refresh_token=%s; HttpOnly%s; SameSite=%s; Path=/api/v1/auth; Max-Age=%d", raw, secureFlag, sameSite, maxAge)
	headers["Set-Cookie"] = cookie
	resp["headers"] = headers
	return resp
}
