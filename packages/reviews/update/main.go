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
	reviewID, errResp := fn.ParseUUID(args, "id")
	if errResp != nil {
		return errResp
	}
	var body struct {
		Rating  int16  `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := fn.ParseBody(args, &body); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	if err := reviewSvc.Update(context.Background(), reviewID, callerID, body.Rating, body.Comment); err != nil {
		return fn.DomainError(err)
	}
	return fn.NoContent()
}

func main() {}
