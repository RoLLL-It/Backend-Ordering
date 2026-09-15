package iface

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
)

type TxBeginner interface {
	Begin(ctx context.Context) (pgx.Tx, error)
}

type UserRepo interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	GetByPhone(ctx context.Context, phone string) (*domain.User, error)
	ExistsByEmail(ctx context.Context, email string) (bool, error)
	ExistsByPhone(ctx context.Context, phone string) (bool, error)
	Update(ctx context.Context, u *domain.User) error
	List(ctx context.Context, search string, role *domain.Role, page, pageSize int) ([]*domain.User, int, error)
	InsertRefreshToken(ctx context.Context, userID uuid.UUID, hash string, familyID uuid.UUID, expiresAt interface{}) error
	GetRefreshToken(ctx context.Context, hash string) (id uuid.UUID, userID uuid.UUID, familyID uuid.UUID, revokedAt *interface{}, err error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeFamilyTokens(ctx context.Context, familyID uuid.UUID) error
}

type OrderRepo interface {
	InsertTx(ctx context.Context, tx pgx.Tx, o *domain.Order, items []domain.OrderItem) error
	InsertEventTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID, from *domain.OrderStatus, to domain.OrderStatus, actorID *uuid.UUID, note string) error
	GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	ListByUser(ctx context.Context, userID uuid.UUID, activeOnly bool, page, pageSize int) ([]*domain.Order, int, error)
	UpdateStatusTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus) error
	SetDeliveredTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	SetCancelledTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus, reason string) error
	AdminList(ctx context.Context, status *domain.OrderStatus, date *time.Time, locationID *uuid.UUID, page, pageSize int) ([]*domain.Order, int, error)
	NextShortCodeTx(ctx context.Context, tx pgx.Tx) (string, error)
}

type MenuRepo interface {
	GetFullMenu(ctx context.Context) ([]*domain.Category, error)
	GetItemByID(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error)
	GetManyByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.MenuItem, error)
	GetManyByIDsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]*domain.MenuItem, error)
	CreateItem(ctx context.Context, mi *domain.MenuItem) error
	UpdateItem(ctx context.Context, mi *domain.MenuItem) error
	SetAvailability(ctx context.Context, id uuid.UUID, available bool) error
	SoftDeleteItem(ctx context.Context, id uuid.UUID) error
	UpdateRating(ctx context.Context, itemID uuid.UUID) error
	GetCategories(ctx context.Context) ([]*domain.Category, error)
	CreateCategory(ctx context.Context, c *domain.Category) error
	UpdateCategory(ctx context.Context, c *domain.Category) error
}

type SlotRepo interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.DeliverySlot, error)
	GetForUpdateTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.DeliverySlot, error)
	IncrementBookedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	DecrementBookedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error
	ListByLocationAndDate(ctx context.Context, locationID uuid.UUID, date time.Time) ([]*domain.DeliverySlot, error)
	ListForAdmin(ctx context.Context, date time.Time) ([]*domain.DeliverySlot, error)
	Create(ctx context.Context, s *domain.DeliverySlot) error
	Update(ctx context.Context, s *domain.DeliverySlot) error
}

type ReviewRepo interface {
	Create(ctx context.Context, rev *repo.ReviewRow, itemIDs []uuid.UUID) error
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*repo.ReviewRow, error)
	GetByID(ctx context.Context, id uuid.UUID) (*repo.ReviewRow, error)
	Update(ctx context.Context, id uuid.UUID, rating int16, comment string) error
	SetHidden(ctx context.Context, id uuid.UUID, hidden bool) error
	List(ctx context.Context, menuItemID *uuid.UUID, page, pageSize int) ([]repo.PublicReview, int, error)
	Summary(ctx context.Context) (avg float64, total int, dist map[string]int, err error)
}
