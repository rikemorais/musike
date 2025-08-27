package database

import (
	"database/sql"
	"log"

	_ "github.com/lib/pq"
	"musike-backend/internal/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Connected to PostgreSQL database")
	return db, nil
}
