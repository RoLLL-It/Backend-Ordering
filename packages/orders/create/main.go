package main

import (
	"context"
	"sync"

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
	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return fn.Err(401, "UNAUTHORIZED", "invalid token subject")
	}

	var body struct {
		Items []struct {
			MenuItemID string `json:"menu_item_id"`
			Quantity   int    `json:"quantity"`
		} `json:"items"`
		LocationID  string `json:"location_id"`
		SlotID      string `json:"slot_id"`
		Notes       string `json:"notes"`
		PaymentMode string `json:"payment_mode"`
	}
	if err := fn.ParseBody(args, &body); err != nil {
		return fn.Err(400, "INVALID_BODY", "invalid request body")
	}
	locID, err := uuid.Parse(body.LocationID)
	if err != nil {
		return fn.Err(400, "INVALID_PARAMS", "location_id must be a valid UUID")
	}
	slotID, err := uuid.Parse(body.SlotID)
	if err != nil {
		return fn.Err(400, "INVALID_PARAMS", "slot_id must be a valid UUID")
	}
	cartItems := make([]service.CartItem, 0, len(body.Items))
	for _, it := range body.Items {
		id, err := uuid.Parse(it.MenuItemID)
		if err != nil {
			continue
		}
		cartItems = append(cartItems, service.CartItem{MenuItemID: id, Quantity: it.Quantity})
	}
	o, err := orderSvc.Create(context.Background(), userID, service.CreateOrderInput{
		Items:       cartItems,
		LocationID:  locID,
		SlotID:      slotID,
		Notes:       body.Notes,
		PaymentMode: domain.PaymentCOD,
	})
	if err != nil {
		return fn.DomainError(err)
	}
	return fn.Created(o)
}

func main() {}
