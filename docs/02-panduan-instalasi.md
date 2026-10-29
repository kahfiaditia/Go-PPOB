# Panduan Instalasi & Menjalankan Server

## Prasyarat

- Go sesuai versi yang tercantum di `go.mod`
- MySQL (minimal versi 8.0)
- Git

## Langkah-langkah

1. **Clone repository**
   ```bash
   git clone <URL-REPOSITORY>
   cd pulsa_go
   ```

2. **Siapkan konfigurasi**
   ```bash
   cp -n .env.example .env
   ```

   Edit `.env` dengan kredensial MySQL lokal. `JWT_SECRET` wajib minimal 32
   karakter; gunakan nilai acak milik sendiri. Nilai seed di contoh hanya untuk
   pembelajaran lokal dan wajib diganti sebelum deployment.

3. **Buat skema database**
   ```bash
   mysql -u root -p < database/migration.sql
   ```

   Migration membuat database `go_pulsa_db`. Pastikan `DB_NAME` di `.env` sama.
   Perintah ini ditujukan untuk database baru. Database lama yang dibuat dari
   versi skema terdahulu perlu di-backup dan dimigrasikan secara terpisah.

4. **Unduh dependency dan buat data demo**
   ```bash
   go mod download
   go run ./cmd/seed
   ```

   Seed membuat akun admin serta 60 akun agent (`agent01`–`agent60`) dan data
   produk contoh. Password dan PIN demo dibaca dari `SEED_ADMIN_PASSWORD`,
   `SEED_AGENT_PASSWORD`, dan `SEED_AGENT_PIN` pada `.env`. Seed tidak mengganti
   password akun yang sudah ada.

5. **Jalankan server**
   ```bash
   go run .
   ```

   Periksa server di `http://localhost:8080/health`. Respons health berisi
   status server dan panjang antrean worker.

## Kredensial demo lokal

- Login admin: username dari `SEED_ADMIN_USERNAME`, password dari
  `SEED_ADMIN_PASSWORD`.
- Login agent: username `agent01` (atau agent lain yang dibuat seed), password
  dari `SEED_AGENT_PASSWORD`.
- PIN transaksi demo: nilai `SEED_AGENT_PIN`.

Ganti kredensial dan PIN pada deployment. Seed mengatur PIN awal untuk akun
demo. Agent harus memberikan PIN lama untuk menggantinya melalui `PATCH
/api/pin`; jika PIN hilang atau belum diatur, superadmin dapat mengatur ulang
melalui `PATCH /api/admin/agents/:id/pin`.

## Pemeriksaan

```bash
go test ./...
go vet ./...
```
