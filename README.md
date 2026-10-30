# Pulsa Backend - Distributor API

Backend API untuk sistem distribusi pulsa multi-agen. Dibangun dengan Golang, MySQL, Gin, dan Worker Pool untuk performa konkurensi tinggi.

## 📚 Dokumentasi Lengkap

Semua dokumentasi teknis dan panduan telah dipisahkan ke dalam folder `docs/`:

- [Arsitektur & Alur Bisnis](docs/01-arsitektur-dan-alur.md)
- [Panduan Instalasi](docs/02-panduan-instalasi.md)
- [API Reference](docs/03-api-reference.md)
- [Callback / Webhook](docs/04-callback-webhook.md)
- [Fitur & Troubleshooting](docs/05-fitur-dan-troubleshooting.md)

## ⚡ Quick Start

Lihat [panduan instalasi](docs/02-panduan-instalasi.md) untuk langkah lengkap.

```bash
cp -n .env.example .env
# Sesuaikan koneksi MySQL dan JWT_SECRET di .env
mysql -u root -p < database/migration.sql
go mod download
go run ./cmd/seed
go run .
```

Seed demo membuat akun `admin` dan `agent01` sampai `agent10`. Password dan PIN
diambil dari `.env`; ganti nilainya sebelum membagikan atau men-deploy aplikasi.
Jangan commit file `.env`.

API dapat diintegrasikan server-to-server oleh sistem agen dengan bahasa atau
framework apa pun (misalnya Java, Node.js, Go, atau Laravel). Owner dan pegawai
memiliki akun API masing-masing dengan role pada JWT. Backend agen menyimpan
token pengguna aktif di server dan mengirim PIN transaksi bersama data top-up.
React atau frontend lain dapat digunakan, tetapi jangan simpan password/JWT di
browser. Admin distributor mengelola akun pegawai. Baca
[panduan integrasi API](docs/03-api-reference.md).

Top-up pada proyek ini adalah simulasi pemrosesan internal; belum terhubung ke
provider/operator pulsa.
