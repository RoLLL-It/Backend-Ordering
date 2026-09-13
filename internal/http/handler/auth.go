package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/errs"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/middleware"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/response"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

type AuthHandler struct {
	svc        *service.AuthService
	refreshTTL time.Duration
	isSecure   bool
}

func NewAuthHandler(svc *service.AuthService, refreshTTL time.Duration, isSecure bool) *AuthHandler {
	return &AuthHandler{svc: svc, refreshTTL: refreshTTL, isSecure: isSecure}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	result, err := h.svc.Register(r.Context(), service.RegisterInput{
		Name: body.Name, Email: body.Email, Phone: body.Phone, Password: body.Password,
	})
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}
	h.setRefreshCookie(w, result.RefreshRaw)
	response.JSON(w, http.StatusCreated, map[string]any{
		"user":         userDTO(result.User),
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Identifier string `json:"identifier"`
		Password   string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		errs.WriteError(w, r, errs.Validation(map[string]string{"body": "invalid JSON"}))
		return
	}
	result, err := h.svc.Login(r.Context(), service.LoginInput{Identifier: body.Identifier, Password: body.Password})
	if err != nil {
		h.handleAuthError(w, r, err)
		return
	}
	h.setRefreshCookie(w, result.RefreshRaw)
	response.JSON(w, http.StatusOK, map[string]any{
		"user":         userDTO(result.User),
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(service.RefreshCookieName())
	if err != nil {
		errs.WriteError(w, r, errs.Unauthorized("refresh token cookie missing"))
		return
	}
	result, err := h.svc.Refresh(r.Context(), cookie.Value)
	if err != nil {
		errs.WriteError(w, r, errs.Unauthorized("invalid or expired refresh token"))
		return
	}
	h.setRefreshCookie(w, result.RefreshRaw)
	response.JSON(w, http.StatusOK, map[string]any{
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(service.RefreshCookieName())
	if err == nil {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}
	http.SetCookie(w, service.ClearRefreshCookie(h.isSecure))
	response.NoContent(w)
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	uid := middleware.GetUserID(r.Context())
	u, err := h.svc.GetMe(r.Context(), uid)
	if err != nil {
		errs.WriteError(w, r, errs.NotFound("user"))
		return
	}
	response.JSON(w, http.StatusOK, userDTO(u))
}

func (h *AuthHandler) setRefreshCookie(w http.ResponseWriter, raw string) {
	http.SetCookie(w, service.NewRefreshCookie(raw, h.refreshTTL, h.isSecure))
}

func (h *AuthHandler) handleAuthError(w http.ResponseWriter, r *http.Request, err error) {
	if fields, ok := service.IsValidationError(err); ok {
		errs.WriteError(w, r, errs.Validation(fields))
		return
	}
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		errs.WriteError(w, r, errs.EmailTaken())
	case errors.Is(err, domain.ErrPhoneTaken):
		errs.WriteError(w, r, errs.PhoneTaken())
	case errors.Is(err, domain.ErrInvalidCredentials):
		errs.WriteError(w, r, errs.InvalidCredentials())
	default:
		errs.WriteError(w, r, errs.Internal("an error occurred"))
	}
}

func userDTO(u *domain.User) map[string]any {
	return map[string]any{
		"id":         u.ID,
		"name":       u.Name,
		"email":      u.Email,
		"phone":      u.Phone,
		"role":       u.Role,
		"is_active":  u.IsActive,
		"created_at": u.CreatedAt,
	}
}
