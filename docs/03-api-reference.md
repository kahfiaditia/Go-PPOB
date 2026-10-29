# API Reference dan Integrasi Klien

Dokumen ini menjelaskan integrasi **server-to-server** antara sistem agen dan
API Golang distributor. Klien dapat menggunakan bahasa atau framework apa pun.

Contoh teknologi klien termasuk Java/Spring, Node.js, Go, PHP/Laravel, dan
Python. API cukup menerima request HTTP dengan payload JSON dan JWT. React
dapat digunakan sebagai frontend, tetapi jangan simpan kredensial API atau JWT
jangka panjang di browser. Gunakan pola `React → backend agen → API distributor`
agar rahasia API tetap berada di server.

- Base URL lokal: `http://localhost:8080`
- Format request/response: JSON
- Autentikasi: JWT melalui header `Authorization: Bearer <token>`

## Model kredensial

Ada tiga kredensial dengan fungsi berbeda:

| Kredensial | Disimpan/diketahui oleh | Digunakan untuk |
|---|---|---|
| Password aplikasi klien | Pengguna dan aplikasi klien | Login ke dashboard klien |
| Email/password API | Pemilik/programmer agen; disimpan di environment server klien | Login backend ke API Golang |
| PIN transaksi | Pemilik agen; hash disimpan di database Golang | Konfirmasi setiap transaksi pulsa |

Jangan pernah mengirim password aplikasi klien ke API Golang. Jangan menaruh
email, password API, atau JWT di JavaScript/browser. Sistem klien sebaiknya
memanggil API Golang dari backend, menyimpan JWT di cache server-side, dan
menampilkan hanya form PIN transaksi kepada karyawan.

Password API hanya dipakai untuk login. Setiap transaksi tetap memerlukan PIN,
sehingga token hasil login saja tidak cukup untuk membuat top-up. Semua request
yang membawa password, token, atau PIN harus menggunakan HTTPS saat deployment.

## 1. Login agent API

`POST /api/agent/login`

```json
{
  "email": "agent01@example.invalid",
  "password": "password-api-dari-environment-klien"
}
```

Login admin menggunakan format yang sama (`email` dan `password`) pada
`POST /api/admin/login`. Kolom `username` tetap disimpan sebagai identitas
internal dan minimal 7 karakter, tetapi bukan kredensial login.

Respons sukses:

```json
{
  "success": true,
  "message": "login success",
  "data": {
    "token": "<JWT>"
  }
}
```

Token berlaku selama jumlah jam pada `JWT_EXPIRE_HOURS` (default 24 jam).
Login ulang jika token kedaluwarsa. Jangan log response login atau token.

## 2. Daftar produk dan saldo

Semua endpoint agent berikut memerlukan JWT:

| Method | Endpoint | Keterangan |
|---|---|---|
| `GET` | `/api/products` | Daftar produk aktif |
| `GET` | `/api/products/:id` | Detail produk aktif |
| `GET` | `/api/balance` | Saldo agent yang terautentikasi |
| `GET` | `/api/history/transactions?limit=20&offset=0` | Riwayat transaksi agent |
| `GET` | `/api/history/deposits?limit=20&offset=0` | Riwayat deposit agent |
| `PATCH` | `/api/pin` | Mengganti PIN transaksi dengan verifikasi PIN lama |

Daftar produk menampilkan nominal pulsa dan harga jual default (`harga_jual`).
Harga modal distributor (`price`) hanya tersedia pada endpoint admin.

Contoh memasang token:

```http
Authorization: Bearer <JWT>
```

Ganti PIN dari dashboard pemilik dengan `PATCH /api/pin`:

```json
{
  "current_pin": "123456",
  "new_pin": "654321"
}
```

PIN harus enam angka. API menyimpan hash bcrypt, bukan PIN teks biasa. Endpoint
ini mewajibkan PIN lama sehingga JWT saja tidak cukup untuk mengambil alih PIN.
Lima PIN salah berturut-turut mengunci verifikasi selama 15 menit; reset oleh
superadmin menghapus lockout.
Jika PIN belum pernah diatur atau pemilik lupa PIN, superadmin dapat
memprovisikan ulang melalui `PATCH /api/admin/agents/:id/pin`.

## 3. Membuat transaksi

`POST /api/topup`

Header:

```http
Authorization: Bearer <JWT>
Content-Type: application/json
```

Body:

```json
{
  "product_id": "TSEL10",
  "customer_no": "081234567890",
  "keuntungan": 400,
  "pin": "123456"
}
```

Backend sistem agen mendapatkan PIN dari form dashboard karyawan, lalu
mengirimnya ke API. Password API tidak disertakan dalam request transaksi.
`keuntungan` bersifat opsional. Jika tidak dikirim, API memakai keuntungan
default `harga_jual - price`. Jika dikirim, nilainya adalah keuntungan total
di atas modal dan harus sama atau lebih besar dari keuntungan default. Harga
yang didebit dari saldo agen adalah `price + keuntungan`.

Respons diterima untuk diproses:

```json
{
  "success": true,
  "message": "transaction accepted for processing",
  "data": {
    "tx_id": "<transaction-id>",
    "status": "PENDING",
    "nominal": 10000,
    "sell_price": 10400
  }
}
```

API menggunakan `202 Accepted`: transaksi belum tentu sudah selesai saat
respons dikembalikan. Sistem agen dapat memeriksa status melalui
`GET /api/history/transactions`. Jika worker antrean penuh atau aplikasi restart,
transaksi pending tetap ada di database dan akan diambil ulang oleh worker.

## 4. Contoh integrasi Laravel (opsional)

Simpan kredensial di `.env` Laravel (jangan commit file tersebut):

```dotenv
PULSA_API_BASE_URL=https://api.example.com
PULSA_API_EMAIL=agent01@example.invalid
PULSA_API_PASSWORD=ganti-dengan-password-api
```

Contoh pemanggilan backend secara ringkas:

```php
use Illuminate\Support\Facades\Cache;
use Illuminate\Support\Facades\Http;

$token = Cache::remember('pulsa_api_token', now()->addHours(23), function () {
    $response = Http::asJson()->post(
        config('services.pulsa.base_url').'/api/agent/login',
        [
            'email' => config('services.pulsa.email'),
            'password' => config('services.pulsa.password'),
        ]
    )->throw();

    return $response->json('data.token');
});

$response = Http::withToken($token)->asJson()->post(
    config('services.pulsa.base_url').'/api/topup',
    [
        'product_id' => $request->string('product_id')->toString(),
        'customer_no' => $request->string('customer_no')->toString(),
        'pin' => $request->string('pin')->toString(),
    ]
)->throw();
```

Konfigurasikan `services.pulsa` untuk membaca tiga nilai environment di atas,
misalnya di `config/services.php`:

```php
'pulsa' => [
    'base_url' => env('PULSA_API_BASE_URL'),
    'email' => env('PULSA_API_EMAIL'),
    'password' => env('PULSA_API_PASSWORD'),
],
```

Validasi PIN di backend agen untuk pengalaman pengguna, tetapi validasi dan
verifikasi otoritatif tetap dilakukan API Golang. Jangan menyimpan PIN dalam
log, session yang berumur panjang, atau tabel transaksi.

Contoh cache di atas memakai masa cache lebih pendek daripada masa JWT default.
Untuk integrasi produksi, tangani respons `401` dengan menghapus token cache,
login ulang, lalu ulangi request secara aman. Jangan mengulang transaksi secara
buta jika request timeout setelah diterima server; cek riwayat/status dahulu.

## 5. Endpoint admin

Endpoint berikut memerlukan JWT admin:

| Method | Endpoint |
|---|---|
| `POST` | `/api/admin/login` |
| `POST` | `/api/admin/deposit` |
| `GET` | `/api/admin/agents` |
| `GET` | `/api/admin/transactions` |
| `GET` | `/api/admin/products` |
| `POST` | `/api/admin/products` |
| `PATCH` | `/api/admin/products/:id` |
| `DELETE` | `/api/admin/products/:id` |
| `PATCH` | `/api/admin/agents/:id/pin` (khusus superadmin) |

## Catatan status simulasi

Proyek ini untuk belajar dan belum mengirim pulsa ke provider/operator.
`SUCCESS` berarti proses simulasi internal berhasil dan saldo telah dipotong,
bukan konfirmasi pengiriman pulsa sesungguhnya.
