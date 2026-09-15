package fn

import (
	"context"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"github.com/RoLLL-It/Backend-Ordering/internal/config"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/db"
	"github.com/RoLLL-It/Backend-Ordering/internal/platform/token"
)

var (
	bootstrapOnce sync.Once
	sharedPool    *pgxpool.Pool
	sharedToken   *token.Manager
	sharedCfg     *config.Config
	bootstrapErr  error
)

// Bootstrap initializes the shared DB pool, token manager, and config once per process.
// Subsequent calls return the same instances (warm invocations reuse them).
func Bootstrap() (*pgxpool.Pool, *token.Manager, *config.Config, error) {
	bootstrapOnce.Do(func() {
		_ = godotenv.Load() // ignored in production if .env absent
		cfg, err := config.Load()
		if err != nil {
			bootstrapErr = fmt.Errorf("config: %w", err)
			return
		}
		pool, err := db.Connect(context.Background(), cfg.DatabaseURL)
		if err != nil {
			bootstrapErr = fmt.Errorf("db: %w", err)
			return
		}
		sharedPool = pool
		sharedToken = token.NewManager(cfg.JWTSecret, cfg.JWTAccessTTL)
		sharedCfg = cfg
	})
	return sharedPool, sharedToken, sharedCfg, bootstrapErr
}
