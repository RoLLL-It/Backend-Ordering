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
	id, errResp := fn.ParseUUID(args, "id")
	if errResp != nil {
		return errResp
	}
	var body struct {
		Role     *string `json:"role"`
		IsActive *bool   `json:"is_active"`
	}
	if err := fn.ParseBody(args, &body); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	user, err := userRepo.GetByID(context.Background(), id)
	if err != nil {
		return fn.DomainError(err)
	}
	if body.Role != nil {
		user.Role = domain.Role(*body.Role)
	}
	if body.IsActive != nil {
		user.IsActive = *body.IsActive
	}
	if err := userRepo.Update(context.Background(), user); err != nil {
		return fn.Err(500, "INTERNAL", "could not update user")
	}
	return fn.OK(map[string]interface{}{
		"id":        user.ID,
		"name":      user.Name,
		"email":     user.Email,
		"phone":     user.Phone,
		"role":      user.Role,
		"is_active": user.IsActive,
	})
}

func main() {}
