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
	once     sync.Once
	orderSvc *service.OrderService
	initErr  error
)

func init() {
	once.Do(func() {
		pool, _, _, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		locSvc := service.NewLocationService(pool)
		settingsSvc := service.NewSettingsService(pool)
		orderSvc = service.NewOrderService(
			pool,
			repo.NewOrderRepo(pool),
			repo.NewMenuRepo(pool),
			repo.NewSlotRepo(pool),
			locSvc,
			settingsSvc,
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
	activeOnly := fn.QueryBool(args, "active_only", false)
	page := fn.QueryInt(args, "page", 1)
	pageSize := fn.QueryInt(args, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	orders, total, err := orderSvc.ListMyOrders(context.Background(), userID, activeOnly, page, pageSize)
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not load orders")
	}
	return fn.OK(map[string]interface{}{
		"data":      orders,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func main() {}
