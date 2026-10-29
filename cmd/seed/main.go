package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"unicode/utf8"

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
	adminEmail := getEnv("SEED_ADMIN_EMAIL", "admin@example.invalid")
	adminUsername := getEnv("SEED_ADMIN_USERNAME", "superadmin")
	if utf8.RuneCountInString(adminUsername) < 7 {
		log.Printf("SEED_ADMIN_USERNAME must be at least 7 characters; using superadmin")
		adminUsername = "superadmin"
	}
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
	_, err = db.Exec(`INSERT INTO admins (username, email, password_hash, name, role) VALUES (?, ?, ?, ?, ?)
	                  ON DUPLICATE KEY UPDATE username = VALUES(username)`,
		adminUsername, adminEmail, string(adminHash), "Super Admin", "SUPERADMIN")
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
		email := fmt.Sprintf("agent%02d@example.invalid", i)
		_, err := db.Exec(`INSERT INTO agents (username, email, password_hash, pin_hash, name, balance, callback_url)
		                   VALUES (?, ?, ?, ?, ?, 500000, ?)
		                   ON DUPLICATE KEY UPDATE username = VALUES(username),
		                   pin_hash = COALESCE(pin_hash, VALUES(pin_hash))`,
			username, email, string(agentHash), string(pinHash), fmt.Sprintf("Agen %02d", i),
			"http://localhost:9090/callback")
		if err != nil {
			log.Fatalf("agent %s: %v", username, err)
		}
	}

	// Products
	products := seedProducts()
	for _, p := range products {
		_, err := db.Exec(`INSERT INTO products (id, operator, nominal, name, price, harga_jual)
		                   VALUES (?, ?, ?, ?, ?, ?)
		                   ON DUPLICATE KEY UPDATE nominal = VALUES(nominal), name = VALUES(name),
		                   price = VALUES(price), harga_jual = VALUES(harga_jual)`,
			p.id, p.operator, p.nominal, p.name, p.price, p.hargaJual)
		if err != nil {
			log.Fatalf("product %s: %v", p.id, err)
		}
	}

	log.Println("✅ Seed complete")
}

type seedProduct struct {
	id        string
	operator  string
	nominal   float64
	name      string
	price     float64
	hargaJual float64
}

func seedProducts() []seedProduct {
	return []seedProduct{
		{id: "TSEL10", operator: "TELKOMSEL", nominal: 10000, name: "Telkomsel 10.000", price: 10500, hargaJual: 11000},
		{id: "TSEL25", operator: "TELKOMSEL", nominal: 25000, name: "Telkomsel 25.000", price: 25500, hargaJual: 26000},
		{id: "ISAT10", operator: "INDOSAT", nominal: 10000, name: "Indosat 10.000", price: 10700, hargaJual: 11200},
		{id: "XL10", operator: "XL", nominal: 10000, name: "XL 10.000", price: 10800, hargaJual: 11300},
		{id: "TRI10", operator: "TRI", nominal: 10000, name: "Tri 10.000", price: 11000, hargaJual: 11500},
	}
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
