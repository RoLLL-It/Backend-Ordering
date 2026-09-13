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

type ReviewRow struct {
	ID           uuid.UUID
	OrderID      uuid.UUID
	UserID       uuid.UUID
	Rating       int16
	Comment      string
	IsHidden     bool
	EditableUntil time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ReviewRepo struct {
	db *pgxpool.Pool
}

func NewReviewRepo(db *pgxpool.Pool) *ReviewRepo { return &ReviewRepo{db: db} }

func (r *ReviewRepo) Create(ctx context.Context, rev *ReviewRow, itemIDs []uuid.UUID) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx,
		`INSERT INTO reviews(id,order_id,user_id,rating,comment,editable_until)
		 VALUES($1,$2,$3,$4,$5,$6)`,
		rev.ID, rev.OrderID, rev.UserID, rev.Rating, rev.Comment, rev.EditableUntil,
	)
	if err != nil {
		return err
	}
	for _, itemID := range itemIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO review_items(review_id,menu_item_id) VALUES($1,$2)
			 ON CONFLICT DO NOTHING`, rev.ID, itemID)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (r *ReviewRepo) GetByOrderID(ctx context.Context, orderID uuid.UUID) (*ReviewRow, error) {
	rev := &ReviewRow{}
	err := r.db.QueryRow(ctx,
		`SELECT id,order_id,user_id,rating,comment,is_hidden,editable_until,created_at,updated_at
		 FROM reviews WHERE order_id=$1`, orderID).
		Scan(&rev.ID, &rev.OrderID, &rev.UserID, &rev.Rating, &rev.Comment,
			&rev.IsHidden, &rev.EditableUntil, &rev.CreatedAt, &rev.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return rev, err
}

func (r *ReviewRepo) GetByID(ctx context.Context, id uuid.UUID) (*ReviewRow, error) {
	rev := &ReviewRow{}
	err := r.db.QueryRow(ctx,
		`SELECT id,order_id,user_id,rating,comment,is_hidden,editable_until,created_at,updated_at
		 FROM reviews WHERE id=$1`, id).
		Scan(&rev.ID, &rev.OrderID, &rev.UserID, &rev.Rating, &rev.Comment,
			&rev.IsHidden, &rev.EditableUntil, &rev.CreatedAt, &rev.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return rev, err
}

func (r *ReviewRepo) Update(ctx context.Context, id uuid.UUID, rating int16, comment string) error {
	_, err := r.db.Exec(ctx, `UPDATE reviews SET rating=$1,comment=$2 WHERE id=$3`, rating, comment, id)
	return err
}

func (r *ReviewRepo) SetHidden(ctx context.Context, id uuid.UUID, hidden bool) error {
	_, err := r.db.Exec(ctx, `UPDATE reviews SET is_hidden=$1 WHERE id=$2`, hidden, id)
	return err
}

type PublicReview struct {
	ID           uuid.UUID
	Rating       int16
	Comment      string
	ReviewerName string
	Items        []string
	CreatedAt    time.Time
}

func (r *ReviewRepo) List(ctx context.Context, menuItemID *uuid.UUID, page, pageSize int) ([]PublicReview, int, error) {
	filter := ""
	args := []any{pageSize, (page - 1) * pageSize}
	if menuItemID != nil {
		filter = ` AND EXISTS(SELECT 1 FROM review_items ri WHERE ri.review_id=rv.id AND ri.menu_item_id=$3)`
		args = append(args, *menuItemID)
	}
	rows, err := r.db.Query(ctx,
		`SELECT rv.id,rv.rating,rv.comment,u.name,rv.created_at
		 FROM reviews rv JOIN users u ON u.id=rv.user_id
		 WHERE NOT rv.is_hidden`+filter+` ORDER BY rv.created_at DESC LIMIT $1 OFFSET $2`, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var reviews []PublicReview
	for rows.Next() {
		pr := PublicReview{}
		var fullName string
		if err := rows.Scan(&pr.ID, &pr.Rating, &pr.Comment, &fullName, &pr.CreatedAt); err != nil {
			return nil, 0, err
		}
		pr.ReviewerName = formatReviewerName(fullName)
		reviews = append(reviews, pr)
	}
	// Load item names for each review
	for i, rv := range reviews {
		irows, err := r.db.Query(ctx,
			`SELECT mi.name FROM review_items ri JOIN menu_items mi ON mi.id=ri.menu_item_id WHERE ri.review_id=$1`, rv.ID)
		if err == nil {
			var names []string
			for irows.Next() {
				var n string
				_ = irows.Scan(&n)
				names = append(names, n)
			}
			irows.Close()
			reviews[i].Items = names
		}
	}
	var total int
	countArgs := args[2:]
	filterCount := ""
	if menuItemID != nil {
		filterCount = ` AND EXISTS(SELECT 1 FROM review_items ri WHERE ri.review_id=rv.id AND ri.menu_item_id=$1)`
		countArgs = []any{*menuItemID}
	}
	_ = r.db.QueryRow(ctx, `SELECT COUNT(*) FROM reviews rv WHERE NOT rv.is_hidden`+filterCount, countArgs...).Scan(&total)
	return reviews, total, nil
}

func (r *ReviewRepo) Summary(ctx context.Context) (avg float64, total int, dist map[string]int, err error) {
	err = r.db.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating),0),COUNT(*) FROM reviews WHERE NOT is_hidden`).Scan(&avg, &total)
	if err != nil {
		return
	}
	dist = map[string]int{"1": 0, "2": 0, "3": 0, "4": 0, "5": 0}
	rows, err := r.db.Query(ctx,
		`SELECT rating,COUNT(*) FROM reviews WHERE NOT is_hidden GROUP BY rating`)
	if err != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var rating, cnt int
		_ = rows.Scan(&rating, &cnt)
		dist[fmt.Sprint(rating)] = cnt
	}
	return
}

func formatReviewerName(name string) string {
	// Same logic as domain.User.DisplayName()
	parts := []string{}
	w := ""
	for _, c := range name {
		if c == ' ' {
			if w != "" {
				parts = append(parts, w)
				w = ""
			}
		} else {
			w += string(c)
		}
	}
	if w != "" {
		parts = append(parts, w)
	}
	if len(parts) == 0 {
		return name
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return parts[0] + " " + string(parts[len(parts)-1][0]) + "."
}

