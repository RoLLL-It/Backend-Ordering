package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	_ = godotenv.Load()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		slog.Error("DATABASE_URL not set")
		os.Exit(1)
	}
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminEmail == "" || adminPassword == "" {
		slog.Error("ADMIN_EMAIL and ADMIN_PASSWORD must be set")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		slog.Error("db connect", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	// Hash admin password
	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), 12)
	if err != nil {
		slog.Error("bcrypt", "error", err)
		os.Exit(1)
	}

	// Insert admin user (idempotent)
	_, err = pool.Exec(ctx,
		`INSERT INTO users(id,name,email,phone,password_hash,role)
		 VALUES($1,'Kitchen Admin',$2,'9000000001',$3,'ADMIN')
		 ON CONFLICT(email) DO NOTHING`,
		uuid.New(), adminEmail, string(hash),
	)
	if err != nil {
		slog.Error("seed admin", "error", err)
		os.Exit(1)
	}
	fmt.Println("✓ Admin user seeded:", adminEmail)

	// Seed sample menu items
	var vegRollsCatID string
	_ = pool.QueryRow(ctx, `SELECT id FROM categories WHERE name='Veg Rolls'`).Scan(&vegRollsCatID)
	if vegRollsCatID != "" {
		items := []struct {
			name  string
			desc  string
			price int
		}{
			{"Paneer Tikka Roll", "Grilled paneer, onion, mint mayo", 12000},
			{"Aloo Tikki Roll", "Spiced potato patty with chutney", 8000},
			{"Mixed Veg Roll", "Seasonal veggies, sauces", 9000},
		}
		for _, it := range items {
			_, _ = pool.Exec(ctx,
				`INSERT INTO menu_items(id,category_id,name,description,price_paise,is_veg,sort_order)
				 VALUES($1,$2,$3,$4,$5,true,$6)
				 ON CONFLICT DO NOTHING`,
				uuid.New(), vegRollsCatID, it.name, it.desc, it.price, 0,
			)
		}
		fmt.Println("✓ Sample menu items seeded")
	}

	// Seed slots for today at LJ
	var ljID string
	_ = pool.QueryRow(ctx, `SELECT id FROM locations WHERE code='LJ'`).Scan(&ljID)
	if ljID != "" {
		today := time.Now().UTC().Format("2006-01-02")
		slots := []struct{ start, end string }{
			{"12:30", "13:00"}, {"13:00", "13:30"}, {"13:30", "14:00"},
		}
		for _, s := range slots {
			_, _ = pool.Exec(ctx,
				`INSERT INTO delivery_slots(id,location_id,slot_date,start_time,end_time,capacity,cutoff_minutes)
				 VALUES($1,$2,$3,$4,$5,20,30)
				 ON CONFLICT DO NOTHING`,
				uuid.New(), ljID, today, s.start, s.end,
			)
		}
		fmt.Println("✓ Slots seeded for today at LJ")
	}

	fmt.Println("✓ Seed complete")
}
