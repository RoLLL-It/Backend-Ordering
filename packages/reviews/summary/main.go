package main

import (
	"context"
	"sync"

	"github.com/RoLLL-It/Backend-Ordering/internal/fn"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

var (
	once      sync.Once
	reviewSvc *service.ReviewService
	initErr   error
)

func init() {
	once.Do(func() {
		pool, _, _, err := fn.Bootstrap()
		if err != nil {
			initErr = err
			return
		}
		reviewSvc = service.NewReviewService(
			repo.NewReviewRepo(pool),
			repo.NewMenuRepo(pool),
			repo.NewOrderRepo(pool),
		)
	})
}

func Main(args map[string]interface{}) map[string]interface{} {
	if initErr != nil {
		return fn.Err(500, "INIT_FAILED", "service initialization failed")
	}
	avg, total, dist, err := reviewSvc.Summary(context.Background())
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not load review summary")
	}
	return fn.OK(map[string]interface{}{
		"average":      avg,
		"total":        total,
		"distribution": dist,
	})
}

func main() {}
