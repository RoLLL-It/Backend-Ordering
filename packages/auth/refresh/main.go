package main

import (
	"context"
	"sync"

	"github.com/RoLLL-It/Backend-Ordering/internal/fn"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

var (
	once    sync.Once
	authSvc *service.AuthService
	initErr error
)

func init() {
	once.Do(func() {
		pool, tokenMgr, cfg, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		authSvc = service.NewAuthService(
			repo.NewUserRepo(pool),
			tokenMgr,
			cfg.BcryptCost,
			cfg.RefreshTTL,
		)
	})
}

func Main(args map[string]interface{}) map[string]interface{} {
	if initErr != nil {
		return fn.Err(500, "INIT_FAILED", "service initialization failed")
	}
	// Extract refresh token from cookie header
	rawToken := extractRefreshCookie(args)
	if rawToken == "" {
		return fn.Err(401, "UNAUTHORIZED", "refresh token cookie missing")
	}
	result, err := authSvc.Refresh(context.Background(), rawToken)
	if err != nil {
		return fn.Err(401, "UNAUTHORIZED", "invalid or expired refresh token")
	}
	_, _, cfg, _ := fn.Bootstrap()
	resp := fn.OK(map[string]interface{}{
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
	return fn.SetRefreshCookie(resp, result.RefreshRaw, int(cfg.RefreshTTL.Seconds()), cfg.Env == "production")
}

func extractRefreshCookie(args map[string]interface{}) string {
	headers, _ := args["__ow_headers"].(map[string]interface{})
	cookieHeader, _ := headers["cookie"].(string)
	if cookieHeader == "" {
		cookieHeader, _ = headers["Cookie"].(string)
	}
	const prefix = "refresh_token="
	start := 0
	for {
		idx := indexOf(cookieHeader[start:], prefix)
		if idx < 0 {
			break
		}
		idx += start
		val := cookieHeader[idx+len(prefix):]
		end := indexOf(val, ";")
		if end >= 0 {
			return val[:end]
		}
		return val
	}
	return ""
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func main() {}
