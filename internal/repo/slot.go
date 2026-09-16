package repo

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type SlotRepo struct {
	db *pgxpool.Pool
}

func NewSlotRepo(db *pgxpool.Pool) *SlotRepo { return &SlotRepo{db: db} }

func scanSlot(row pgx.Row) (*domain.DeliverySlot, error) {
	s := &domain.DeliverySlot{}
	var startStr, endStr string
	err := row.Scan(&s.ID, &s.LocationID, &s.SlotDate, &startStr, &endStr,
		&s.Capacity, &s.BookedCount, &s.CutoffMinutes, &s.IsActive, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	s.StartTime = startStr[:5] // "HH:MM:SS" -> "HH:MM"
	s.EndTime = endStr[:5]
	return s, nil
}

func (r *SlotRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.DeliverySlot, error) {
	return scanSlot(r.db.QueryRow(ctx,
		`SELECT id,location_id,slot_date,start_time,end_time,capacity,booked_count,cutoff_minutes,is_active,created_at
		 FROM delivery_slots WHERE id=$1`, id))
}

// GetForUpdate locks the slot row for the current transaction — prevents concurrent oversell.
func (r *SlotRepo) GetForUpdateTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) (*domain.DeliverySlot, error) {
	return scanSlot(tx.QueryRow(ctx,
		`SELECT id,location_id,slot_date,start_time,end_time,capacity,booked_count,cutoff_minutes,is_active,created_at
		 FROM delivery_slots WHERE id=$1 FOR UPDATE`, id))
}

func (r *SlotRepo) IncrementBookedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	tag, err := tx.Exec(ctx,
		`UPDATE delivery_slots SET booked_count=booked_count+1 WHERE id=$1 AND booked_count<capacity`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrSlotFull
	}
	return nil
}

func (r *SlotRepo) DecrementBookedTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE delivery_slots SET booked_count=GREATEST(0,booked_count-1) WHERE id=$1`, id)
	return err
}

func (r *SlotRepo) ListByLocationAndDate(ctx context.Context, locationID uuid.UUID, date time.Time) ([]*domain.DeliverySlot, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,location_id,slot_date,start_time,end_time,capacity,booked_count,cutoff_minutes,is_active,created_at
		 FROM delivery_slots WHERE location_id=$1 AND slot_date=$2 AND is_active=true
		 ORDER BY start_time`, locationID, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slots := []*domain.DeliverySlot{}
	for rows.Next() {
		var startStr, endStr string
		s := &domain.DeliverySlot{}
		if err := rows.Scan(&s.ID, &s.LocationID, &s.SlotDate, &startStr, &endStr,
			&s.Capacity, &s.BookedCount, &s.CutoffMinutes, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.StartTime = startStr[:5]
		s.EndTime = endStr[:5]
		slots = append(slots, s)
	}
	return slots, nil
}

func (r *SlotRepo) Create(ctx context.Context, s *domain.DeliverySlot) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO delivery_slots(id,location_id,slot_date,start_time,end_time,capacity,cutoff_minutes)
		 VALUES($1,$2,$3,$4,$5,$6,$7)`,
		s.ID, s.LocationID, s.SlotDate.Format("2006-01-02"), s.StartTime, s.EndTime, s.Capacity, s.CutoffMinutes,
	)
	return err
}

func (r *SlotRepo) Update(ctx context.Context, s *domain.DeliverySlot) error {
	_, err := r.db.Exec(ctx,
		`UPDATE delivery_slots SET capacity=$1,cutoff_minutes=$2,is_active=$3 WHERE id=$4`,
		s.Capacity, s.CutoffMinutes, s.IsActive, s.ID,
	)
	return err
}

func (r *SlotRepo) ListForAdmin(ctx context.Context, date time.Time) ([]*domain.DeliverySlot, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,location_id,slot_date,start_time,end_time,capacity,booked_count,cutoff_minutes,is_active,created_at
		 FROM delivery_slots WHERE slot_date=$1 ORDER BY location_id,start_time`, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	slots := []*domain.DeliverySlot{}
	for rows.Next() {
		var startStr, endStr string
		s := &domain.DeliverySlot{}
		if err := rows.Scan(&s.ID, &s.LocationID, &s.SlotDate, &startStr, &endStr,
			&s.Capacity, &s.BookedCount, &s.CutoffMinutes, &s.IsActive, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.StartTime = startStr[:5]
		s.EndTime = endStr[:5]
		slots = append(slots, s)
	}
	return slots, nil
}
