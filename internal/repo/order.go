package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type OrderRepo struct {
	db *pgxpool.Pool
}

func NewOrderRepo(db *pgxpool.Pool) *OrderRepo { return &OrderRepo{db: db} }

func (r *OrderRepo) InsertTx(ctx context.Context, tx pgx.Tx, o *domain.Order, items []domain.OrderItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO orders(id,short_code,user_id,location_id,slot_id,status,payment_mode,payment_status,
		 subtotal_paise,delivery_fee_paise,total_paise,notes,cancel_deadline_at)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		o.ID, o.ShortCode, o.UserID, o.LocationID, o.SlotID, o.Status,
		o.PaymentMode, o.PaymentStatus, o.SubtotalPaise, o.DeliveryFeePaise,
		o.TotalPaise, o.Notes, o.CancelDeadlineAt,
	)
	if err != nil {
		return err
	}
	for _, it := range items {
		it.ID = uuid.New()
		it.OrderID = o.ID
		_, err = tx.Exec(ctx,
			`INSERT INTO order_items(id,order_id,menu_item_id,name_snapshot,price_snapshot_paise,quantity,line_total_paise)
			 VALUES($1,$2,$3,$4,$5,$6,$7)`,
			it.ID, it.OrderID, it.MenuItemID, it.NameSnapshot, it.PriceSnapshotPaise, it.Quantity, it.LineTotalPaise,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *OrderRepo) InsertEventTx(ctx context.Context, tx pgx.Tx, orderID uuid.UUID,
	from *domain.OrderStatus, to domain.OrderStatus, actorID *uuid.UUID, note string) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO order_status_events(id,order_id,from_status,to_status,actor_id,note)
		 VALUES($1,$2,$3,$4,$5,$6)`,
		uuid.New(), orderID, from, to, actorID, note,
	)
	return err
}

// GetByIDWithDetails is the proper implementation.
func (r *OrderRepo) GetByIDWithDetails(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	o := &domain.Order{Location: &domain.Location{}, Slot: &domain.DeliverySlot{}}
	var startStr, endStr string
	err := r.db.QueryRow(ctx,
		`SELECT o.id,o.short_code,o.user_id,o.location_id,o.slot_id,o.status,
		        o.payment_mode,o.payment_status,o.subtotal_paise,o.delivery_fee_paise,o.total_paise,
		        o.notes,o.cancel_deadline_at,o.placed_at,o.delivered_at,o.cancelled_at,o.cancel_reason,o.updated_at,
		        l.id,l.code,l.name,l.delivery_enabled,l.delivery_fee_paise,
		        ds.id,ds.slot_date,ds.start_time,ds.end_time
		 FROM orders o
		 JOIN locations l ON l.id=o.location_id
		 JOIN delivery_slots ds ON ds.id=o.slot_id
		 WHERE o.id=$1`, id).
		Scan(&o.ID, &o.ShortCode, &o.UserID, &o.LocationID, &o.SlotID, &o.Status,
			&o.PaymentMode, &o.PaymentStatus, &o.SubtotalPaise, &o.DeliveryFeePaise, &o.TotalPaise,
			&o.Notes, &o.CancelDeadlineAt, &o.PlacedAt, &o.DeliveredAt, &o.CancelledAt, &o.CancelReason, &o.UpdatedAt,
			&o.Location.ID, &o.Location.Code, &o.Location.Name, &o.Location.DeliveryEnabled, &o.Location.DeliveryFeePaise,
			&o.Slot.ID, &o.Slot.SlotDate, &startStr, &endStr,
		)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(startStr) >= 5 {
		o.Slot.StartTime = startStr[:5]
	}
	if len(endStr) >= 5 {
		o.Slot.EndTime = endStr[:5]
	}

	// Load items
	items, err := r.getItems(ctx, id)
	if err != nil {
		return nil, err
	}
	o.Items = items

	// Load events
	events, err := r.getEvents(ctx, id)
	if err != nil {
		return nil, err
	}
	o.Events = events
	return o, nil
}

func (r *OrderRepo) getItems(ctx context.Context, orderID uuid.UUID) ([]domain.OrderItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,order_id,menu_item_id,name_snapshot,price_snapshot_paise,quantity,line_total_paise
		 FROM order_items WHERE order_id=$1`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.OrderItem
	for rows.Next() {
		it := domain.OrderItem{}
		if err := rows.Scan(&it.ID, &it.OrderID, &it.MenuItemID, &it.NameSnapshot,
			&it.PriceSnapshotPaise, &it.Quantity, &it.LineTotalPaise); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *OrderRepo) getEvents(ctx context.Context, orderID uuid.UUID) ([]domain.OrderStatusEvent, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,order_id,from_status,to_status,actor_id,note,created_at
		 FROM order_status_events WHERE order_id=$1 ORDER BY created_at`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []domain.OrderStatusEvent
	for rows.Next() {
		e := domain.OrderStatusEvent{}
		if err := rows.Scan(&e.ID, &e.OrderID, &e.FromStatus, &e.ToStatus, &e.ActorID, &e.Note, &e.CreatedAt); err != nil {
			return nil, err
		}
		events = append(events, e)
	}
	return events, nil
}

func (r *OrderRepo) ListByUser(ctx context.Context, userID uuid.UUID, activeOnly bool, page, pageSize int) ([]*domain.Order, int, error) {
	filter := ""
	args := []any{userID, pageSize, (page - 1) * pageSize}
	if activeOnly {
		filter = " AND status IN ('PLACED','ACCEPTED','PREPARING','READY','OUT_FOR_DELIVERY')"
	}
	rows, err := r.db.Query(ctx,
		`SELECT id,short_code,user_id,location_id,slot_id,status,payment_mode,payment_status,
		        subtotal_paise,delivery_fee_paise,total_paise,notes,cancel_deadline_at,placed_at,
		        delivered_at,cancelled_at,cancel_reason,updated_at
		 FROM orders WHERE user_id=$1`+filter+` ORDER BY placed_at DESC LIMIT $2 OFFSET $3`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(&o.ID, &o.ShortCode, &o.UserID, &o.LocationID, &o.SlotID, &o.Status,
			&o.PaymentMode, &o.PaymentStatus, &o.SubtotalPaise, &o.DeliveryFeePaise, &o.TotalPaise,
			&o.Notes, &o.CancelDeadlineAt, &o.PlacedAt, &o.DeliveredAt, &o.CancelledAt, &o.CancelReason, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	var total int
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders WHERE user_id=$1`+filter, args[:1]...).Scan(&total)
	return orders, total, nil
}

func (r *OrderRepo) UpdateStatusTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus) error {
	tag, err := tx.Exec(ctx, `UPDATE orders SET status=$1 WHERE id=$2`, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *OrderRepo) SetDeliveredTx(ctx context.Context, tx pgx.Tx, id uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE orders SET status='DELIVERED',delivered_at=now(),payment_status='PAID' WHERE id=$1`, id)
	return err
}

func (r *OrderRepo) SetCancelledTx(ctx context.Context, tx pgx.Tx, id uuid.UUID, status domain.OrderStatus, reason string) error {
	_, err := tx.Exec(ctx,
		`UPDATE orders SET status=$1,cancelled_at=now(),cancel_reason=$2 WHERE id=$3`,
		status, reason, id,
	)
	return err
}

func (r *OrderRepo) AdminList(ctx context.Context, status *domain.OrderStatus, date *time.Time, locationID *uuid.UUID, page, pageSize int) ([]*domain.Order, int, error) {
	where := "WHERE 1=1"
	args := []any{}
	i := 1
	if status != nil {
		where += fmt.Sprintf(" AND status=$%d", i)
		args = append(args, *status)
		i++
	}
	if date != nil {
		where += fmt.Sprintf(" AND DATE(placed_at)=$%d", i)
		args = append(args, date.Format("2006-01-02"))
		i++
	}
	if locationID != nil {
		where += fmt.Sprintf(" AND location_id=$%d", i)
		args = append(args, *locationID)
		i++
	}
	args = append(args, pageSize, (page-1)*pageSize)
	rows, err := r.db.Query(ctx,
		`SELECT id,short_code,user_id,location_id,slot_id,status,payment_mode,payment_status,
		        subtotal_paise,delivery_fee_paise,total_paise,notes,cancel_deadline_at,placed_at,
		        delivered_at,cancelled_at,cancel_reason,updated_at
		 FROM orders `+where+fmt.Sprintf(` ORDER BY placed_at DESC LIMIT $%d OFFSET $%d`, i, i+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var orders []*domain.Order
	for rows.Next() {
		o := &domain.Order{}
		if err := rows.Scan(&o.ID, &o.ShortCode, &o.UserID, &o.LocationID, &o.SlotID, &o.Status,
			&o.PaymentMode, &o.PaymentStatus, &o.SubtotalPaise, &o.DeliveryFeePaise, &o.TotalPaise,
			&o.Notes, &o.CancelDeadlineAt, &o.PlacedAt, &o.DeliveredAt, &o.CancelledAt, &o.CancelReason, &o.UpdatedAt); err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	var total int
	countArgs := args[:len(args)-2]
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM orders `+where, countArgs...).Scan(&total)
	return orders, total, nil
}

// NextShortCode generates the next RIT-XXX code inside a transaction.
func (r *OrderRepo) NextShortCodeTx(ctx context.Context, tx pgx.Tx) (string, error) {
	today := time.Now().UTC().Format("2006-01-02")
	var seq int
	err := tx.QueryRow(ctx,
		`INSERT INTO daily_order_counters(counter_date,sequence)
		 VALUES($1,1)
		 ON CONFLICT(counter_date) DO UPDATE SET sequence=daily_order_counters.sequence+1
		 RETURNING sequence`, today).Scan(&seq)
	if err != nil {
		return "", fmt.Errorf("short code: %w", err)
	}
	letters := "ABCDEFGHJKLMNPQRSTUVWXYZ" // no I or O
	letter := letters[(seq-1)/100%len(letters)]
	number := (seq - 1) % 100
	return fmt.Sprintf("RIT-%c%02d", letter, number), nil
}
