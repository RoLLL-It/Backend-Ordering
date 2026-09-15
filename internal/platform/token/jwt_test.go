package token

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

func newTestManager() *Manager {
	return NewManager("test-secret-key-for-unit-tests", 15*time.Minute)
}

func TestIssueAndParseAccess(t *testing.T) {
	mgr := newTestManager()
	userID := uuid.New()
	role := domain.RoleCustomer

	tok, err := mgr.IssueAccess(userID, role)
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}
	if tok == "" {
		t.Fatal("IssueAccess() returned empty token")
	}

	claims, err := mgr.Parse(tok)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if claims.Subject != userID.String() {
		t.Errorf("claims.Subject = %s, want %s", claims.Subject, userID.String())
	}
	if claims.Role != role {
		t.Errorf("claims.Role = %s, want %s", claims.Role, role)
	}
}

func TestParseExpiredToken(t *testing.T) {
	// Manager with negative TTL — token is already expired when issued.
	mgr := NewManager("test-secret-key-for-unit-tests", -1*time.Second)
	userID := uuid.New()

	tok, err := mgr.IssueAccess(userID, domain.RoleCustomer)
	if err != nil {
		t.Fatalf("IssueAccess() error = %v", err)
	}

	_, err = mgr.Parse(tok)
	if err == nil {
		t.Fatal("expected error parsing expired token, got nil")
	}
}

func TestGenerateRefreshTokenUniqueness(t *testing.T) {
	seen := make(map[string]bool, 100)
	for i := 0; i < 100; i++ {
		raw, hash, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken() error = %v", err)
		}
		if raw == "" || hash == "" {
			t.Fatal("empty raw or hash returned")
		}
		if seen[raw] {
			t.Fatalf("duplicate raw token at iteration %d", i)
		}
		seen[raw] = true
	}
}

func TestHashRefreshToken(t *testing.T) {
	raw1 := "token-aaa"
	raw2 := "token-bbb"

	h1a := HashRefreshToken(raw1)
	h1b := HashRefreshToken(raw1)
	h2 := HashRefreshToken(raw2)

	if h1a != h1b {
		t.Error("HashRefreshToken is not deterministic")
	}
	if h1a == h2 {
		t.Error("different inputs produced same hash")
	}
}

func TestGenerateRefreshToken_HashMatchesRaw(t *testing.T) {
	raw, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	if HashRefreshToken(raw) != hash {
		t.Error("HashRefreshToken(raw) does not match hash returned by GenerateRefreshToken")
	}
}
