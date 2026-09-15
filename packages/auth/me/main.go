package main

import (
	"context"
	"sync"

	"github.com/google/uuid"

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
	_, tokenMgr, _, _ := fn.Bootstrap()
	claims, errResp := fn.RequireAuth(args, tokenMgr)
	if errResp != nil {
		return errResp
	}
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return fn.Err(401, "UNAUTHORIZED", "invalid token subject")
	}
	u, err := authSvc.GetMe(context.Background(), userID)
	if err != nil {
		return fn.DomainError(err)
	}
	return fn.OK(u)
}

func main() {}
