package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
)

type rateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	rl := &rateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
	go rl.cleanup()
	return rl
}

func (rl *rateLimiter) allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now()
	cutoff := now.Add(-rl.window)
	times := rl.requests[key]
	var valid []time.Time
	for _, t := range times {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	if len(valid) >= rl.limit {
		rl.requests[key] = valid
		return false
	}
	rl.requests[key] = append(valid, now)
	return true
}

func (rl *rateLimiter) cleanup() {
	for {
		time.Sleep(5 * time.Minute)
		rl.mu.Lock()
		cutoff := time.Now().Add(-rl.window)
		for k, times := range rl.requests {
			var valid []time.Time
			for _, t := range times {
				if t.After(cutoff) {
					valid = append(valid, t)
				}
			}
			if len(valid) == 0 {
				delete(rl.requests, k)
			} else {
				rl.requests[k] = valid
			}
		}
		rl.mu.Unlock()
	}
}

var (
	authLimiter   = newRateLimiter(5, 15*time.Minute)
	regLimiter    = newRateLimiter(3, time.Hour)
)

// AuthRateLimit limits login to 5 attempts per identifier per 15 minutes.
func AuthRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if !authLimiter.allow(ip) {
			w.Header().Set("Retry-After", "900")
			errs.WriteError(w, r, errs.RateLimited())
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RegisterRateLimit limits registration to 3 per IP per hour.
func RegisterRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, _ := net.SplitHostPort(r.RemoteAddr)
		if !regLimiter.allow(ip) {
			w.Header().Set("Retry-After", "3600")
			errs.WriteError(w, r, errs.RateLimited())
			return
		}
		next.ServeHTTP(w, r)
	})
}
