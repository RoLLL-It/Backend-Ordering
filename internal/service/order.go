package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo/iface"
)

type OrderService struct {
	db         iface.TxBeginner
	orderRepo  iface.OrderRepo
	menuRepo   iface.MenuRepo
	slotRepo   iface.SlotRepo
	locRepo    *LocationService
	settings   *SettingsService
}

func NewOrderService(db iface.TxBeginner, or iface.OrderRepo, mr iface.MenuRepo, sr iface.SlotRepo, ls *LocationService, ss *SettingsService) *OrderService {
	return &OrderService{db: db, orderRepo: or, menuRepo: mr, slotRepo: sr, locRepo: ls, settings: ss}
}

type CartItem struct {
	MenuItemID uuid.UUID
	Quantity   int
}

type CreateOrderInput struct {
	Items       []CartItem
	LocationID  uuid.UUID
	SlotID      uuid.UUID
	Notes       string
	PaymentMode domain.PaymentMode
}

func (s *OrderService) Create(ctx context.Context, userID uuid.UUID, in CreateOrderInput) (*domain.Order, error) {
	return s.createOrderTx(ctx, userID, in)
}

func (s *OrderService) createOrderTx(ctx context.Context, userID uuid.UUID, in CreateOrderInput) (*domain.Order, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 1. Global delivery flag
	if enabled, _ := s.settings.GetBool(ctx, "delivery_enabled"); !enabled {
		return nil, domain.ErrDeliveryDisabled
	}

	// 2. Location delivery flag
	loc, err := s.locRepo.GetByID(ctx, in.LocationID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if !loc.DeliveryEnabled {
		return nil, domain.ErrDeliveryDisabled
	}

	// 3. LOCK THE SLOT ROW — critical for preventing oversell
	slot, err := s.slotRepo.GetForUpdateTx(ctx, tx, in.SlotID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	if slot.LocationID != in.LocationID {
		return nil, domain.ErrNotFound
	}
	if !slot.IsActive {
		return nil, domain.ErrNotFound
	}
	if time.Now().After(slot.CutoffAt()) {
		return nil, domain.ErrSlotExpired
	}
	if slot.BookedCount >= slot.Capacity {
		return nil, domain.ErrSlotFull
	}

	// 4. Re-read items INSIDE the transaction — never trust client prices
	ids := make([]uuid.UUID, 0, len(in.Items))
	for _, ci := range in.Items {
		ids = append(ids, ci.MenuItemID)
	}
	items, err := s.menuRepo.GetManyByIDsTx(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	if len(items) != len(ids) {
		return nil, domain.ErrNotFound
	}

	itemMap := make(map[uuid.UUID]*domain.MenuItem, len(items))
	for _, it := range items {
		itemMap[it.ID] = it
	}

	var unavailable []string
	for _, it := range items {
		if !it.IsAvailable || !it.IsActive {
			unavailable = append(unavailable, it.Name)
		}
	}
	if len(unavailable) > 0 {
		return nil, &itemsUnavailableError{items: unavailable}
	}

	// 5. Compute totals server-side (integer paise only — never float)
	var subtotal int64
	lines := make([]domain.OrderItem, 0, len(in.Items))
	for _, ci := range in.Items {
		it := itemMap[ci.MenuItemID]
		lineTotal := it.PricePaise * int64(ci.Quantity)
		subtotal += lineTotal
		lines = append(lines, domain.OrderItem{
			MenuItemID:         it.ID,
			NameSnapshot:       it.Name,
			PriceSnapshotPaise: it.PricePaise,
			Quantity:           ci.Quantity,
			LineTotalPaise:     lineTotal,
		})
	}
	total := subtotal + loc.DeliveryFeePaise

	// 6. Generate short code inside the transaction
	shortCode, err := s.orderRepo.NextShortCodeTx(ctx, tx)
	if err != nil {
		return nil, err
	}

	o := &domain.Order{
		ID:               uuid.New(),
		ShortCode:        shortCode,
		UserID:           userID,
		LocationID:       loc.ID,
		SlotID:           slot.ID,
		Status:           domain.StatusPlaced,
		PaymentMode:      domain.PaymentCOD,
		PaymentStatus:    domain.PayPending,
		SubtotalPaise:    subtotal,
		DeliveryFeePaise: loc.DeliveryFeePaise,
		TotalPaise:       total,
		Notes:            in.Notes,
		CancelDeadlineAt: time.Now().Add(2 * time.Minute),
	}

	// 7. Insert order + items
	if err := s.orderRepo.InsertTx(ctx, tx, o, lines); err != nil {
		return nil, err
	}

	// 8. Increment booked_count — CHECK constraint is last defence against oversell
	if err := s.slotRepo.IncrementBookedTx(ctx, tx, slot.ID); err != nil {
		if errors.Is(err, domain.ErrSlotFull) {
			return nil, domain.ErrSlotFull
		}
		return nil, err
	}

	// 9. Audit event
	actorID := userID
	_ = s.orderRepo.InsertEventTx(ctx, tx, o.ID, nil, domain.StatusPlaced, &actorID, "")

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return o, nil
}

func (s *OrderService) GetOrder(ctx context.Context, orderID, callerID uuid.UUID, callerRole domain.Role) (*domain.Order, error) {
	o, err := s.orderRepo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	// Ownership check — return NOT_FOUND not FORBIDDEN so existence isn't leaked
	if callerRole != domain.RoleAdmin && callerRole != domain.RoleStaff && o.UserID != callerID {
		return nil, domain.ErrNotFound
	}
	return o, nil
}

func (s *OrderService) ListMyOrders(ctx context.Context, userID uuid.UUID, activeOnly bool, page, pageSize int) ([]*domain.Order, int, error) {
	return s.orderRepo.ListByUser(ctx, userID, activeOnly, page, pageSize)
}

func (s *OrderService) CancelOrder(ctx context.Context, orderID, callerID uuid.UUID) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	o, err := s.orderRepo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return domain.ErrNotFound
	}
	if o.UserID != callerID {
		return domain.ErrNotFound
	}
	if o.Status != domain.StatusPlaced {
		return domain.ErrInvalidTransition
	}
	if !o.CanCancel() {
		return domain.ErrCancelWindowPassed
	}

	if err := s.orderRepo.SetCancelledTx(ctx, tx, orderID, domain.StatusCancelledUser, "Cancelled by customer"); err != nil {
		return err
	}
	if err := s.slotRepo.DecrementBookedTx(ctx, tx, o.SlotID); err != nil {
		return err
	}
	actorID := callerID
	_ = s.orderRepo.InsertEventTx(ctx, tx, orderID, &o.Status, domain.StatusCancelledUser, &actorID, "")
	return tx.Commit(ctx)
}

func (s *OrderService) AdminUpdateStatus(ctx context.Context, orderID uuid.UUID, newStatus domain.OrderStatus, actorID uuid.UUID, note string) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	o, err := s.orderRepo.GetByIDWithDetails(ctx, orderID)
	if err != nil {
		return domain.ErrNotFound
	}
	if !o.Status.CanTransitionTo(newStatus) {
		return domain.ErrInvalidTransition
	}

	if newStatus == domain.StatusDelivered {
		if err := s.orderRepo.SetDeliveredTx(ctx, tx, orderID); err != nil {
			return err
		}
	} else if newStatus.IsCancelled() {
		if err := s.orderRepo.SetCancelledTx(ctx, tx, orderID, newStatus, note); err != nil {
			return err
		}
		_ = s.slotRepo.DecrementBookedTx(ctx, tx, o.SlotID)
	} else {
		if err := s.orderRepo.UpdateStatusTx(ctx, tx, orderID, newStatus); err != nil {
			return err
		}
	}
	_ = s.orderRepo.InsertEventTx(ctx, tx, orderID, &o.Status, newStatus, &actorID, note)
	return tx.Commit(ctx)
}

func (s *OrderService) AdminList(ctx context.Context, status *domain.OrderStatus, date *time.Time, locationID *uuid.UUID, page, pageSize int) ([]*domain.Order, int, error) {
	return s.orderRepo.AdminList(ctx, status, date, locationID, page, pageSize)
}

type itemsUnavailableError struct{ items []string }

func (e *itemsUnavailableError) Error() string { return "items unavailable" }
func (e *itemsUnavailableError) Items() []string { return e.items }

func IsItemsUnavailable(err error) ([]string, bool) {
	var iue *itemsUnavailableError
	if errors.As(err, &iue) {
		return iue.items, true
	}
	return nil, false
}
