package main

import (
	"database/sql"
	"fmt"
	"log"

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

	// Admin
	adminHash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	_, err = db.Exec(`INSERT INTO admins (username, password_hash, name, role) VALUES (?, ?, ?, ?)
	                  ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash)`,
		"admin", string(adminHash), "Super Admin", "SUPERADMIN")
	if err != nil {
		log.Fatal(err)
	}

	// 60 Agents
	agentHash, _ := bcrypt.GenerateFromPassword([]byte("agent123"), bcrypt.DefaultCost)
	for i := 1; i <= 60; i++ {
		username := fmt.Sprintf("agent%02d", i)
		_, err := db.Exec(`INSERT INTO agents (username, password_hash, name, balance, callback_url)
		                   VALUES (?, ?, ?, 500000, ?)
		                   ON DUPLICATE KEY UPDATE password_hash = VALUES(password_hash)`,
			username, string(agentHash), fmt.Sprintf("Agen %02d", i),
			"http://localhost:9090/callback")
		if err != nil {
			log.Printf("agent %s: %v", username, err)
		}
	}

	// Products
	products := [][]interface{}{
		{"TSEL10", "TELKOMSEL", "Telkomsel 10.000", 10500},
		{"TSEL25", "TELKOMSEL", "Telkomsel 25.000", 24500},
		{"ISAT10", "INDOSAT", "Indosat 10.000", 10300},
		{"XL10", "XL", "XL 10.000", 10200},
		{"TRI10", "TRI", "Tri 10.000", 10000},
	}
	for _, p := range products {
		_, err := db.Exec(`INSERT INTO products (id, operator, name, price)
		                   VALUES (?, ?, ?, ?)
		                   ON DUPLICATE KEY UPDATE price = VALUES(price)`,
			p[0], p[1], p[2], p[3])
		if err != nil {
			log.Printf("product %v: %v", p[0], err)
		}
	}

	log.Println("✅ Seed complete")
	_ = sql.ErrNoRows
}
