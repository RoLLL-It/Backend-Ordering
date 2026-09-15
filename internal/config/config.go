package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL         string
	JWTSecret           string
	JWTAccessTTL        time.Duration
	RefreshTTL          time.Duration
	BcryptCost          int
	Port                string
	CORSAllowedOrigins  string
	S3Endpoint          string
	S3Region            string
	S3Bucket            string
	S3AccessKey         string
	S3SecretKey         string
	LogLevel            string
	Env                 string
}

func Load() (*Config, error) {
	// Load .env if present (local dev); ignore error in production
	_ = godotenv.Load()

	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		Env:                getEnv("ENV", "development"),
	}

	var err error
	cfg.DatabaseURL, err = requireEnv("DATABASE_URL")
	if err != nil {
		return nil, err
	}
	cfg.JWTSecret, err = requireEnv("JWT_SECRET")
	if err != nil {
		return nil, err
	}
	// Neon Object Storage injects AWS-standard var names (AWS_ENDPOINT_URL_S3,
	// AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY, AWS_REGION). Fall back to the
	// legacy S3_* names for other S3-compatible providers (e.g. DO Spaces, R2).
	cfg.S3Endpoint, err = requireEnvAny("AWS_ENDPOINT_URL_S3", "S3_ENDPOINT")
	if err != nil {
		return nil, err
	}
	cfg.S3Bucket = getEnv("S3_BUCKET", "roll-it")
	cfg.S3Region = getEnv("AWS_REGION", "us-east-1")
	cfg.S3AccessKey, err = requireEnvAny("AWS_ACCESS_KEY_ID", "S3_ACCESS_KEY")
	if err != nil {
		return nil, err
	}
	cfg.S3SecretKey, err = requireEnvAny("AWS_SECRET_ACCESS_KEY", "S3_SECRET_KEY")
	if err != nil {
		return nil, err
	}

	cfg.JWTAccessTTL, err = parseDuration("JWT_ACCESS_TTL", "15m")
	if err != nil {
		return nil, err
	}
	cfg.RefreshTTL, err = parseDuration("REFRESH_TTL", "720h")
	if err != nil {
		return nil, err
	}
	cost, err := parseInt("BCRYPT_COST", "12")
	if err != nil {
		return nil, err
	}
	cfg.BcryptCost = cost

	return cfg, nil
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %q is not set", key)
	}
	return v, nil
}

// requireEnvAny returns the first non-empty value among the given keys,
// checked in order. Errors if none are set.
func requireEnvAny(keys ...string) (string, error) {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v, nil
		}
	}
	return "", fmt.Errorf("required environment variable not set (tried: %v)", keys)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseDuration(key, fallback string) (time.Duration, error) {
	v := getEnv(key, fallback)
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: %w", key, v, err)
	}
	return d, nil
}

func parseInt(key, fallback string) (int, error) {
	v := getEnv(key, fallback)
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: %w", key, v, err)
	}
	return n, nil
}
