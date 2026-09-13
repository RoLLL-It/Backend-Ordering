package service

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/token"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
)

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
var phoneRe = regexp.MustCompile(`^[6-9][0-9]{9}$`)

type AuthService struct {
	userRepo   *repo.UserRepo
	tokenMgr   *token.Manager
	bcryptCost int
	refreshTTL time.Duration
}

func NewAuthService(ur *repo.UserRepo, tm *token.Manager, bcryptCost int, refreshTTL time.Duration) *AuthService {
	return &AuthService{userRepo: ur, tokenMgr: tm, bcryptCost: bcryptCost, refreshTTL: refreshTTL}
}

type RegisterInput struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

type AuthResult struct {
	User        *domain.User
	AccessToken string
	ExpiresIn   int
	RefreshRaw  string
	FamilyID    uuid.UUID
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	// Normalize
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Phone = normalizePhone(in.Phone)

	// Validate
	errs := map[string]string{}
	if len(strings.TrimSpace(in.Name)) < 2 || len(in.Name) > 60 {
		errs["name"] = "Must be 2–60 characters."
	}
	if !emailRe.MatchString(in.Email) {
		errs["email"] = "Must be a valid email address."
	}
	if !phoneRe.MatchString(in.Phone) {
		errs["phone"] = "Must be a 10-digit Indian mobile number."
	}
	if err := validatePassword(in.Password); err != nil {
		errs["password"] = err.Error()
	}
	if len(errs) > 0 {
		return nil, &validationError{fields: errs}
	}

	// Uniqueness
	if ok, _ := s.userRepo.ExistsByEmail(ctx, in.Email); ok {
		return nil, domain.ErrEmailTaken
	}
	if ok, _ := s.userRepo.ExistsByPhone(ctx, in.Phone); ok {
		return nil, domain.ErrPhoneTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		ID:           uuid.New(),
		Name:         strings.TrimSpace(in.Name),
		Email:        in.Email,
		Phone:        in.Phone,
		PasswordHash: string(hash),
		Role:         domain.RoleCustomer,
		IsActive:     true,
	}
	if err := s.userRepo.Create(ctx, u); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, u)
}

type LoginInput struct {
	Identifier string
	Password   string
}

func (s *AuthService) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	in.Identifier = strings.TrimSpace(in.Identifier)

	var u *domain.User
	var err error
	if strings.Contains(in.Identifier, "@") {
		u, err = s.userRepo.GetByEmail(ctx, strings.ToLower(in.Identifier))
	} else {
		phone := normalizePhone(in.Identifier)
		u, err = s.userRepo.GetByPhone(ctx, phone)
	}

	// Constant-time compare — even if user not found, hash a dummy password
	// so the response time doesn't leak whether the identifier exists.
	if errors.Is(err, domain.ErrNotFound) || !u.IsActive {
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$12$dummyhashfortimingattackmitigation"), []byte(in.Password))
		return nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}
	return s.issueTokens(ctx, u)
}

func (s *AuthService) Refresh(ctx context.Context, rawToken string) (*AuthResult, error) {
	hash := token.HashRefreshToken(rawToken)
	id, userID, familyID, revoked, err := s.userRepo.GetRefreshToken(ctx, hash)
	if errors.Is(err, domain.ErrNotFound) {
		return nil, domain.ErrUnauthorized
	}
	if err != nil {
		return nil, err
	}
	if revoked != nil {
		// Stolen token — revoke the whole family
		_ = s.userRepo.RevokeFamilyTokens(ctx, familyID)
		return nil, domain.ErrUnauthorized
	}
	// Revoke old token
	_ = s.userRepo.RevokeRefreshToken(ctx, id)

	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueTokensWithFamily(ctx, u, familyID)
}

func (s *AuthService) Logout(ctx context.Context, rawToken string) error {
	hash := token.HashRefreshToken(rawToken)
	id, _, _, _, err := s.userRepo.GetRefreshToken(ctx, hash)
	if err != nil {
		return nil // already logged out
	}
	return s.userRepo.RevokeRefreshToken(ctx, id)
}

func (s *AuthService) GetMe(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

func (s *AuthService) issueTokens(ctx context.Context, u *domain.User) (*AuthResult, error) {
	return s.issueTokensWithFamily(ctx, u, uuid.New())
}

func (s *AuthService) issueTokensWithFamily(ctx context.Context, u *domain.User, familyID uuid.UUID) (*AuthResult, error) {
	accessToken, err := s.tokenMgr.IssueAccess(u.ID, u.Role)
	if err != nil {
		return nil, err
	}
	raw, hash, err := token.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().Add(s.refreshTTL)
	if err := s.userRepo.InsertRefreshToken(ctx, u.ID, hash, familyID, expiresAt); err != nil {
		return nil, err
	}
	return &AuthResult{
		User:        u,
		AccessToken: accessToken,
		ExpiresIn:   900, // 15 min in seconds
		RefreshRaw:  raw,
		FamilyID:    familyID,
	}, nil
}

// --- helpers ---

func normalizePhone(p string) string {
	// Strip +91, spaces, dashes, then keep 10 digits
	p = strings.TrimPrefix(p, "+91")
	p = strings.TrimPrefix(p, "91")
	var out []rune
	for _, c := range p {
		if unicode.IsDigit(c) {
			out = append(out, c)
		}
	}
	s := string(out)
	if len(s) > 10 {
		s = s[len(s)-10:]
	}
	return s
}

func validatePassword(p string) error {
	if len(p) < 8 {
		return errors.New("must be at least 8 characters")
	}
	hasLetter, hasDigit := false, false
	for _, c := range p {
		if unicode.IsLetter(c) {
			hasLetter = true
		}
		if unicode.IsDigit(c) {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("must contain at least one letter and one digit")
	}
	return nil
}

type validationError struct {
	fields map[string]string
}

func (e *validationError) Error() string { return "validation failed" }
func (e *validationError) Fields() map[string]string { return e.fields }

// IsValidationError checks if err is a validation error and returns its fields.
func IsValidationError(err error) (map[string]string, bool) {
	var ve *validationError
	if errors.As(err, &ve) {
		return ve.fields, true
	}
	return nil, false
}

// Cookie helper — used by auth handler
func RefreshCookieName() string { return "refresh_token" }
func NewRefreshCookie(raw string, ttl time.Duration, secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     RefreshCookieName(),
		Value:    raw,
		Path:     "/api/v1/auth",
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}
func ClearRefreshCookie(secure bool) *http.Cookie {
	return &http.Cookie{
		Name:     RefreshCookieName(),
		Value:    "",
		Path:     "/api/v1/auth",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}
