package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo/iface"
)

type ReviewService struct {
	reviewRepo iface.ReviewRepo
	menuRepo   iface.MenuRepo
	orderRepo  iface.OrderRepo
}

func NewReviewService(rr iface.ReviewRepo, mr iface.MenuRepo, or iface.OrderRepo) *ReviewService {
	return &ReviewService{reviewRepo: rr, menuRepo: mr, orderRepo: or}
}

type CreateReviewInput struct {
	OrderID  uuid.UUID
	Rating   int16
	Comment  string
}

func (s *ReviewService) Create(ctx context.Context, callerID uuid.UUID, in CreateReviewInput) error {
	// Verify order ownership and status
	o, err := s.orderRepo.GetByIDWithDetails(ctx, in.OrderID)
	if err != nil || o.UserID != callerID {
		return domain.ErrReviewNotAllowed
	}
	if !o.CanReview() {
		return domain.ErrReviewNotAllowed
	}
	// Check duplicate
	_, err = s.reviewRepo.GetByOrderID(ctx, in.OrderID)
	if err == nil {
		return domain.ErrReviewExists
	}

	// Collect item IDs from the order
	itemIDs := make([]uuid.UUID, 0, len(o.Items))
	for _, it := range o.Items {
		itemIDs = append(itemIDs, it.MenuItemID)
	}

	rev := &repo.ReviewRow{
		ID:            uuid.New(),
		OrderID:       in.OrderID,
		UserID:        callerID,
		Rating:        in.Rating,
		Comment:       in.Comment,
		EditableUntil: time.Now().Add(24 * time.Hour),
	}
	if err := s.reviewRepo.Create(ctx, rev, itemIDs); err != nil {
		return err
	}
	// Update cached rating on each item
	for _, id := range itemIDs {
		_ = s.menuRepo.UpdateRating(ctx, id)
	}
	return nil
}

func (s *ReviewService) Update(ctx context.Context, reviewID, callerID uuid.UUID, rating int16, comment string) error {
	rev, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil || rev.UserID != callerID {
		return domain.ErrNotFound
	}
	if time.Now().After(rev.EditableUntil) {
		return domain.ErrReviewLocked
	}
	if err := s.reviewRepo.Update(ctx, reviewID, rating, comment); err != nil {
		return err
	}
	// Refresh item ratings
	o, _ := s.orderRepo.GetByIDWithDetails(ctx, rev.OrderID)
	if o != nil {
		for _, it := range o.Items {
			_ = s.menuRepo.UpdateRating(ctx, it.MenuItemID)
		}
	}
	return nil
}

func (s *ReviewService) List(ctx context.Context, menuItemID *uuid.UUID, page, pageSize int) ([]repo.PublicReview, int, error) {
	return s.reviewRepo.List(ctx, menuItemID, page, pageSize)
}

func (s *ReviewService) Summary(ctx context.Context) (float64, int, map[string]int, error) {
	return s.reviewRepo.Summary(ctx)
}

func (s *ReviewService) AdminHide(ctx context.Context, reviewID uuid.UUID, hidden bool) error {
	rev, err := s.reviewRepo.GetByID(ctx, reviewID)
	if err != nil {
		return domain.ErrNotFound
	}
	_ = rev
	return s.reviewRepo.SetHidden(ctx, reviewID, hidden)
}
