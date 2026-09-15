package mock

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type MenuRepo struct {
	GetFullMenuFn      func(ctx context.Context) ([]*domain.Category, error)
	GetItemByIDFn      func(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error)
	GetManyByIDsFn     func(ctx context.Context, ids []uuid.UUID) ([]*domain.MenuItem, error)
	GetManyByIDsTxFn   func(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]*domain.MenuItem, error)
	CreateItemFn       func(ctx context.Context, mi *domain.MenuItem) error
	UpdateItemFn       func(ctx context.Context, mi *domain.MenuItem) error
	SetAvailabilityFn  func(ctx context.Context, id uuid.UUID, available bool) error
	SoftDeleteItemFn   func(ctx context.Context, id uuid.UUID) error
	UpdateRatingFn     func(ctx context.Context, itemID uuid.UUID) error
	GetCategoriesFn    func(ctx context.Context) ([]*domain.Category, error)
	CreateCategoryFn   func(ctx context.Context, c *domain.Category) error
	UpdateCategoryFn   func(ctx context.Context, c *domain.Category) error
}

func (m *MenuRepo) GetFullMenu(ctx context.Context) ([]*domain.Category, error) {
	if m.GetFullMenuFn == nil {
		panic("mock.MenuRepo.GetFullMenuFn not set")
	}
	return m.GetFullMenuFn(ctx)
}

func (m *MenuRepo) GetItemByID(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error) {
	if m.GetItemByIDFn == nil {
		panic("mock.MenuRepo.GetItemByIDFn not set")
	}
	return m.GetItemByIDFn(ctx, id)
}

func (m *MenuRepo) GetManyByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.MenuItem, error) {
	if m.GetManyByIDsFn == nil {
		panic("mock.MenuRepo.GetManyByIDsFn not set")
	}
	return m.GetManyByIDsFn(ctx, ids)
}

func (m *MenuRepo) GetManyByIDsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]*domain.MenuItem, error) {
	if m.GetManyByIDsTxFn == nil {
		panic("mock.MenuRepo.GetManyByIDsTxFn not set")
	}
	return m.GetManyByIDsTxFn(ctx, tx, ids)
}

func (m *MenuRepo) CreateItem(ctx context.Context, mi *domain.MenuItem) error {
	if m.CreateItemFn == nil {
		panic("mock.MenuRepo.CreateItemFn not set")
	}
	return m.CreateItemFn(ctx, mi)
}

func (m *MenuRepo) UpdateItem(ctx context.Context, mi *domain.MenuItem) error {
	if m.UpdateItemFn == nil {
		panic("mock.MenuRepo.UpdateItemFn not set")
	}
	return m.UpdateItemFn(ctx, mi)
}

func (m *MenuRepo) SetAvailability(ctx context.Context, id uuid.UUID, available bool) error {
	if m.SetAvailabilityFn == nil {
		panic("mock.MenuRepo.SetAvailabilityFn not set")
	}
	return m.SetAvailabilityFn(ctx, id, available)
}

func (m *MenuRepo) SoftDeleteItem(ctx context.Context, id uuid.UUID) error {
	if m.SoftDeleteItemFn == nil {
		panic("mock.MenuRepo.SoftDeleteItemFn not set")
	}
	return m.SoftDeleteItemFn(ctx, id)
}

func (m *MenuRepo) UpdateRating(ctx context.Context, itemID uuid.UUID) error {
	if m.UpdateRatingFn == nil {
		return nil // rating update is best-effort in tests
	}
	return m.UpdateRatingFn(ctx, itemID)
}

func (m *MenuRepo) GetCategories(ctx context.Context) ([]*domain.Category, error) {
	if m.GetCategoriesFn == nil {
		panic("mock.MenuRepo.GetCategoriesFn not set")
	}
	return m.GetCategoriesFn(ctx)
}

func (m *MenuRepo) CreateCategory(ctx context.Context, c *domain.Category) error {
	if m.CreateCategoryFn == nil {
		panic("mock.MenuRepo.CreateCategoryFn not set")
	}
	return m.CreateCategoryFn(ctx, c)
}

func (m *MenuRepo) UpdateCategory(ctx context.Context, c *domain.Category) error {
	if m.UpdateCategoryFn == nil {
		panic("mock.MenuRepo.UpdateCategoryFn not set")
	}
	return m.UpdateCategoryFn(ctx, c)
}
