package service

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SettingsService struct {
	db *pgxpool.Pool
}

func NewSettingsService(db *pgxpool.Pool) *SettingsService {
	return &SettingsService{db: db}
}

func (s *SettingsService) GetBool(ctx context.Context, key string) (bool, error) {
	var val string
	err := s.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key=$1`, key).Scan(&val)
	if err != nil {
		return false, err
	}
	return strconv.ParseBool(val)
}

func (s *SettingsService) GetString(ctx context.Context, key string) (string, error) {
	var val string
	err := s.db.QueryRow(ctx, `SELECT value FROM app_settings WHERE key=$1`, key).Scan(&val)
	return val, err
}

func (s *SettingsService) Set(ctx context.Context, key, value string) error {
	_, err := s.db.Exec(ctx,
		`INSERT INTO app_settings(key,value) VALUES($1,$2)
		 ON CONFLICT(key) DO UPDATE SET value=$2,updated_at=now()`, key, value)
	return err
}

type AppSettings struct {
	DeliveryEnabled bool
	KitchenOpen     bool
	Announcement    string
}

func (s *SettingsService) GetAll(ctx context.Context) (*AppSettings, error) {
	rows, err := s.db.Query(ctx, `SELECT key,value FROM app_settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	as := &AppSettings{}
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		switch k {
		case "delivery_enabled":
			as.DeliveryEnabled, _ = strconv.ParseBool(v)
		case "kitchen_open":
			as.KitchenOpen, _ = strconv.ParseBool(v)
		case "announcement":
			as.Announcement = v
		}
	}
	return as, nil
}
