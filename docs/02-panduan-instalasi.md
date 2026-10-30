# Panduan Instalasi & Menjalankan Server

## Prasyarat

- Go sesuai versi yang tercantum di `go.mod`
- MySQL (minimal versi 8.0)
- Git

## Langkah-langkah

1. **Clone repository**
   ```bash
   git clone https://github.com/kahfiaditia/Go-PPOB.git
   cd Go-PPOB
   ```

2. **Siapkan konfigurasi**
   ```bash
   cp -n .env.example .env
   ```

   Edit `.env` dengan kredensial MySQL lokal. `JWT_SECRET` wajib minimal 32
   karakter; gunakan nilai acak milik sendiri. Nilai seed di contoh hanya untuk
   pembelajaran lokal dan wajib diganti sebelum deployment.

   Buat secret acak menggunakan:
   ```bash
   openssl rand -hex 32
   ```
   Salin hasilnya ke nilai `JWT_SECRET` di `.env`.

3. **Buat skema database**
   ```bash
   mysql -u root -p < database/migration.sql
   ```

   `database/migration.sql` mendefinisikan skema lengkap untuk database baru.
   Pastikan `DB_NAME` di `.env` sama dengan nama database pada skrip. Skrip
   memakai `CREATE TABLE IF NOT EXISTS`, sehingga tidak mengubah tabel yang
   sudah ada.

   Untuk database yang masih memakai skema legacy sebelum akun pegawai dan
   permintaan deposit, backup database terlebih dahulu, lalu jalankan migrasi
   upgrade satu kali:
   ```bash
   mysqldump -u root -p go_pulsa_db > backup-go-pulsa.sql
   mysql -u root -p < database/migration_upgrade_legacy.sql
   ```
   Migrasi upgrade mengisi email kosong dari username dengan domain
   `example.invalid` agar akun lama tetap dapat login melalui email. Sesuaikan
   email tersebut setelah upgrade. Skrip ini ditujukan untuk skema legacy
   proyek; jangan jalankan ulang atau pada database dengan skema berbeda.

4. **Unduh dependency dan buat data demo**
   ```bash
   go mod download
   go run ./cmd/seed
   ```

   Seed membuat akun admin serta 60 akun owner (`agent01`–`agent60`) dan data
   produk contoh. Password dan PIN demo dibaca dari `SEED_ADMIN_PASSWORD`,
   `SEED_AGENT_PASSWORD`, dan `SEED_AGENT_PIN` pada `.env`. Seed tidak mengganti
   password akun yang sudah ada. Harga produk demo mengatur `price` Rp500–Rp1.000
   di atas nominal dan `harga_jual` Rp500 di atas `price`.

5. **Jalankan server**
   ```bash
   go run .
   ```

   Periksa server di `http://localhost:8080/health`. Respons health berisi
   status server dan panjang antrean worker. Secara default aplikasi hanya
   mendengarkan pada `127.0.0.1` agar HTTP tidak terbuka ke jaringan.

   Untuk deployment, letakkan aplikasi di belakang reverse proxy yang
   mengaktifkan HTTPS/TLS. Proxy pada host yang sama dapat meneruskan request
   ke `127.0.0.1:8080`. Jika proxy berjalan di host/container lain, atur
   `APP_HOST` ke alamat interface privat yang dapat dijangkau proxy dan batasi
   akses ke port aplikasi melalui firewall atau konfigurasi jaringan. Jangan
   mengekspos listener HTTP aplikasi langsung ke internet.

## Kredensial demo lokal

- Login admin: email dari `SEED_ADMIN_EMAIL` (default
  `admin@example.invalid`) dan password dari `SEED_ADMIN_PASSWORD`.
- Login owner: email `agent01@example.invalid` (atau email agen yang dibuat
  admin), password dari `SEED_AGENT_PASSWORD` untuk akun demo.
- PIN transaksi demo: nilai `SEED_AGENT_PIN`.

Username tetap disimpan sebagai identitas internal dan minimal 7 karakter.
Login API menggunakan email, bukan username. Email `.invalid` pada akun demo
adalah placeholder lokal; gunakan alamat email terverifikasi/milik Anda untuk
deployment.

Ganti kredensial dan PIN pada deployment. Seed mengatur PIN awal untuk akun
owner demo. Hanya owner yang dapat mengganti password login melalui
`PATCH /api/password` dan PIN melalui `PATCH /api/pin`; perubahan PIN
memerlukan PIN lama. Jika PIN hilang atau belum diatur, superadmin dapat
mengatur ulang melalui `PATCH /api/admin/agents/:id/pin`.
Admin distributor membuat akun owner melalui `POST /api/admin/agents` dan
pegawai melalui `POST /api/admin/agents/:id/employees`.

## Pemeriksaan

```bash
go test ./...
go vet ./...
```
