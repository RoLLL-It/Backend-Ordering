package service

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type LocationService struct {
	db *pgxpool.Pool
}

func NewLocationService(db *pgxpool.Pool) *LocationService {
	return &LocationService{db: db}
}

func (s *LocationService) List(ctx context.Context) ([]*domain.Location, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id,code,name,delivery_enabled,delivery_fee_paise,sort_order,created_at
		 FROM locations ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var locs []*domain.Location
	for rows.Next() {
		l := &domain.Location{}
		if err := rows.Scan(&l.ID, &l.Code, &l.Name, &l.DeliveryEnabled, &l.DeliveryFeePaise, &l.SortOrder, &l.CreatedAt); err != nil {
			return nil, err
		}
		locs = append(locs, l)
	}
	return locs, nil
}

func (s *LocationService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Location, error) {
	l := &domain.Location{}
	err := s.db.QueryRow(ctx,
		`SELECT id,code,name,delivery_enabled,delivery_fee_paise,sort_order,created_at
		 FROM locations WHERE id=$1`, id).
		Scan(&l.ID, &l.Code, &l.Name, &l.DeliveryEnabled, &l.DeliveryFeePaise, &l.SortOrder, &l.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return l, err
}

func (s *LocationService) SetDelivery(ctx context.Context, id uuid.UUID, enabled bool) error {
	_, err := s.db.Exec(ctx, `UPDATE locations SET delivery_enabled=$1 WHERE id=$2`, enabled, id)
	return err
}
