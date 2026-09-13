package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"github.com/RoLLL-It/Backend-Ordering/internal/config"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/handler"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/middleware"
	"github.com/RoLLL-It/Backend-Ordering/internal/domain"
)

type Handlers struct {
	Auth      *handler.AuthHandler
	Menu      *handler.MenuHandler
	Order     *handler.OrderHandler
	Slot      *handler.SlotHandler
	Review    *handler.ReviewHandler
	AdminUser *handler.AdminUserHandler
}

func NewRouter(cfg *config.Config, h *Handlers, authMw func(http.Handler) http.Handler) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.CORS(cfg.CORSAllowedOrigins))
	r.Use(chimw.Timeout(30 * time.Second))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		// --- Auth ---
		r.Route("/auth", func(r chi.Router) {
			r.With(middleware.RegisterRateLimit).Post("/register", h.Auth.Register)
			r.With(middleware.AuthRateLimit).Post("/login", h.Auth.Login)
			r.Post("/refresh", h.Auth.Refresh)
			r.With(authMw).Post("/logout", h.Auth.Logout)
			r.With(authMw).Get("/me", h.Auth.Me)
		})

		// --- Menu (public reads) ---
		r.Get("/menu", h.Menu.GetMenu)
		r.Get("/menu/items/{id}", h.Menu.GetItem)

		// --- Locations + Slots (authed) ---
		r.Get("/locations", h.Slot.GetLocations)
		r.With(authMw).Get("/slots", h.Slot.ListSlots)

		// --- Cart validation (authed) ---
		r.With(authMw).Post("/cart/validate", h.Order.ValidateCart)

		// --- Reviews (public list, authed write) ---
		r.Get("/reviews", h.Review.List)
		r.Get("/reviews/summary", h.Review.Summary)
		r.With(authMw).Post("/reviews", h.Review.Create)
		r.With(authMw).Patch("/reviews/{id}", h.Review.Update)

		// --- Orders (authed) ---
		r.With(authMw).Route("/orders", func(r chi.Router) {
			r.Post("/", h.Order.CreateOrder)
			r.Get("/", h.Order.ListMyOrders)
			r.Get("/{id}", h.Order.GetOrder)
			r.Post("/{id}/cancel", h.Order.CancelOrder)
		})

		// --- Admin ---
		r.With(authMw).Route("/admin", func(r chi.Router) {
			r.Use(middleware.StaffOrAdmin)

			// Orders
			r.Get("/orders", h.Order.AdminListOrders)
			r.Patch("/orders/{id}/status", h.Order.AdminUpdateStatus)
			r.With(middleware.AdminOnly).Post("/orders/{id}/cancel", h.Order.AdminCancelOrder)

			// Menu
			r.Get("/menu/items", h.Menu.GetMenu) // reuses public but with auth
			r.With(middleware.AdminOnly).Post("/menu/items", h.Menu.AdminCreateItem)
			r.With(middleware.AdminOnly).Patch("/menu/items/{id}", h.Menu.AdminUpdateItem)
			r.Patch("/menu/items/{id}/availability", h.Menu.AdminSetAvailability)
			r.With(middleware.AdminOnly).Delete("/menu/items/{id}", h.Menu.AdminDeleteItem)
			r.With(middleware.AdminOnly).Post("/menu/categories", h.Menu.AdminCreateCategory)
			r.With(middleware.AdminOnly).Patch("/menu/categories/{id}", h.Menu.AdminUpdateCategory)

			// Slots
			r.Get("/slots", h.Slot.AdminListSlots)
			r.With(middleware.AdminOnly).Post("/slots", h.Slot.AdminCreateSlot)
			r.With(middleware.AdminOnly).Patch("/slots/{id}", h.Slot.AdminUpdateSlot)

			// Locations + settings
			r.With(middleware.AdminOnly).Patch("/locations/{id}", h.Slot.AdminUpdateLocation)
			r.With(middleware.AdminOnly).Patch("/settings", h.Slot.AdminUpdateSettings)

			// Users
			r.With(middleware.AdminOnly).Get("/users", h.AdminUser.List)
			r.With(middleware.AdminOnly).Patch("/users/{id}", h.AdminUser.Update)

			// Reviews
			r.With(middleware.AdminOnly).Patch("/reviews/{id}/hide", h.Review.AdminHide)

			// Stats
			r.With(middleware.AdminOnly).Get("/stats", func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"data":{"orders_today":0,"revenue_today_paise":0,"active_orders":0}}`))
			})
		})
	})

	// Role correction in middleware
	_ = domain.RoleAdmin // ensure import used

	return r
}
