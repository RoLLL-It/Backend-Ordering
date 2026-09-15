package main

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/fn"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
)

var (
	once     sync.Once
	slotRepo *repo.SlotRepo
	initErr  error
)

func init() {
	once.Do(func() {
		pool, _, _, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		slotRepo = repo.NewSlotRepo(pool)
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
	var slot domain.DeliverySlot
	if err := fn.ParseBody(args, &slot); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	slot.ID = uuid.New()
	if err := slotRepo.Create(context.Background(), &slot); err != nil {
		return fn.DomainError(err)
	}
	return fn.Created(slot)
}

func main() {}
