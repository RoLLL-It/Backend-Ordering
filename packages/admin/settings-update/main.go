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
	if errResp = fn.RequireRole(claims, domain.RoleAdmin); errResp != nil {
		return errResp
	}
	var body struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := fn.ParseBody(args, &body); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	if body.Key == "" {
		return fn.Err(400, "INVALID_PARAMS", "key is required")
	}
	if err := settingsSvc.Set(context.Background(), body.Key, body.Value); err != nil {
		return fn.Err(500, "INTERNAL", "could not update setting")
	}
	return fn.NoContent()
}

func main() {}
