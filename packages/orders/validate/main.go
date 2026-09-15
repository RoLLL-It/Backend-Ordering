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
	once    sync.Once
	menuSvc *service.MenuService
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
		menuSvc = service.NewMenuService(repo.NewMenuRepo(pool))
		locSvc = service.NewLocationService(pool)
	})
}

func Main(args map[string]interface{}) map[string]interface{} {
	if initErr != nil {
		return fn.Err(500, "INIT_FAILED", "service initialization failed")
	}
	var body struct {
		Items []struct {
			MenuItemID string `json:"menu_item_id"`
			Quantity   int    `json:"quantity"`
		} `json:"items"`
		LocationID string `json:"location_id"`
	}
	if err := fn.ParseBody(args, &body); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	locID, err := uuid.Parse(body.LocationID)
	if err != nil {
		return fn.Err(400, "INVALID_PARAMS", "location_id must be a valid UUID")
	}
	cartItems := make([]service.CartItem, 0, len(body.Items))
	for _, it := range body.Items {
		id, err := uuid.Parse(it.MenuItemID)
		if err != nil {
			continue
		}
		cartItems = append(cartItems, service.CartItem{MenuItemID: id, Quantity: it.Quantity})
	}
	result, err := menuSvc.ValidateCart(context.Background(), cartItems, locID)
	if err != nil {
		return fn.Err(500, "INTERNAL", "could not validate cart")
	}
	loc, _ := locSvc.GetByID(context.Background(), locID)
	if loc != nil {
		result.DeliveryFeePaise = loc.DeliveryFeePaise
		result.TotalPaise = result.SubtotalPaise + loc.DeliveryFeePaise
		result.DeliveryEnabled = loc.DeliveryEnabled
	}
	return fn.OK(result)
}

func main() {}
