package main

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

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
	locationIDStr := fn.QueryString(args, "location_id")
	locationID, err := uuid.Parse(locationIDStr)
	if err != nil {
		return fn.Err(400, "INVALID_PARAMS", "location_id must be a valid UUID")
	}
	dateStr := fn.QueryString(args, "date")
	var date time.Time
	if dateStr != "" {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fn.Err(400, "INVALID_PARAMS", "date must be in YYYY-MM-DD format")
		}
	} else {
		date = time.Now().UTC().Truncate(24 * time.Hour)
	}
	slots, err := slotRepo.ListByLocationAndDate(context.Background(), locationID, date)
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not load slots")
	}
	return fn.OK(slots)
}

func main() {}
