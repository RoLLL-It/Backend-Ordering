package mock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type SlotRepo struct {
	GetByIDFn                func(ctx context.Context, id uuid.UUID) (*domain.DeliverySlot, error)
	GetForUpdateTxFn         func(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.DeliverySlot, error)
	IncrementBookedTxFn      func(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	DecrementBookedTxFn      func(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	ListByLocationAndDateFn  func(ctx context.Context, locationID uuid.UUID, date time.Time) ([]*domain.DeliverySlot, error)
	ListForAdminFn           func(ctx context.Context, date time.Time) ([]*domain.DeliverySlot, error)
	CreateFn                 func(ctx context.Context, s *domain.DeliverySlot) error
	UpdateFn                 func(ctx context.Context, s *domain.DeliverySlot) error
}

func (m *SlotRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DeliverySlot, error) {
	if m.GetByIDFn == nil {
		panic("mock.SlotRepo.GetByIDFn not set")
	}
	return m.GetByIDFn(ctx, id)
}

func (m *SlotRepo) GetForUpdateTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.DeliverySlot, error) {
	if m.GetForUpdateTxFn == nil {
		panic("mock.SlotRepo.GetForUpdateTxFn not set")
	}
	return m.GetForUpdateTxFn(ctx, tx, id)
}

func (m *SlotRepo) IncrementBookedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	if m.IncrementBookedTxFn == nil {
		panic("mock.SlotRepo.IncrementBookedTxFn not set")
	}
	return m.IncrementBookedTxFn(ctx, tx, id)
}

func (m *SlotRepo) DecrementBookedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	if m.DecrementBookedTxFn == nil {
		panic("mock.SlotRepo.DecrementBookedTxFn not set")
	}
	return m.DecrementBookedTxFn(ctx, tx, id)
}

func (m *SlotRepo) ListByLocationAndDate(ctx context.Context, locationID uuid.UUID, date time.Time) ([]*domain.DeliverySlot, error) {
	if m.ListByLocationAndDateFn == nil {
		panic("mock.SlotRepo.ListByLocationAndDateFn not set")
	}
	return m.ListByLocationAndDateFn(ctx, locationID, date)
}

func (m *SlotRepo) ListForAdmin(ctx context.Context, date time.Time) ([]*domain.DeliverySlot, error) {
	if m.ListForAdminFn == nil {
		panic("mock.SlotRepo.ListForAdminFn not set")
	}
	return m.ListForAdminFn(ctx, date)
}

func (m *SlotRepo) Create(ctx context.Context, s *domain.DeliverySlot) error {
	if m.CreateFn == nil {
		panic("mock.SlotRepo.CreateFn not set")
	}
	return m.CreateFn(ctx, s)
}

func (m *SlotRepo) Update(ctx context.Context, s *domain.DeliverySlot) error {
	if m.UpdateFn == nil {
		panic("mock.SlotRepo.UpdateFn not set")
	}
	return m.UpdateFn(ctx, s)
}
