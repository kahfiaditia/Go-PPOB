package database

import (
	"database/sql"
	"time"

	"pulsa-backend/internal/config"
	"pulsa-backend/util"

	_ "github.com/go-sql-driver/mysql"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, err
	}

	// Tuning connection pool — penting untuk 50+ agent
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	util.Log.Info("[DB] Connected to MySQL: ", cfg.DBName)
	return db, nil
}
