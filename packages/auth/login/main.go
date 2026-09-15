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
	var input service.LoginInput
	if err := fn.ParseBody(args, &input); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	result, err := authSvc.Login(context.Background(), input)
	if err != nil {
		return fn.DomainError(err)
	}
	_, _, cfg, _ := fn.Bootstrap()
	resp := fn.OK(map[string]interface{}{
		"user":         result.User,
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
	return fn.SetRefreshCookie(resp, result.RefreshRaw, int(cfg.RefreshTTL.Seconds()), cfg.Env == "production")
}

func main() {}
