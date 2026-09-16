package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo { return &UserRepo{db: db} }

func scanUser(row pgx.Row) (*domain.User, error) {
	u := &domain.User{}
	err := row.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash,
		&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return u, nil
}

func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users(id,name,email,phone,password_hash,role,is_active)
		 VALUES($1,$2,$3,$4,$5,$6,$7)`,
		u.ID, u.Name, u.Email, u.Phone, u.PasswordHash, u.Role, u.IsActive,
	)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT id,name,email,phone,password_hash,role,is_active,created_at,updated_at
		 FROM users WHERE id=$1`, id))
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT id,name,email,phone,password_hash,role,is_active,created_at,updated_at
		 FROM users WHERE email=$1`, email))
}

func (r *UserRepo) GetByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return scanUser(r.db.QueryRow(ctx,
		`SELECT id,name,email,phone,password_hash,role,is_active,created_at,updated_at
		 FROM users WHERE phone=$1`, phone))
}

func (r *UserRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE email=$1)`, email).Scan(&exists)
	return exists, err
}

func (r *UserRepo) ExistsByPhone(ctx context.Context, phone string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE phone=$1)`, phone).Scan(&exists)
	return exists, err
}

func (r *UserRepo) Update(ctx context.Context, u *domain.User) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET name=$1,email=$2,phone=$3,role=$4,is_active=$5 WHERE id=$6`,
		u.Name, u.Email, u.Phone, u.Role, u.IsActive, u.ID,
	)
	return err
}

func (r *UserRepo) List(ctx context.Context, search string, role *domain.Role, page, pageSize int) ([]*domain.User, int, error) {
	offset := (page - 1) * pageSize
	args := []any{"%" + search + "%", pageSize, offset}
	roleFilter := ""
	if role != nil {
		roleFilter = " AND role=$4"
		args = append(args, *role)
	}
	rows, err := r.db.Query(ctx,
		`SELECT id,name,email,phone,password_hash,role,is_active,created_at,updated_at
		 FROM users WHERE (name ILIKE $1 OR email ILIKE $1 OR phone ILIKE $1)`+roleFilter+
			` ORDER BY created_at DESC LIMIT $2 OFFSET $3`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	users := []*domain.User{}
	for rows.Next() {
		u := &domain.User{}
		if err := rows.Scan(&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash,
			&u.Role, &u.IsActive, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}
	var total int
	_ = r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE (name ILIKE $1 OR email ILIKE $1 OR phone ILIKE $1)`+roleFilter,
		args[:len(args)-2]...).Scan(&total)
	return users, total, nil
}

// --- Refresh tokens ---

func (r *UserRepo) InsertRefreshToken(ctx context.Context, userID uuid.UUID, hash string, familyID uuid.UUID, expiresAt interface{}) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO refresh_tokens(user_id,token_hash,family_id,expires_at) VALUES($1,$2,$3,$4)`,
		userID, hash, familyID, expiresAt,
	)
	return err
}

func (r *UserRepo) GetRefreshToken(ctx context.Context, hash string) (id uuid.UUID, userID uuid.UUID, familyID uuid.UUID, revokedAt *interface{}, err error) {
	var revoked *interface{}
	err = r.db.QueryRow(ctx,
		`SELECT id,user_id,family_id,revoked_at FROM refresh_tokens WHERE token_hash=$1 AND expires_at>now()`,
		hash).Scan(&id, &userID, &familyID, &revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, uuid.Nil, uuid.Nil, nil, domain.ErrNotFound
	}
	return id, userID, familyID, revoked, err
}

func (r *UserRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now() WHERE id=$1`, id)
	return err
}

func (r *UserRepo) RevokeFamilyTokens(ctx context.Context, familyID uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now() WHERE family_id=$1 AND revoked_at IS NULL`, familyID)
	return err
}
