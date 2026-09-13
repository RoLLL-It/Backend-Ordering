package middleware

import (
	"net/http"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
)

// RequireRole returns a middleware that enforces one of the allowed roles.
func RequireRole(roles ...domain.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			userRole := GetUserRole(req.Context())
			for _, allowed := range roles {
				if userRole == allowed {
					next.ServeHTTP(w, req)
					return
				}
			}
			errs.WriteError(w, req, errs.Forbidden("insufficient permissions"))
		})
	}
}

// StaffOrAdmin is shorthand for RequireRole(STAFF, ADMIN).
func StaffOrAdmin(next http.Handler) http.Handler {
	return RequireRole(domain.RoleStaff, domain.RoleAdmin)(next)
}

// AdminOnly is shorthand for RequireRole(ADMIN).
func AdminOnly(next http.Handler) http.Handler {
	return RequireRole(domain.RoleAdmin)(next)
}

