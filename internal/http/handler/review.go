package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/middleware"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/response"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

type ReviewHandler struct {
	svc *service.ReviewService
}

func NewReviewHandler(svc *service.ReviewService) *ReviewHandler {
	return &ReviewHandler{svc: svc}
}

func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	var body struct {
		OrderID string `json:"order_id"`
		Rating  int16  `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	orderID, err := uuid.Parse(body.OrderID)
	if err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"order_id": "invalid UUID"}))
		return
	}
	if err := h.svc.Create(r.Context(), uid, service.CreateReviewInput{
		OrderID: orderID, Rating: body.Rating, Comment: body.Comment,
	}); err != nil {
		h.handleReviewError(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, map[string]string{"message": "review created"})
}

func (h *ReviewHandler) Update(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("review"))
		return
	}
	var body struct {
		Rating  int16  `json:"rating"`
		Comment string `json:"comment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	if err := h.svc.Update(r.Context(), id, uid, body.Rating, body.Comment); err != nil {
		h.handleReviewError(w, r, err)
		return
	}
	response.NoContent(w)
}

func (h *ReviewHandler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	pageSize := 20
	var menuItemID *uuid.UUID
	if id := r.URL.Query().Get("menu_item_id"); id != "" {
		uid, err := uuid.Parse(id)
		if err == nil {
			menuItemID = &uid
		}
	}
	reviews, total, err := h.svc.List(r.Context(), menuItemID, page, pageSize)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load reviews"))
		return
	}
	response.JSONWithMeta(w, http.StatusOK, reviews, response.Meta{
		Page: page, PageSize: pageSize, Total: total, TotalPages: response.TotalPages(total, pageSize),
	})
}

func (h *ReviewHandler) Summary(w http.ResponseWriter, r *http.Request) {
	avg, total, dist, err := h.svc.Summary(r.Context())
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load summary"))
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{
		"average": avg, "total": total, "distribution": dist,
	})
}

func (h *ReviewHandler) AdminHide(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("review"))
		return
	}
	var body struct {
		IsHidden bool `json:"is_hidden"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	if err := h.svc.AdminHide(r.Context(), id, body.IsHidden); err != nil {
		errs.WriteError(w, r, errs.NotFound("review"))
		return
	}
	response.NoContent(w)
}

func (h *ReviewHandler) handleReviewError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrReviewExists):
		errs.WriteError(w, r, errs.ReviewExists())
	case errors.Is(err, domain.ErrReviewNotAllowed):
		errs.WriteError(w, r, errs.ReviewNotAllowed())
	case errors.Is(err, domain.ErrReviewLocked):
		errs.WriteError(w, r, errs.ReviewLocked())
	case errors.Is(err, domain.ErrNotFound):
		errs.WriteError(w, r, errs.NotFound("review"))
	default:
		errs.WriteError(w, r, errs.Internal("an error occurred"))
	}
}
