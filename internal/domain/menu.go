package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	SortOrder int        `json:"sort_order"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	Items     []MenuItem `json:"items"`
}

type MenuItem struct {
	ID          uuid.UUID `json:"id"`
	CategoryID  uuid.UUID `json:"category_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	PricePaise  int64     `json:"price_paise"`
	ImageURL    *string   `json:"image_url"`
	IsVeg       bool      `json:"is_veg"`
	IsAvailable bool      `json:"is_available"` // staff sold-out toggle
	IsActive    bool      `json:"is_active"`     // admin soft-delete
	RatingAvg   float64   `json:"rating_avg"`
	RatingCount int       `json:"rating_count"`
	SortOrder   int       `json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
