# Panduan Instalasi & Menjalankan Server

## Prasyarat

- Go version go1.27.3 darwin/arm64
- MySQL (minimal versi 8.0)
- Git

## Langkah-langkah

1. **Clone Repository**
   ```bash
   git clone https://github.com/kahfiaditia/Go-PPOB.git
   cd pulsa-backend
   ```

- cp .env.example .env

- DB_USER=root
  DB_PASS=password_mysql_anda
  DB_NAME=pulsa_db
  JWT_SECRET=buat-string-random-minimal-32-karakter

  //cara buat di terminal ketik openssl rand -hex 32 atau
  echo "JWT_SECRET=$(openssl rand -hex 32)" >> .env

## Install Dependencies

- go mod download

Migrasi Database

- mysql -u root -p < internal/database/migration.sql

## Seed Data (Opsional)

go run ./cmd/seed

## Jalankan

go run main.go
