package main

import (
	"context"
	"sync"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/fn"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
)

var (
	once     sync.Once
	userRepo *repo.UserRepo
	initErr  error
)

func init() {
	once.Do(func() {
		pool, _, _, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		userRepo = repo.NewUserRepo(pool)
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
	search := fn.QueryString(args, "search")
	page := fn.QueryInt(args, "page", 1)
	pageSize := fn.QueryInt(args, "page_size", 20)
	var role *domain.Role
	if rs := fn.QueryString(args, "role"); rs != "" {
		r := domain.Role(rs)
		role = &r
	}
	users, total, err := userRepo.List(context.Background(), search, role, page, pageSize)
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not load users")
	}
	return fn.OK(map[string]interface{}{
		"data":  users,
		"total": total,
		"page":  page,
	})
}

func main() {}
