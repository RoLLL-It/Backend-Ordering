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
	rawToken := extractRefreshCookie(args)
	if rawToken != "" {
		_ = authSvc.Logout(context.Background(), rawToken)
	}
	resp := fn.NoContent()
	// Clear the cookie
	headers, _ := resp["headers"].(map[string]interface{})
	if headers == nil {
		headers = map[string]interface{}{}
	}
	headers["Set-Cookie"] = "refresh_token=; HttpOnly; SameSite=Lax; Path=/api/v1/auth; Max-Age=-1"
	resp["headers"] = headers
	return resp
}

func extractRefreshCookie(args map[string]interface{}) string {
	headers, _ := args["__ow_headers"].(map[string]interface{})
	cookieHeader, _ := headers["cookie"].(string)
	if cookieHeader == "" {
		cookieHeader, _ = headers["Cookie"].(string)
	}
	const prefix = "refresh_token="
	for i := 0; i <= len(cookieHeader)-len(prefix); i++ {
		if cookieHeader[i:i+len(prefix)] == prefix {
			val := cookieHeader[i+len(prefix):]
			for j, c := range val {
				if c == ';' {
					return val[:j]
				}
			}
			return val
		}
	}
	return ""
}

func main() {}
