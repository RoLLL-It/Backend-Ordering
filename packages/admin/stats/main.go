package main

import (
	"context"
	"sync"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/fn"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

var (
	once        sync.Once
	settingsSvc *service.SettingsService
	initErr     error
)

func init() {
	once.Do(func() {
		pool, _, _, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		settingsSvc = service.NewSettingsService(pool)
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
	settings, err := settingsSvc.GetAll(context.Background())
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not load settings")
	}
	return fn.OK(map[string]interface{}{
		"delivery_enabled": settings.DeliveryEnabled,
		"kitchen_open":     settings.KitchenOpen,
		"announcement":     settings.Announcement,
	})
}

func main() {}
