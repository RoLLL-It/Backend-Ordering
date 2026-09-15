package main

import (
	"context"
	"sync"

	"github.com/RoLLL-It/Backend-Ordering/internal/fn"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

var (
	once    sync.Once
	locSvc  *service.LocationService
	initErr error
)

func init() {
	once.Do(func() {
		pool, _, _, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		locSvc = service.NewLocationService(pool)
	})
}

func Main(args map[string]interface{}) map[string]interface{} {
	if initErr != nil {
		return fn.Err(500, "INIT_FAILED", "service initialization failed")
	}
	locs, err := locSvc.List(context.Background())
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not load locations")
	}
	return fn.OK(locs)
}

func main() {}
