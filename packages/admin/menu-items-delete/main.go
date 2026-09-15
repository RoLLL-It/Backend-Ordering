package main

import (
	"context"
	"sync"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/fn"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

var (
	once    sync.Once
	menuSvc *service.MenuService
	initErr error
)

func init() {
	once.Do(func() {
		pool, _, _, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		menuSvc = service.NewMenuService(repo.NewMenuRepo(pool))
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
	if errResp = fn.RequireRole(claims, domain.RoleAdmin); errResp != nil {
		return errResp
	}
	id, errResp := fn.ParseUUID(args, "id")
	if errResp != nil {
		return errResp
	}
	if err := menuSvc.SoftDelete(context.Background(), id); err != nil {
		return fn.DomainError(err)
	}
	return fn.NoContent()
}

func main() {}
