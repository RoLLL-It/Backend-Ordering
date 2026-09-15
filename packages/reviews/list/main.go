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
	page := fn.QueryInt(args, "page", 1)
	pageSize := fn.QueryInt(args, "page_size", 20)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	var menuItemID *uuid.UUID
	if s := fn.QueryString(args, "menu_item_id"); s != "" {
		id, err := uuid.Parse(s)
		if err == nil {
			menuItemID = &id
		}
	}
	reviews, total, err := reviewSvc.List(context.Background(), menuItemID, page, pageSize)
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not load reviews")
	}
	return fn.OK(map[string]interface{}{
		"data":  reviews,
		"total": total,
		"page":  page,
	})
}

func main() {}
