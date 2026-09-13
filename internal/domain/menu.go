package domain

import (
	"time"

	"github.com/google/uuid"
)

type Category struct {
	ID        uuid.UUID
	Name      string
	SortOrder int
	IsActive  bool
	CreatedAt time.Time
	Items     []MenuItem
}

type MenuItem struct {
	ID          uuid.UUID
	CategoryID  uuid.UUID
	Name        string
	Description string
	PricePaise  int64
	ImageURL    *string
	IsVeg       bool
	IsAvailable bool // staff sold-out toggle
	IsActive    bool // admin soft-delete
	RatingAvg   float64
	RatingCount int
	SortOrder   int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
