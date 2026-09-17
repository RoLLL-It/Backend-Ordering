package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/response"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

type MenuHandler struct {
	svc      *service.MenuService
	settings *service.SettingsService
	locSvc   *service.LocationService
}

func NewMenuHandler(svc *service.MenuService, ss *service.SettingsService, ls *service.LocationService) *MenuHandler {
	return &MenuHandler{svc: svc, settings: ss, locSvc: ls}
}

func (h *MenuHandler) GetMenu(w http.ResponseWriter, r *http.Request) {
	cats, err := h.svc.GetMenu(r.Context())
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load menu"))
		return
	}
	settings, _ := h.settings.GetAll(r.Context())
	response.JSON(w, http.StatusOK, map[string]any{
		"categories":       cats,
		"delivery_enabled": settings != nil && settings.DeliveryEnabled,
		"kitchen_open":     settings != nil && settings.KitchenOpen,
		"announcement":     func() string { if settings != nil { return settings.Announcement }; return "" }(),
	})
}

func (h *MenuHandler) GetItem(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("item"))
		return
	}
	item, err := h.svc.GetItem(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		errs.WriteError(w, r, errs.NotFound("item"))
		return
	}
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load item"))
		return
	}
	response.JSON(w, http.StatusOK, item)
}

// Admin handlers

func (h *MenuHandler) AdminCreateItem(w http.ResponseWriter, r *http.Request) {
	var body struct {
		CategoryID  string  `json:"category_id"`
		Name        string  `json:"name"`
		Description string  `json:"description"`
		PricePaise  int64   `json:"price_paise"`
		ImageURL    *string `json:"image_url"`
		IsVeg       bool    `json:"is_veg"`
		SortOrder   int     `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	catID, err := uuid.Parse(body.CategoryID)
	if err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"category_id": "invalid UUID"}))
		return
	}
	mi := &domain.MenuItem{
		CategoryID: catID, Name: body.Name, Description: body.Description,
		PricePaise: body.PricePaise, ImageURL: body.ImageURL, IsVeg: body.IsVeg, IsAvailable: true, IsActive: true,
		SortOrder: body.SortOrder,
	}
	if err := h.svc.CreateItem(r.Context(), mi); err != nil {
		errs.WriteError(w, r, errs.Internal("could not create item"))
		return
	}
	response.JSON(w, http.StatusCreated, mi)
}

func (h *MenuHandler) AdminUpdateItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("item"))
		return
	}
	var body struct {
		CategoryID  *string `json:"category_id"`
		Name        *string `json:"name"`
		Description *string `json:"description"`
		PricePaise  *int64  `json:"price_paise"`
		ImageURL    *string `json:"image_url"`
		IsVeg       *bool   `json:"is_veg"`
		SortOrder   *int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}

	// Fetch the existing item first — a partial update must not clobber fields
	// (category_id, image_url) the client didn't send.
	mi, err := h.svc.GetItem(r.Context(), id)
	if errors.Is(err, domain.ErrNotFound) {
		errs.WriteError(w, r, errs.NotFound("item"))
		return
	}
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load item"))
		return
	}

	if body.CategoryID != nil {
		catID, err := uuid.Parse(*body.CategoryID)
		if err != nil {
			errs.WriteError(w, r, errs.Validation(map[string]string{"category_id": "invalid UUID"}))
			return
		}
		mi.CategoryID = catID
	}
	if body.Name != nil {
		mi.Name = *body.Name
	}
	if body.Description != nil {
		mi.Description = *body.Description
	}
	if body.PricePaise != nil {
		mi.PricePaise = *body.PricePaise
	}
	if body.ImageURL != nil {
		mi.ImageURL = body.ImageURL
	}
	if body.IsVeg != nil {
		mi.IsVeg = *body.IsVeg
	}
	if body.SortOrder != nil {
		mi.SortOrder = *body.SortOrder
	}

	if err := h.svc.UpdateItem(r.Context(), mi); err != nil {
		errs.WriteError(w, r, errs.Internal("could not update item"))
		return
	}
	response.JSON(w, http.StatusOK, mi)
}

func (h *MenuHandler) AdminSetAvailability(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("item"))
		return
	}
	var body struct {
		IsAvailable bool `json:"is_available"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	if err := h.svc.SetAvailability(r.Context(), id, body.IsAvailable); err != nil {
		errs.WriteError(w, r, errs.Internal("could not update availability"))
		return
	}
	response.NoContent(w)
}

func (h *MenuHandler) AdminDeleteItem(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("item"))
		return
	}
	if err := h.svc.SoftDelete(r.Context(), id); err != nil {
		errs.WriteError(w, r, errs.Internal("could not delete item"))
		return
	}
	response.NoContent(w)
}

func (h *MenuHandler) AdminCreateCategory(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	c := &domain.Category{Name: body.Name, SortOrder: body.SortOrder, IsActive: true}
	if err := h.svc.CreateCategory(r.Context(), c); err != nil {
		errs.WriteError(w, r, errs.Internal("could not create category"))
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *MenuHandler) AdminUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("category"))
		return
	}
	var body struct {
		Name      string `json:"name"`
		SortOrder int    `json:"sort_order"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	c := &domain.Category{ID: id, Name: body.Name, SortOrder: body.SortOrder}
	if err := h.svc.UpdateCategory(r.Context(), c); err != nil {
		errs.WriteError(w, r, errs.Internal("could not update category"))
		return
	}
	response.JSON(w, http.StatusOK, c)
}
