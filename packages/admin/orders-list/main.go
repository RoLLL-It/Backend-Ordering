package main

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
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
	if errResp = fn.RequireRole(claims, domain.RoleAdmin, domain.RoleStaff); errResp != nil {
		return errResp
	}
	page := fn.QueryInt(args, "page", 1)
	pageSize := fn.QueryInt(args, "page_size", 20)
	var status *domain.OrderStatus
	if s := fn.QueryString(args, "status"); s != "" {
		st := domain.OrderStatus(s)
		status = &st
	}
	var date *time.Time
	if d := fn.QueryString(args, "date"); d != "" {
		t, err := time.Parse("2006-01-02", d)
		if err == nil {
			date = &t
		}
	}
	var locationID *uuid.UUID
	if l := fn.QueryString(args, "location_id"); l != "" {
		id, err := uuid.Parse(l)
		if err == nil {
			locationID = &id
		}
	}
	orders, total, err := orderSvc.AdminList(context.Background(), status, date, locationID, page, pageSize)
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
