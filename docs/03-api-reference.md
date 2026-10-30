# API Reference dan Integrasi Klien

Dokumen ini menjelaskan integrasi **server-to-server** antara sistem agen dan
API Golang distributor. Klien dapat menggunakan bahasa/framework apa pun yang
mendukung HTTP dan JSON.

- Base URL lokal: `http://localhost:8080`
- Format request/response: JSON
- Autentikasi: `Authorization: Bearer USER_JWT`
- Semua request yang membawa password, token, atau PIN harus memakai HTTPS
  saat deployment.

## 1. Login owner atau pegawai

Owner dan pegawai memiliki akun distributor masing-masing dan memakai endpoint
login yang sama. Owner dibuat saat pendaftaran agen; admin distributor
membuat akun pegawai.

`POST /api/agent/login`

```json
{
  "email": "pegawai@example.invalid",
  "password": "password-pengguna"
}
```

Respons sukses berisi JWT pengguna. JWT memuat `agent_id`, `sub` pengguna, dan
role `owner` atau `employee`. API menolak akun yang dinonaktifkan dan
memvalidasi status akun saat setiap request. Jangan log password atau token.

Login admin distributor menggunakan `POST /api/admin/login` dengan format
`email` dan `password` yang sama, tetapi menghasilkan token admin yang tidak
dapat dipakai pada endpoint agen.

Sistem agen harus menyimpan JWT di server-side session per pengguna, bukan di
browser atau cache global yang dipakai bersama owner dan pegawai.

## 2. Produk, operator, saldo, dan laporan

Semua endpoint agent berikut memerlukan JWT owner atau pegawai:

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/api/products?operator=TELKOMSEL` | Produk aktif; operator dipilih manual |
| `GET` | `/api/products/:id` | Detail produk aktif |
| `GET` | `/api/pricing` | Keuntungan default per produk |
| `PATCH` | `/api/pricing/:id` | Mengubah keuntungan default (owner saja) |
| `GET` | `/api/balance` | Saldo agen |
| `GET` | `/api/dashboard/summary` | Ringkasan KPI hari ini |
| `GET` | `/api/history/transactions` | Riwayat dengan filter/pagination |
| `GET` | `/api/history/deposits` | Deposit yang disetujui |
| `POST` | `/api/deposits` | Membuat permintaan deposit |
| `GET` | `/api/history/deposit-requests` | Riwayat permintaan deposit |
| `PATCH` | `/api/pin` | Mengganti PIN transaksi (owner saja) |
| `PATCH` | `/api/password` | Mengganti password owner (owner saja) |
| `POST` | `/api/topup` | Membuat transaksi pulsa |

Operator dipilih manual dari katalog produk. Sistem tidak menebak atau
memvalidasi operator berdasarkan prefix nomor tujuan.

Filter `GET /api/history/transactions`:

- `date_from`, `date_to`: format `YYYY-MM-DD`, keduanya inklusif.
- `status`: `PENDING`, `SUCCESS`, `FAILED`, atau `REFUNDED`.
- `operator`: kode operator sesuai katalog, misalnya `TELKOMSEL`.
- `limit` (default 20, maksimum 100) dan `offset` (default 0).

Tanggal mengikuti zona waktu server aplikasi. `GET /api/dashboard/summary`
mengembalikan saldo saat ini, jumlah semua transaksi hari ini, jumlah
transaksi sukses, total keuntungan, total saldo terpotong, serta operator
terbanyak. Keuntungan, saldo terpotong, dan operator terbanyak hanya dihitung
dari transaksi `SUCCESS`; hari mengikuti zona waktu server.

### Keuntungan default per produk

Default keuntungan untuk produk yang belum dikonfigurasi adalah Rp500. Owner
dapat mengubah default melalui:

```http
PATCH /api/pricing/TSEL10
Authorization: Bearer OWNER_JWT
Content-Type: application/json
```

```json
{
  "default_profit": 1000
}
```

Nilai default maupun keuntungan yang dikirim saat top-up harus Rp500–Rp2.000.
Pegawai yang memanggil endpoint ini akan menerima `403 Forbidden`.

### Mengganti PIN dan password

PIN harus enam angka dan API menyimpan hash bcrypt. Penggantian PIN memerlukan
PIN lama:

```http
PATCH /api/pin
Authorization: Bearer OWNER_JWT
Content-Type: application/json
```

```json
{
  "current_pin": "123456",
  "new_pin": "654321"
}
```

Lima PIN salah berturut-turut mengunci verifikasi selama 15 menit. Jika PIN
belum diatur atau owner lupa, superadmin dapat mengatur ulang melalui
`PATCH /api/admin/agents/:id/pin`.

Owner mengganti password login miliknya sendiri melalui `PATCH /api/password`:

```json
{
  "current_password": "password-lama",
  "new_password": "password-baru"
}
```

Password minimal enam karakter. Pegawai tidak dapat memakai endpoint
perubahan PIN, password, atau keuntungan default.

## 3. Membuat transaksi

`POST /api/topup`

```http
Authorization: Bearer USER_JWT
Content-Type: application/json
```

```json
{
  "product_id": "TSEL10",
  "customer_no": "081234567890",
  "keuntungan": 1200,
  "pin": "123456"
}
```

`keuntungan` opsional. Jika tidak dikirim, API memakai keuntungan default
untuk produk tersebut, atau Rp500 jika belum diatur. `price` adalah modal
distributor, sedangkan saldo agen didebit sebesar `harga_jual`. Harga konsumen
adalah `harga_jual + keuntungan`.

Respons `202 Accepted`:

```json
{
  "success": true,
  "message": "transaction accepted for processing",
  "data": {
    "tx_id": "<transaction-id>",
    "status": "PENDING",
    "nominal": 10000,
    "sell_price": 11000,
    "keuntungan": 1200,
    "customer_price": 12200
  }
}
```

Transaksi belum tentu selesai saat respons diterima. Periksa status melalui
`GET /api/history/transactions`. Jika worker antrean penuh atau aplikasi
restart, transaksi pending tersimpan dan akan diambil ulang oleh worker.

## 4. Permintaan deposit

Setelah transfer manual, owner atau pegawai mengajukan deposit:

`POST /api/deposits`

```json
{
  "amount": 100000,
  "reference": "REF-TRANSFER-001"
}
```

Permintaan berstatus `PENDING` dan belum menambah saldo. Admin memeriksa
transfer melalui:

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/api/admin/deposit-requests?status=PENDING` | Daftar permintaan |
| `POST` | `/api/admin/deposit-requests/:id/approve` | Setujui dan tambahkan saldo |
| `POST` | `/api/admin/deposit-requests/:id/reject` | Tolak tanpa mengubah saldo |

Request approve/reject menerima JSON `{"note":"..."}`. Keputusan hanya dapat
dilakukan sekali; permintaan yang sudah diputus menghasilkan `409 Conflict`.
`reference` wajib diisi dan unik per agen agar transfer yang sama tidak dapat
diajukan dua kali.
Riwayat permintaan ada di `/api/history/deposit-requests`; deposit yang telah
disetujui ada di `/api/history/deposits`.

## 5. Pengelolaan akun agen oleh admin distributor

Semua endpoint berikut memerlukan JWT admin:

| Method | Endpoint |
|---|---|
| `POST` | `/api/admin/agents` |
| `GET` | `/api/admin/agents` |
| `GET` | `/api/admin/agents/:id/employees` |
| `POST` | `/api/admin/agents/:id/employees` |
| `PATCH` | `/api/admin/agents/:id/employees/:employee_id` |
| `GET` | `/api/admin/transactions` |
| `PATCH` | `/api/admin/agents/:id/pin` (khusus superadmin) |
| `POST` | `/api/admin/agents/:id/callback-secret` (khusus superadmin) |

`POST /api/admin/agents` membuat akun owner dan menerima `username`, `email`,
`password`, `name`, `phone`, `address`, `bank_name`, serta
`bank_account_number`. `POST /api/admin/agents/:id/employees` menerima
`username`, `email`, `name`, dan `password`. `PATCH` akun pegawai menerima
`is_active` untuk mengaktifkan atau menonaktifkannya. Akun pegawai yang
dinonaktifkan langsung ditolak pada request API berikutnya.

Data alamat dan rekening bank hanya tersedia bagi admin distributor melalui
pengelolaan profil agen.

Endpoint produk admin:

| Method | Endpoint |
|---|---|
| `GET` | `/api/admin/products` |
| `POST` | `/api/admin/products` |
| `PATCH` | `/api/admin/products/:id` |
| `DELETE` | `/api/admin/products/:id` |

## Catatan status simulasi

Proyek ini untuk belajar dan belum mengirim pulsa ke provider/operator.
`SUCCESS` berarti proses simulasi internal berhasil dan saldo telah dipotong,
bukan konfirmasi pengiriman pulsa sesungguhnya.
