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
		DatabaseURL:        requireEnv("DATABASE_URL"),
		JWTSecret:          requireEnv("JWT_SECRET"),
		S3Endpoint:         requireEnv("S3_ENDPOINT"),
		S3Bucket:           requireEnv("S3_BUCKET"),
		S3AccessKey:        requireEnv("S3_ACCESS_KEY"),
		S3SecretKey:        requireEnv("S3_SECRET_KEY"),
		Port:               getEnv("PORT", "8080"),
		CORSAllowedOrigins: getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000"),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		Env:                getEnv("ENV", "development"),
	}

	var err error
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

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic(fmt.Sprintf("required environment variable %q is not set", key))
	}
	return v
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
