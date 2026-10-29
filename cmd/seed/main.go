package main

import (
	"fmt"
	"log"
	"os"
	"regexp"

	"pulsa-backend/database"
	"pulsa-backend/internal/config"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg := config.Load()
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	adminPassword := requiredEnv("SEED_ADMIN_PASSWORD")
	agentPassword := requiredEnv("SEED_AGENT_PASSWORD")
	agentPIN := requiredEnv("SEED_AGENT_PIN")
	if !regexp.MustCompile(`^[0-9]{6}$`).MatchString(agentPIN) {
		log.Fatal("SEED_AGENT_PIN harus terdiri dari 6 angka")
	}

	// Admin
	adminHash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("failed to hash seed admin password: ", err)
	}
	_, err = db.Exec(`INSERT INTO admins (username, password_hash, name, role) VALUES (?, ?, ?, ?)
	                  ON DUPLICATE KEY UPDATE username = VALUES(username)`,
		getEnv("SEED_ADMIN_USERNAME", "admin"), string(adminHash), "Super Admin", "SUPERADMIN")
	if err != nil {
		log.Fatal(err)
	}

	// 60 Agents
	agentHash, err := bcrypt.GenerateFromPassword([]byte(agentPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("failed to hash seed agent password: ", err)
	}
	pinHash, err := bcrypt.GenerateFromPassword([]byte(agentPIN), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("failed to hash seed agent PIN: ", err)
	}
	for i := 1; i <= 60; i++ {
		username := fmt.Sprintf("agent%02d", i)
		_, err := db.Exec(`INSERT INTO agents (username, password_hash, pin_hash, name, balance, callback_url)
		                   VALUES (?, ?, ?, ?, 500000, ?)
		                   ON DUPLICATE KEY UPDATE username = VALUES(username),
		                   pin_hash = COALESCE(pin_hash, VALUES(pin_hash))`,
			username, string(agentHash), string(pinHash), fmt.Sprintf("Agen %02d", i),
			"http://localhost:9090/callback")
		if err != nil {
			log.Fatalf("agent %s: %v", username, err)
		}
	}

	// Products
	products := [][]interface{}{
		{"TSEL10", "TELKOMSEL", 10000, "Telkomsel 10.000", 10500, 0},
		{"TSEL25", "TELKOMSEL", 25000, "Telkomsel 25.000", 24500, 0},
		{"ISAT10", "INDOSAT", 10000, "Indosat 10.000", 10300, 0},
		{"XL10", "XL", 10000, "XL 10.000", 10200, 0},
		{"TRI10", "TRI", 10000, "Tri 10.000", 10000, 0},
	}
	for _, p := range products {
		_, err := db.Exec(`INSERT INTO products (id, operator, nominal, name, base_price, default_markup)
		                   VALUES (?, ?, ?, ?, ?, ?)
		                   ON DUPLICATE KEY UPDATE nominal = VALUES(nominal), name = VALUES(name),
		                   base_price = VALUES(base_price), default_markup = VALUES(default_markup)`,
			p[0], p[1], p[2], p[3], p[4], p[5])
		if err != nil {
			log.Fatalf("product %v: %v", p[0], err)
		}
	}

	log.Println("✅ Seed complete")
}

func requiredEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("%s wajib diatur pada environment atau file .env", key)
	}
	return value
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
