package mock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type OrderRepo struct {
	InsertTxFn            func(ctx context.Context, tx pgx.Tx, o *domain.Order, items []domain.OrderItem) error
	InsertEventTxFn       func(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, from *domain.OrderStatus, to domain.OrderStatus, actorID *uuid.UUID, note string) error
	GetByIDWithDetailsFn  func(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	ListByUserFn          func(ctx context.Context, userID uuid.UUID, activeOnly bool, page, pageSize int) ([]*domain.Order, int, error)
	UpdateStatusTxFn      func(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus) error
	SetDeliveredTxFn      func(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	SetCancelledTxFn      func(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus, reason string) error
	AdminListFn           func(ctx context.Context, status *domain.OrderStatus, date *time.Time, locationID *uuid.UUID, page, pageSize int) ([]*domain.Order, int, error)
	NextShortCodeTxFn     func(ctx context.Context, tx pgx.Tx) (string, error)
}

func (m *OrderRepo) InsertTx(ctx context.Context, tx pgx.Tx, o *domain.Order, items []domain.OrderItem) error {
	if m.InsertTxFn == nil {
		panic("mock.OrderRepo.InsertTxFn not set")
	}
	return m.InsertTxFn(ctx, tx, o, items)
}

func (m *OrderRepo) InsertEventTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, from *domain.OrderStatus, to domain.OrderStatus, actorID *uuid.UUID, note string) error {
	if m.InsertEventTxFn == nil {
		return nil // audit events are optional in tests
	}
	return m.InsertEventTxFn(ctx, tx, orderID, from, to, actorID, note)
}

func (m *OrderRepo) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	if m.GetByIDWithDetailsFn == nil {
		panic("mock.OrderRepo.GetByIDWithDetailsFn not set")
	}
	return m.GetByIDWithDetailsFn(ctx, id)
}

func (m *OrderRepo) ListByUser(ctx context.Context, userID uuid.UUID, activeOnly bool, page, pageSize int) ([]*domain.Order, int, error) {
	if m.ListByUserFn == nil {
		panic("mock.OrderRepo.ListByUserFn not set")
	}
	return m.ListByUserFn(ctx, userID, activeOnly, page, pageSize)
}

func (m *OrderRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus) error {
	if m.UpdateStatusTxFn == nil {
		panic("mock.OrderRepo.UpdateStatusTxFn not set")
	}
	return m.UpdateStatusTxFn(ctx, tx, id, status)
}

func (m *OrderRepo) SetDeliveredTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	if m.SetDeliveredTxFn == nil {
		panic("mock.OrderRepo.SetDeliveredTxFn not set")
	}
	return m.SetDeliveredTxFn(ctx, tx, id)
}

func (m *OrderRepo) SetCancelledTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus, reason string) error {
	if m.SetCancelledTxFn == nil {
		panic("mock.OrderRepo.SetCancelledTxFn not set")
	}
	return m.SetCancelledTxFn(ctx, tx, id, status, reason)
}

func (m *OrderRepo) AdminList(ctx context.Context, status *domain.OrderStatus, date *time.Time, locationID *uuid.UUID, page, pageSize int) ([]*domain.Order, int, error) {
	if m.AdminListFn == nil {
		panic("mock.OrderRepo.AdminListFn not set")
	}
	return m.AdminListFn(ctx, status, date, locationID, page, pageSize)
}

func (m *OrderRepo) NextShortCodeTx(ctx context.Context, tx pgx.Tx) (string, error) {
	if m.NextShortCodeTxFn == nil {
		return "RIT-A00", nil // default stub
	}
	return m.NextShortCodeTxFn(ctx, tx)
}
