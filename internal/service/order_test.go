package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/mock"
)

// newTestOrderService builds an OrderService with injected mocks.
// locSvc and settingsSvc are optional (pass nil to skip those checks).
func newTestOrderService(
	db *mock.TxBeginner,
	or *mock.OrderRepo,
	mr *mock.MenuRepo,
	sr *mock.SlotRepo,
) *OrderService {
	// We can't create LocationService/SettingsService without a real pool,
	// so for tests that exercise CancelOrder/AdminUpdateStatus we pass nil.
	return &OrderService{
		db:        db,
		orderRepo: or,
		menuRepo:  mr,
		slotRepo:  sr,
		locRepo:   nil,
		settings:  nil,
	}
}

func noopTx() *mock.Tx {
	return &mock.Tx{
		ExecFn:     func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) { return pgconn.CommandTag{}, nil },
		CommitFn:   func(_ context.Context) error { return nil },
		RollbackFn: func(_ context.Context) error { return nil },
	}
}

func testTxBeginner(tx pgx.Tx) *mock.TxBeginner {
	return &mock.TxBeginner{
		BeginFn: func(_ context.Context) (pgx.Tx, error) { return tx, nil },
	}
}

// --- CancelOrder ---

func TestCancelOrder_Success(t *testing.T) {
	orderID := uuid.New()
	callerID := uuid.New()
	slotID := uuid.New()

	order := &domain.Order{
		ID:               orderID,
		UserID:           callerID,
		SlotID:           slotID,
		Status:           domain.StatusPlaced,
		CancelDeadlineAt: time.Now().Add(5 * time.Minute),
	}

	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, id uuid.UUID) (*domain.Order, error) {
			return order, nil
		},
		SetCancelledTxFn: func(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ domain.OrderStatus, _ string) error {
			return nil
		},
	}
	sr := &mock.SlotRepo{
		DecrementBookedTxFn: func(_ context.Context, _ pgx.Tx, _ uuid.UUID) error { return nil },
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, sr)
	if err := svc.CancelOrder(context.Background(), orderID, callerID); err != nil {
		t.Errorf("CancelOrder() unexpected error: %v", err)
	}
}

func TestCancelOrder_NotFound(t *testing.T) {
	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, &mock.SlotRepo{})
	err := svc.CancelOrder(context.Background(), uuid.New(), uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCancelOrder_WrongCaller(t *testing.T) {
	orderID := uuid.New()
	order := &domain.Order{
		ID:     orderID,
		UserID: uuid.New(), // different from callerID
		Status: domain.StatusPlaced,
	}

	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
			return order, nil
		},
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, &mock.SlotRepo{})
	err := svc.CancelOrder(context.Background(), orderID, uuid.New())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("expected ErrNotFound for wrong caller, got %v", err)
	}
}

func TestCancelOrder_WrongStatus(t *testing.T) {
	orderID := uuid.New()
	callerID := uuid.New()
	order := &domain.Order{
		ID:     orderID,
		UserID: callerID,
		Status: domain.StatusAccepted, // not PLACED
	}

	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
			return order, nil
		},
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, &mock.SlotRepo{})
	err := svc.CancelOrder(context.Background(), orderID, callerID)
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition for wrong status, got %v", err)
	}
}

func TestCancelOrder_WindowPassed(t *testing.T) {
	orderID := uuid.New()
	callerID := uuid.New()
	order := &domain.Order{
		ID:               orderID,
		UserID:           callerID,
		Status:           domain.StatusPlaced,
		CancelDeadlineAt: time.Now().Add(-1 * time.Minute), // past deadline
	}

	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
			return order, nil
		},
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, &mock.SlotRepo{})
	err := svc.CancelOrder(context.Background(), orderID, callerID)
	if !errors.Is(err, domain.ErrCancelWindowPassed) {
		t.Errorf("expected ErrCancelWindowPassed, got %v", err)
	}
}

// --- AdminUpdateStatus ---

func TestAdminUpdateStatus_InvalidTransition(t *testing.T) {
	orderID := uuid.New()
	order := &domain.Order{
		ID:     orderID,
		Status: domain.StatusDelivered, // terminal — no transitions allowed
	}

	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
			return order, nil
		},
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, &mock.SlotRepo{})
	err := svc.AdminUpdateStatus(context.Background(), orderID, domain.StatusAccepted, uuid.New(), "")
	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition, got %v", err)
	}
}

func TestAdminUpdateStatus_Delivered(t *testing.T) {
	orderID := uuid.New()
	slotID := uuid.New()
	order := &domain.Order{
		ID:     orderID,
		SlotID: slotID,
		Status: domain.StatusOutForDelivery,
	}
	delivered := false

	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
			return order, nil
		},
		SetDeliveredTxFn: func(_ context.Context, _ pgx.Tx, _ uuid.UUID) error {
			delivered = true
			return nil
		},
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, &mock.SlotRepo{})
	if err := svc.AdminUpdateStatus(context.Background(), orderID, domain.StatusDelivered, uuid.New(), ""); err != nil {
		t.Fatalf("AdminUpdateStatus() error: %v", err)
	}
	if !delivered {
		t.Error("expected SetDeliveredTx to be called")
	}
}

func TestAdminUpdateStatus_CancelledDecrementsSlot(t *testing.T) {
	orderID := uuid.New()
	slotID := uuid.New()
	order := &domain.Order{
		ID:     orderID,
		SlotID: slotID,
		Status: domain.StatusPlaced,
	}
	decremented := false

	tx := noopTx()
	db := testTxBeginner(tx)
	or := &mock.OrderRepo{
		GetByIDWithDetailsFn: func(_ context.Context, _ uuid.UUID) (*domain.Order, error) {
			return order, nil
		},
		SetCancelledTxFn: func(_ context.Context, _ pgx.Tx, _ uuid.UUID, _ domain.OrderStatus, _ string) error {
			return nil
		},
	}
	sr := &mock.SlotRepo{
		DecrementBookedTxFn: func(_ context.Context, _ pgx.Tx, id uuid.UUID) error {
			if id == slotID {
				decremented = true
			}
			return nil
		},
	}

	svc := newTestOrderService(db, or, &mock.MenuRepo{}, sr)
	if err := svc.AdminUpdateStatus(context.Background(), orderID, domain.StatusCancelledAdmin, uuid.New(), "reason"); err != nil {
		t.Fatalf("AdminUpdateStatus() error: %v", err)
	}
	if !decremented {
		t.Error("expected slot booked count to be decremented on cancellation")
	}
}
