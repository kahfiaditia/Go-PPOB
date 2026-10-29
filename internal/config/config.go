package config

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv           string
	AppPort          string
	DBHost           string
	DBPort           string
	DBUser           string
	DBPass           string
	DBName           string
	JWTSecret        string
	JWTExpireHours   int
	WorkerCount      int
	QueueSize        int
	CallbackTimeout  int
	CallbackMaxRetry int
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		AppPort:          getEnv("APP_PORT", "8080"),
		DBHost:           getEnv("DB_HOST", "127.0.0.1"),
		DBPort:           getEnv("DB_PORT", "3306"),
		DBUser:           getEnv("DB_USER", "root"),
		DBPass:           getEnv("DB_PASS", ""),
		DBName:           getEnv("DB_NAME", "go_pulsa_db"),
		JWTSecret:        getEnv("JWT_SECRET", ""),
		JWTExpireHours:   getEnvInt("JWT_EXPIRE_HOURS", 24),
		WorkerCount:      getEnvInt("WORKER_COUNT", 10),
		QueueSize:        getEnvInt("QUEUE_SIZE", 1000),
		CallbackTimeout:  getEnvInt("CALLBACK_TIMEOUT_SECONDS", 5),
		CallbackMaxRetry: getEnvInt("CALLBACK_MAX_RETRY", 3),
	}

	if len(cfg.JWTSecret) < 32 {
		log.Fatal("[CONFIG] JWT_SECRET minimal 32 karakter untuk keamanan")
	}
	if cfg.JWTExpireHours <= 0 {
		log.Fatal("[CONFIG] JWT_EXPIRE_HOURS harus lebih besar dari 0")
	}
	if cfg.WorkerCount <= 0 {
		log.Fatal("[CONFIG] WORKER_COUNT harus lebih besar dari 0")
	}
	if cfg.QueueSize <= 0 {
		log.Fatal("[CONFIG] QUEUE_SIZE harus lebih besar dari 0")
	}
	if cfg.CallbackTimeout <= 0 {
		log.Fatal("[CONFIG] CALLBACK_TIMEOUT_SECONDS harus lebih besar dari 0")
	}
	if cfg.CallbackMaxRetry <= 0 {
		log.Fatal("[CONFIG] CALLBACK_MAX_RETRY harus lebih besar dari 0")
	}
	return cfg
}

func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&loc=Local",
		c.DBUser, c.DBPass, c.DBHost, c.DBPort, c.DBName)
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
		log.Fatalf("[CONFIG] %s harus berupa angka bulat", key)
	}
	return def
}
