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
	_, tokenMgr, _, _ := fn.Bootstrap()
	claims, errResp := fn.RequireAuth(args, tokenMgr)
	if errResp != nil {
		return errResp
	}
	callerID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return fn.Err(401, "UNAUTHORIZED", "invalid token subject")
	}
	var body struct {
		OrderID string `json:"order_id"`
		Rating  int16  `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := fn.ParseBody(args, &body); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	orderID, err := uuid.Parse(body.OrderID)
	if err != nil {
		return fn.Err(400, "INVALID_PARAMS", "order_id must be a valid UUID")
	}
	if err := reviewSvc.Create(context.Background(), callerID, service.CreateReviewInput{
		OrderID: orderID,
		Rating:  body.Rating,
		Comment: body.Comment,
	}); err != nil {
		return fn.DomainError(err)
	}
	return fn.Created(map[string]interface{}{"message": "review created"})
}

func main() {}
