package repo

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type MenuRepo struct {
	db *pgxpool.Pool
}

func NewMenuRepo(db *pgxpool.Pool) *MenuRepo { return &MenuRepo{db: db} }

func (r *MenuRepo) GetFullMenu(ctx context.Context) ([]*domain.Category, error) {
	rows, err := r.db.Query(ctx,
		`SELECT c.id,c.name,c.sort_order,
		        mi.id,mi.name,mi.description,mi.price_paise,mi.image_url,
		        mi.is_veg,mi.is_available,mi.is_active,mi.rating_avg,mi.rating_count,mi.sort_order
		 FROM categories c
		 LEFT JOIN menu_items mi ON mi.category_id=c.id AND mi.is_active=true
		 WHERE c.is_active=true
		 ORDER BY c.sort_order,mi.sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	catMap := make(map[uuid.UUID]*domain.Category)
	var catOrder []uuid.UUID
	for rows.Next() {
		var (
			cid   uuid.UUID
			cName string
			cSort int
			// LEFT JOIN produces NULL for every mi.* column when a category has
			// zero active items — all item columns must be nullable scan targets.
			mID                    *uuid.UUID
			mName, mDesc           *string
			mPrice                 *int64
			mImgURL                *string
			mVeg, mAvail, mActive  *bool
			mRatingAvg             *float64
			mRatingCnt, mSort      *int
		)
		if err := rows.Scan(&cid, &cName, &cSort,
			&mID, &mName, &mDesc, &mPrice, &mImgURL,
			&mVeg, &mAvail, &mActive, &mRatingAvg, &mRatingCnt, &mSort); err != nil {
			return nil, err
		}
		cat, exists := catMap[cid]
		if !exists {
			cat = &domain.Category{ID: cid, Name: cName, SortOrder: cSort, IsActive: true, Items: []domain.MenuItem{}}
			catMap[cid] = cat
			catOrder = append(catOrder, cid)
		}
		// LEFT JOIN produces a null item row if the category has no items
		if mID != nil {
			cat.Items = append(cat.Items, domain.MenuItem{
				ID: *mID, CategoryID: cid, Name: *mName, Description: *mDesc,
				PricePaise: *mPrice, ImageURL: mImgURL, IsVeg: *mVeg,
				IsAvailable: *mAvail, IsActive: *mActive,
				RatingAvg: *mRatingAvg, RatingCount: *mRatingCnt, SortOrder: *mSort,
			})
		}
	}
	cats := make([]*domain.Category, 0, len(catOrder))
	for _, id := range catOrder {
		cats = append(cats, catMap[id])
	}
	return cats, nil
}

func (r *MenuRepo) GetItemByID(ctx context.Context, id uuid.UUID) (*domain.MenuItem, error) {
	mi := &domain.MenuItem{}
	err := r.db.QueryRow(ctx,
		`SELECT id,category_id,name,description,price_paise,image_url,
		        is_veg,is_available,is_active,rating_avg,rating_count,sort_order,created_at,updated_at
		 FROM menu_items WHERE id=$1 AND is_active=true`, id).
		Scan(&mi.ID, &mi.CategoryID, &mi.Name, &mi.Description, &mi.PricePaise, &mi.ImageURL,
			&mi.IsVeg, &mi.IsAvailable, &mi.IsActive, &mi.RatingAvg, &mi.RatingCount, &mi.SortOrder, &mi.CreatedAt, &mi.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return mi, err
}

func (r *MenuRepo) GetManyByIDs(ctx context.Context, ids []uuid.UUID) ([]*domain.MenuItem, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id,category_id,name,description,price_paise,image_url,
		        is_veg,is_available,is_active,rating_avg,rating_count,sort_order,created_at,updated_at
		 FROM menu_items WHERE id=ANY($1) AND is_active=true`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*domain.MenuItem{}
	for rows.Next() {
		mi := &domain.MenuItem{}
		if err := rows.Scan(&mi.ID, &mi.CategoryID, &mi.Name, &mi.Description, &mi.PricePaise, &mi.ImageURL,
			&mi.IsVeg, &mi.IsAvailable, &mi.IsActive, &mi.RatingAvg, &mi.RatingCount, &mi.SortOrder, &mi.CreatedAt, &mi.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, mi)
	}
	return items, nil
}

func (r *MenuRepo) GetManyByIDsTx(ctx context.Context, tx pgx.Tx, ids []uuid.UUID) ([]*domain.MenuItem, error) {
	rows, err := tx.Query(ctx,
		`SELECT id,category_id,name,price_paise,is_available,is_active
		 FROM menu_items WHERE id=ANY($1) AND is_active=true`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []*domain.MenuItem{}
	for rows.Next() {
		mi := &domain.MenuItem{}
		if err := rows.Scan(&mi.ID, &mi.CategoryID, &mi.Name, &mi.PricePaise, &mi.IsAvailable, &mi.IsActive); err != nil {
			return nil, err
		}
		items = append(items, mi)
	}
	return items, nil
}

func (r *MenuRepo) CreateItem(ctx context.Context, mi *domain.MenuItem) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO menu_items(id,category_id,name,description,price_paise,image_url,is_veg,is_available,sort_order)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		mi.ID, mi.CategoryID, mi.Name, mi.Description, mi.PricePaise, mi.ImageURL, mi.IsVeg, mi.IsAvailable, mi.SortOrder,
	)
	return err
}

func (r *MenuRepo) UpdateItem(ctx context.Context, mi *domain.MenuItem) error {
	_, err := r.db.Exec(ctx,
		`UPDATE menu_items SET category_id=$1,name=$2,description=$3,price_paise=$4,
		 image_url=$5,is_veg=$6,sort_order=$7 WHERE id=$8`,
		mi.CategoryID, mi.Name, mi.Description, mi.PricePaise, mi.ImageURL, mi.IsVeg, mi.SortOrder, mi.ID,
	)
	return err
}

func (r *MenuRepo) SetAvailability(ctx context.Context, id uuid.UUID, available bool) error {
	_, err := r.db.Exec(ctx, `UPDATE menu_items SET is_available=$1 WHERE id=$2`, available, id)
	return err
}

func (r *MenuRepo) SoftDeleteItem(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, `UPDATE menu_items SET is_active=false WHERE id=$1`, id)
	return err
}

func (r *MenuRepo) UpdateRating(ctx context.Context, itemID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE menu_items SET
		  rating_avg = (SELECT COALESCE(AVG(r.rating),0) FROM reviews r
		                JOIN review_items ri ON ri.review_id=r.id
		                WHERE ri.menu_item_id=$1 AND NOT r.is_hidden),
		  rating_count = (SELECT COUNT(*) FROM reviews r
		                  JOIN review_items ri ON ri.review_id=r.id
		                  WHERE ri.menu_item_id=$1 AND NOT r.is_hidden)
		 WHERE id=$1`, itemID)
	return err
}

// --- Categories ---

func (r *MenuRepo) GetCategories(ctx context.Context) ([]*domain.Category, error) {
	rows, err := r.db.Query(ctx, `SELECT id,name,sort_order,is_active,created_at FROM categories WHERE is_active=true ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	cats := []*domain.Category{}
	for rows.Next() {
		c := &domain.Category{}
		if err := rows.Scan(&c.ID, &c.Name, &c.SortOrder, &c.IsActive, &c.CreatedAt); err != nil {
			return nil, err
		}
		cats = append(cats, c)
	}
	return cats, nil
}

func (r *MenuRepo) CreateCategory(ctx context.Context, c *domain.Category) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO categories(id,name,sort_order) VALUES($1,$2,$3)`,
		c.ID, c.Name, c.SortOrder,
	)
	return err
}

func (r *MenuRepo) UpdateCategory(ctx context.Context, c *domain.Category) error {
	_, err := r.db.Exec(ctx,
		`UPDATE categories SET name=$1,sort_order=$2 WHERE id=$3`,
		c.Name, c.SortOrder, c.ID,
	)
	return err
}
