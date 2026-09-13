package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/response"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
)

type AdminUserHandler struct {
	userRepo *repo.UserRepo
}

func NewAdminUserHandler(ur *repo.UserRepo) *AdminUserHandler {
	return &AdminUserHandler{userRepo: ur}
}

func (h *AdminUserHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 { page = 1 }
	pageSize := 20
	var role *domain.Role
	if rs := r.URL.Query().Get("role"); rs != "" {
		r := domain.Role(rs)
		role = &r
	}
	users, total, err := h.userRepo.List(r.Context(), search, role, page, pageSize)
	if err != nil {
		errs.WriteError(w, r, errs.Internal("could not load users"))
		return
	}
	response.JSONWithMeta(w, http.StatusOK, users, response.Meta{
		Page: page, PageSize: pageSize, Total: total, TotalPages: response.TotalPages(total, pageSize),
	})
}

func (h *AdminUserHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("user"))
		return
	}
	var body struct {
		Role     *string `json:"role"`
		IsActive *bool   `json:"is_active"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("user"))
		return
	}
	if body.Role != nil {
		user.Role = domain.Role(*body.Role)
	}
	if body.IsActive != nil {
		user.IsActive = *body.IsActive
	}
	if err := h.userRepo.Update(r.Context(), user); err != nil {
		errs.WriteError(w, r, errs.Internal("could not update user"))
		return
	}
	response.JSON(w, http.StatusOK, userDTO(user))
}
