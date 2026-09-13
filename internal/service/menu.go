package service

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
)

type MenuService struct {
	menuRepo *repo.MenuRepo
	cache    *menuCache
}

type menuCache struct {
	mu        sync.RWMutex
	data      []*domain.Category
	expiresAt time.Time
}

func NewMenuService(mr *repo.MenuRepo) *MenuService {
	return &MenuService{menuRepo: mr, cache: &menuCache{}}
}

func (s *MenuService) GetMenu(ctx context.Context) ([]*domain.Category, error) {
	s.cache.mu.RLock()
	if time.Now().Before(s.cache.expiresAt) {
		data := s.cache.data
		s.cache.mu.RUnlock()
		return data, nil
	}
	s.cache.mu.RUnlock()

	cats, err := s.menuRepo.GetFullMenu(ctx)
	if err != nil {
		return nil, err
	}
	s.cache.mu.Lock()
	s.cache.data = cats
	s.cache.expiresAt = time.Now().Add(60 * time.Second)
	s.cache.mu.Unlock()
	return cats, nil
}

func (s *MenuService) BustCache() {
	s.cache.mu.Lock()
	s.cache.expiresAt = time.Time{}
	s.cache.mu.Unlock()
}

func (s *MenuService) GetItem(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error) {
	return s.menuRepo.GetItemByID(ctx, id)
}

func (s *MenuService) CreateItem(ctx context.Context, mi *domain.MenuItem) error {
	mi.ID = uuid.New()
	if err := s.menuRepo.CreateItem(ctx, mi); err != nil {
		return err
	}
	s.BustCache()
	return nil
}

func (s *MenuService) UpdateItem(ctx context.Context, mi *domain.MenuItem) error {
	if err := s.menuRepo.UpdateItem(ctx, mi); err != nil {
		return err
	}
	s.BustCache()
	return nil
}

func (s *MenuService) SetAvailability(ctx context.Context, id uuid.UUID, available bool) error {
	if err := s.menuRepo.SetAvailability(ctx, id, available); err != nil {
		return err
	}
	s.BustCache()
	return nil
}

func (s *MenuService) SoftDelete(ctx context.Context, id uuid.UUID) error {
	if err := s.menuRepo.SoftDeleteItem(ctx, id); err != nil {
		return err
	}
	s.BustCache()
	return nil
}

func (s *MenuService) CreateCategory(ctx context.Context, c *domain.Category) error {
	c.ID = uuid.New()
	if err := s.menuRepo.CreateCategory(ctx, c); err != nil {
		return err
	}
	s.BustCache()
	return nil
}

func (s *MenuService) UpdateCategory(ctx context.Context, c *domain.Category) error {
	if err := s.menuRepo.UpdateCategory(ctx, c); err != nil {
		return err
	}
	s.BustCache()
	return nil
}

func (s *MenuService) ValidateCart(ctx context.Context, items []CartItem, locationID uuid.UUID) (*CartValidation, error) {
	ids := make([]uuid.UUID, 0, len(items))
	for _, ci := range items {
		ids = append(ids, ci.MenuItemID)
	}
	menuItems, err := s.menuRepo.GetManyByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	itemMap := make(map[uuid.UUID]*domain.MenuItem, len(menuItems))
	for _, mi := range menuItems {
		itemMap[mi.ID] = mi
	}

	var result CartValidation
	var unavailable []uuid.UUID
	for _, ci := range items {
		mi, ok := itemMap[ci.MenuItemID]
		if !ok {
			continue
		}
		lineTotal := mi.PricePaise * int64(ci.Quantity)
		vi := ValidatedItem{
			MenuItemID:    mi.ID,
			Name:          mi.Name,
			PricePaise:    mi.PricePaise,
			Quantity:      ci.Quantity,
			LineTotalPaise: lineTotal,
			IsAvailable:   mi.IsAvailable && mi.IsActive,
		}
		result.Items = append(result.Items, vi)
		result.SubtotalPaise += lineTotal
		if !vi.IsAvailable {
			unavailable = append(unavailable, mi.ID)
			result.UnavailableItems = append(result.UnavailableItems, mi.ID)
		}
	}
	return &result, nil
}

type CartValidation struct {
	Items            []ValidatedItem
	SubtotalPaise    int64
	DeliveryFeePaise int64
	TotalPaise       int64
	UnavailableItems []uuid.UUID
	DeliveryEnabled  bool
}

type ValidatedItem struct {
	MenuItemID     uuid.UUID
	Name           string
	PricePaise     int64
	Quantity       int
	LineTotalPaise int64
	IsAvailable    bool
}
