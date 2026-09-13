package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/token"
)

type userIDKey struct{}
type userRoleKey struct{}

// Authenticate parses the Bearer token and injects user claims into context.
// Returns 401 if token is missing or invalid.
func Authenticate(tm *token.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("Authorization")
			if !strings.HasPrefix(raw, "Bearer ") {
				errs.WriteError(w, r, errs.Unauthorized("missing or invalid authorization header"))
				return
			}
			tokenStr := strings.TrimPrefix(raw, "Bearer ")
			claims, err := tm.Parse(tokenStr)
			if err != nil {
				errs.WriteError(w, r, errs.TokenExpired())
				return
			}
			uid, err := uuid.Parse(claims.Subject)
			if err != nil {
				errs.WriteError(w, r, errs.Unauthorized("invalid token subject"))
				return
			}
			ctx := context.WithValue(r.Context(), userIDKey{}, uid)
			ctx = context.WithValue(ctx, userRoleKey{}, claims.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserID returns the authenticated user's UUID from context.
func GetUserID(ctx context.Context) uuid.UUID {
	if id, ok := ctx.Value(userIDKey{}).(uuid.UUID); ok {
		return id
	}
	return uuid.Nil
}

// GetUserRole returns the authenticated user's role from context.
func GetUserRole(ctx context.Context) domain.Role {
	if role, ok := ctx.Value(userRoleKey{}).(domain.Role); ok {
		return role
	}
	return ""
}
