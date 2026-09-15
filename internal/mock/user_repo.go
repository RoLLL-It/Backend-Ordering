package mock

import (
	"context"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type UserRepo struct {
	CreateFn             func(ctx context.Context, u *domain.User) error
	GetByIDFn            func(ctx context.Context, id uuid.UUID) (*domain.User, error)
	GetByEmailFn         func(ctx context.Context, email string) (*domain.User, error)
	GetByPhoneFn         func(ctx context.Context, phone string) (*domain.User, error)
	ExistsByEmailFn      func(ctx context.Context, email string) (bool, error)
	ExistsByPhoneFn      func(ctx context.Context, phone string) (bool, error)
	UpdateFn             func(ctx context.Context, u *domain.User) error
	ListFn               func(ctx context.Context, search string, role *domain.Role, page, pageSize int) ([]*domain.User, int, error)
	InsertRefreshTokenFn func(ctx context.Context, userID uuid.UUID, hash string, familyID uuid.UUID, expiresAt interface{}) error
	GetRefreshTokenFn    func(ctx context.Context, hash string) (uuid.UUID, uuid.UUID, uuid.UUID, *interface{}, error)
	RevokeRefreshTokenFn func(ctx context.Context, id uuid.UUID) error
	RevokeFamilyTokensFn func(ctx context.Context, familyID uuid.UUID) error
}

func (m *UserRepo) Create(ctx context.Context, u *domain.User) error {
	if m.CreateFn == nil {
		panic("mock.UserRepo.CreateFn not set")
	}
	return m.CreateFn(ctx, u)
}

func (m *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	if m.GetByIDFn == nil {
		panic("mock.UserRepo.GetByIDFn not set")
	}
	return m.GetByIDFn(ctx, id)
}

func (m *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	if m.GetByEmailFn == nil {
		panic("mock.UserRepo.GetByEmailFn not set")
	}
	return m.GetByEmailFn(ctx, email)
}

func (m *UserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	if m.GetByPhoneFn == nil {
		panic("mock.UserRepo.GetByPhoneFn not set")
	}
	return m.GetByPhoneFn(ctx, phone)
}

func (m *UserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	if m.ExistsByEmailFn == nil {
		panic("mock.UserRepo.ExistsByEmailFn not set")
	}
	return m.ExistsByEmailFn(ctx, email)
}

func (m *UserRepo) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	if m.ExistsByPhoneFn == nil {
		panic("mock.UserRepo.ExistsByPhoneFn not set")
	}
	return m.ExistsByPhoneFn(ctx, phone)
}

func (m *UserRepo) Update(ctx context.Context, u *domain.User) error {
	if m.UpdateFn == nil {
		panic("mock.UserRepo.UpdateFn not set")
	}
	return m.UpdateFn(ctx, u)
}

func (m *UserRepo) List(ctx context.Context, search string, role *domain.Role, page, pageSize int) ([]*domain.User, int, error) {
	if m.ListFn == nil {
		panic("mock.UserRepo.ListFn not set")
	}
	return m.ListFn(ctx, search, role, page, pageSize)
}

func (m *UserRepo) InsertRefreshToken(ctx context.Context, userID uuid.UUID, hash string, familyID uuid.UUID, expiresAt interface{}) error {
	if m.InsertRefreshTokenFn == nil {
		panic("mock.UserRepo.InsertRefreshTokenFn not set")
	}
	return m.InsertRefreshTokenFn(ctx, userID, hash, familyID, expiresAt)
}

func (m *UserRepo) GetRefreshToken(ctx context.Context, hash string) (uuid.UUID, uuid.UUID, uuid.UUID, *interface{}, error) {
	if m.GetRefreshTokenFn == nil {
		panic("mock.UserRepo.GetRefreshTokenFn not set")
	}
	return m.GetRefreshTokenFn(ctx, hash)
}

func (m *UserRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	if m.RevokeRefreshTokenFn == nil {
		panic("mock.UserRepo.RevokeRefreshTokenFn not set")
	}
	return m.RevokeRefreshTokenFn(ctx, id)
}

func (m *UserRepo) RevokeFamilyTokens(ctx context.Context, familyID uuid.UUID) error {
	if m.RevokeFamilyTokensFn == nil {
		panic("mock.UserRepo.RevokeFamilyTokensFn not set")
	}
	return m.RevokeFamilyTokensFn(ctx, familyID)
}
