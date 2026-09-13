package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/RoLLL-It/Backend-Ordering/internal/config"
	apphttp "github.com/RoLLL-It/Backend-Ordering/internal/http"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/handler"
	"github.com/RoLLL-It/Backend-Ordering/internal/http/middleware"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/db"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/token"
	"github.com/RoLLL-It/Backend-Ordering/internal/repo"
	"github.com/RoLLL-It/Backend-Ordering/internal/service"
)

func main() {
	// Setup structured logging
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Database
	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	slog.Info("database connected")

	// Token manager
	tokenMgr := token.NewManager(cfg.JWTSecret, cfg.JWTAccessTTL)

	// Repos
	userRepo := repo.NewUserRepo(pool)
	menuRepo := repo.NewMenuRepo(pool)
	slotRepo := repo.NewSlotRepo(pool)
	orderRepo := repo.NewOrderRepo(pool)
	reviewRepo := repo.NewReviewRepo(pool)

	// Services
	settingsSvc := service.NewSettingsService(pool)
	locationSvc := service.NewLocationService(pool)
	authSvc := service.NewAuthService(userRepo, tokenMgr, cfg.BcryptCost, cfg.RefreshTTL)
	menuSvc := service.NewMenuService(menuRepo)
	orderSvc := service.NewOrderService(pool, orderRepo, menuRepo, slotRepo, locationSvc, settingsSvc)
	reviewSvc := service.NewReviewService(reviewRepo, menuRepo, orderRepo)

	// Middleware
	isSecure := cfg.Env == "production"
	authMw := middleware.Authenticate(tokenMgr)

	// Handlers
	handlers := &apphttp.Handlers{
		Auth:      handler.NewAuthHandler(authSvc, cfg.RefreshTTL, isSecure),
		Menu:      handler.NewMenuHandler(menuSvc, settingsSvc, locationSvc),
		Order:     handler.NewOrderHandler(orderSvc, menuSvc, locationSvc),
		Slot:      handler.NewSlotHandler(slotRepo, locationSvc, settingsSvc),
		Review:    handler.NewReviewHandler(reviewSvc),
		AdminUser: handler.NewAdminUserHandler(userRepo),
	}

	router := apphttp.NewRouter(cfg, handlers, authMw)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		slog.Info("starting server", "port", cfg.Port, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-quit
	slog.Info("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown", "error", err)
	}
	slog.Info("server stopped")
}
