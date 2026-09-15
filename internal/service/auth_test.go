package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/mock"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/token"
)

func newTestAuthService(ur *mock.UserRepo) *AuthService {
	mgr := token.NewManager("test-secret", 15*time.Minute)
	return NewAuthService(ur, mgr, 4 /* bcryptCost=4 for speed */, 720*time.Hour)
}

// registerHelper calls Register and returns the resulting user+raw token.
// It sets up the mock to store the created user so later tests can reference it.
func registerHelper(t *testing.T, svc *AuthService, ur *mock.UserRepo, email, phone, password string) (*domain.User, string) {
	t.Helper()
	var created *domain.User
	origCreate := ur.CreateFn
	ur.CreateFn = func(ctx context.Context, u *domain.User) error {
		created = u
		if origCreate != nil {
			return origCreate(ctx, u)
		}
		return nil
	}
	result, err := svc.Register(context.Background(), RegisterInput{
		Name: "Test User", Email: email, Phone: phone, Password: password,
	})
	if err != nil {
		t.Fatalf("registerHelper: Register failed: %v", err)
	}
	return created, result.RefreshRaw
}

// --- Register ---

func TestRegister_Success(t *testing.T) {
	ur := &mock.UserRepo{
		ExistsByEmailFn:      func(_ context.Context, _ string) (bool, error) { return false, nil },
		ExistsByPhoneFn:      func(_ context.Context, _ string) (bool, error) { return false, nil },
		CreateFn:             func(_ context.Context, _ *domain.User) error { return nil },
		InsertRefreshTokenFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ interface{}) error { return nil },
	}
	svc := newTestAuthService(ur)
	result, err := svc.Register(context.Background(), RegisterInput{
		Name:     "Heet Test",
		Email:    "heet@example.com",
		Phone:    "9876543210",
		Password: "Secret123",
	})
	if err != nil {
		t.Fatalf("Register() unexpected error: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected non-empty access token")
	}
	if result.User == nil {
		t.Error("expected non-nil user")
	}
}

func TestRegister_NameTooShort(t *testing.T) {
	svc := newTestAuthService(&mock.UserRepo{})
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "A", Email: "a@b.com", Phone: "9876543210", Password: "Secret123",
	})
	if err == nil {
		t.Fatal("expected validation error for short name")
	}
	_, ok := IsValidationError(err)
	if !ok {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestRegister_BadEmail(t *testing.T) {
	svc := newTestAuthService(&mock.UserRepo{})
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "Valid Name", Email: "not-an-email", Phone: "9876543210", Password: "Secret123",
	})
	fields, ok := IsValidationError(err)
	if !ok {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, has := fields["email"]; !has {
		t.Error("expected 'email' field in validation error")
	}
}

func TestRegister_BadPhone(t *testing.T) {
	svc := newTestAuthService(&mock.UserRepo{})
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "Valid Name", Email: "a@b.com", Phone: "12345", Password: "Secret123",
	})
	fields, ok := IsValidationError(err)
	if !ok {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, has := fields["phone"]; !has {
		t.Error("expected 'phone' field in validation error")
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	svc := newTestAuthService(&mock.UserRepo{})
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "Valid Name", Email: "a@b.com", Phone: "9876543210", Password: "abc",
	})
	fields, ok := IsValidationError(err)
	if !ok {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, has := fields["password"]; !has {
		t.Error("expected 'password' field in validation error")
	}
}

func TestRegister_PasswordNoDigit(t *testing.T) {
	svc := newTestAuthService(&mock.UserRepo{})
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "Valid Name", Email: "a@b.com", Phone: "9876543210", Password: "NoDigitPassword",
	})
	fields, ok := IsValidationError(err)
	if !ok {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if _, has := fields["password"]; !has {
		t.Error("expected 'password' field in validation error")
	}
}

func TestRegister_EmailTaken(t *testing.T) {
	ur := &mock.UserRepo{
		ExistsByEmailFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
		ExistsByPhoneFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
	}
	svc := newTestAuthService(ur)
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "Valid Name", Email: "taken@b.com", Phone: "9876543210", Password: "Secret123",
	})
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("expected ErrEmailTaken, got %v", err)
	}
}

func TestRegister_PhoneTaken(t *testing.T) {
	ur := &mock.UserRepo{
		ExistsByEmailFn: func(_ context.Context, _ string) (bool, error) { return false, nil },
		ExistsByPhoneFn: func(_ context.Context, _ string) (bool, error) { return true, nil },
	}
	svc := newTestAuthService(ur)
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "Valid Name", Email: "a@b.com", Phone: "9876543210", Password: "Secret123",
	})
	if !errors.Is(err, domain.ErrPhoneTaken) {
		t.Errorf("expected ErrPhoneTaken, got %v", err)
	}
}

// --- Login ---

func setupRegisteredUser(t *testing.T) (*domain.User, *AuthService, *mock.UserRepo) {
	t.Helper()
	var createdUser *domain.User
	ur := &mock.UserRepo{
		ExistsByEmailFn:      func(_ context.Context, _ string) (bool, error) { return false, nil },
		ExistsByPhoneFn:      func(_ context.Context, _ string) (bool, error) { return false, nil },
		CreateFn:             func(_ context.Context, u *domain.User) error { createdUser = u; return nil },
		InsertRefreshTokenFn: func(_ context.Context, _ uuid.UUID, _ string, _ uuid.UUID, _ interface{}) error { return nil },
	}
	svc := newTestAuthService(ur)
	// Phone must start with 6-9 but NOT start with "91" (normalizePhone strips that prefix)
	_, err := svc.Register(context.Background(), RegisterInput{
		Name: "Heet", Email: "heet@test.com", Phone: "8123456789", Password: "Secret123",
	})
	if err != nil {
		t.Fatalf("setup Register failed: %v", err)
	}
	return createdUser, svc, ur
}

func TestLogin_EmailSuccess(t *testing.T) {
	user, svc, ur := setupRegisteredUser(t)
	ur.GetByEmailFn = func(_ context.Context, _ string) (*domain.User, error) { return user, nil }

	result, err := svc.Login(context.Background(), LoginInput{Identifier: "heet@test.com", Password: "Secret123"})
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestLogin_PhoneSuccess(t *testing.T) {
	user, svc, ur := setupRegisteredUser(t)
	ur.GetByPhoneFn = func(_ context.Context, _ string) (*domain.User, error) { return user, nil }

	// Phone login: identifier has no "@"
	_, err := svc.Login(context.Background(), LoginInput{Identifier: "8123456789", Password: "Secret123"})
	if err != nil {
		t.Fatalf("Login() via phone error: %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	user, svc, ur := setupRegisteredUser(t)
	ur.GetByEmailFn = func(_ context.Context, _ string) (*domain.User, error) { return user, nil }

	_, err := svc.Login(context.Background(), LoginInput{Identifier: "heet@test.com", Password: "WrongPass9"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	ur := &mock.UserRepo{
		GetByEmailFn: func(_ context.Context, _ string) (*domain.User, error) {
			return nil, domain.ErrNotFound
		},
	}
	svc := newTestAuthService(ur)
	_, err := svc.Login(context.Background(), LoginInput{Identifier: "notfound@test.com", Password: "Secret123"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for not found, got %v", err)
	}
}

func TestLogin_InactiveUser(t *testing.T) {
	user, svc, ur := setupRegisteredUser(t)
	user.IsActive = false
	ur.GetByEmailFn = func(_ context.Context, _ string) (*domain.User, error) { return user, nil }

	_, err := svc.Login(context.Background(), LoginInput{Identifier: "heet@test.com", Password: "Secret123"})
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials for inactive user, got %v", err)
	}
}

// --- Refresh ---

func TestRefresh_HappyPath(t *testing.T) {
	userID := uuid.New()
	familyID := uuid.New()
	tokenID := uuid.New()
	var capturedHash string

	ur := &mock.UserRepo{
		ExistsByEmailFn:      func(_ context.Context, _ string) (bool, error) { return false, nil },
		ExistsByPhoneFn:      func(_ context.Context, _ string) (bool, error) { return false, nil },
		CreateFn:             func(_ context.Context, _ *domain.User) error { return nil },
		InsertRefreshTokenFn: func(_ context.Context, _ uuid.UUID, hash string, _ uuid.UUID, _ interface{}) error {
			capturedHash = hash
			return nil
		},
		GetRefreshTokenFn: func(_ context.Context, hash string) (uuid.UUID, uuid.UUID, uuid.UUID, *interface{}, error) {
			if hash == capturedHash {
				return tokenID, userID, familyID, nil, nil
			}
			return uuid.Nil, uuid.Nil, uuid.Nil, nil, domain.ErrNotFound
		},
		RevokeRefreshTokenFn: func(_ context.Context, _ uuid.UUID) error { return nil },
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.User, error) {
			return &domain.User{ID: userID, Role: domain.RoleCustomer, IsActive: true}, nil
		},
	}

	svc := newTestAuthService(ur)
	result, err := svc.Register(context.Background(), RegisterInput{
		Name: "Heet", Email: "heet@refresh.com", Phone: "9000000002", Password: "Secret123",
	})
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	result2, err := svc.Refresh(context.Background(), result.RefreshRaw)
	if err != nil {
		t.Fatalf("Refresh() error: %v", err)
	}
	if result2.AccessToken == "" {
		t.Error("expected access token after refresh")
	}
}

func TestRefresh_RevokedToken(t *testing.T) {
	ur := &mock.UserRepo{
		GetRefreshTokenFn: func(_ context.Context, _ string) (uuid.UUID, uuid.UUID, uuid.UUID, *interface{}, error) {
			return uuid.Nil, uuid.Nil, uuid.Nil, nil, domain.ErrNotFound
		},
	}
	svc := newTestAuthService(ur)
	_, err := svc.Refresh(context.Background(), "bogus-token")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for revoked token, got %v", err)
	}
}

func TestRefresh_StolenToken_RevokesFamily(t *testing.T) {
	familyID := uuid.New()
	familyRevoked := false

	revokedAt := interface{}("2024-01-01T00:00:00Z")
	ur := &mock.UserRepo{
		GetRefreshTokenFn: func(_ context.Context, _ string) (uuid.UUID, uuid.UUID, uuid.UUID, *interface{}, error) {
			return uuid.New(), uuid.New(), familyID, &revokedAt, nil
		},
		RevokeFamilyTokensFn: func(_ context.Context, fid uuid.UUID) error {
			if fid == familyID {
				familyRevoked = true
			}
			return nil
		},
	}
	svc := newTestAuthService(ur)
	_, err := svc.Refresh(context.Background(), "stolen-token")
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized for stolen token, got %v", err)
	}
	if !familyRevoked {
		t.Error("expected family tokens to be revoked on stolen token detection")
	}
}

// --- Logout ---

func TestLogout_Success(t *testing.T) {
	tokenID := uuid.New()
	revoked := false
	ur := &mock.UserRepo{
		GetRefreshTokenFn: func(_ context.Context, _ string) (uuid.UUID, uuid.UUID, uuid.UUID, *interface{}, error) {
			return tokenID, uuid.New(), uuid.New(), nil, nil
		},
		RevokeRefreshTokenFn: func(_ context.Context, _ uuid.UUID) error {
			revoked = true
			return nil
		},
	}
	svc := newTestAuthService(ur)
	if err := svc.Logout(context.Background(), "some-raw-token"); err != nil {
		t.Errorf("Logout() error: %v", err)
	}
	if !revoked {
		t.Error("expected token to be revoked")
	}
}

func TestLogout_TokenNotFound_ReturnsNil(t *testing.T) {
	ur := &mock.UserRepo{
		GetRefreshTokenFn: func(_ context.Context, _ string) (uuid.UUID, uuid.UUID, uuid.UUID, *interface{}, error) {
			return uuid.Nil, uuid.Nil, uuid.Nil, nil, domain.ErrNotFound
		},
	}
	svc := newTestAuthService(ur)
	if err := svc.Logout(context.Background(), "nonexistent-token"); err != nil {
		t.Errorf("Logout() with nonexistent token returned error: %v", err)
	}
}
