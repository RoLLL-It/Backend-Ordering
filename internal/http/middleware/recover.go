package middleware

import (
	"log/slog"
	"net/http"

	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
)

// Recoverer catches panics, logs them, and returns 500 without leaking stack traces.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered",
					"request_id", GetRequestID(r.Context()),
					"panic", rec,
				)
				errs.WriteError(w, r, errs.Internal("an unexpected error occurred"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
