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
	var item domain.MenuItem
	if err := fn.ParseBody(args, &item); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	if err := menuSvc.CreateItem(context.Background(), &item); err != nil {
		return fn.DomainError(err)
	}
	return fn.Created(item)
}

func main() {}
