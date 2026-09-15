package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/http/reqctx"
)

// RequestID injects a unique request ID into the context and response header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = "req_" + uuid.New().String()
		}
		ctx := reqctx.Set(r.Context(), id)
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID returns the request ID from context.
func GetRequestID(ctx context.Context) string {
	return reqctx.Get(ctx)
}
