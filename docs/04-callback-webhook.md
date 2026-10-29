# Panduan Webhook (Callback API)

Dokumen ini ditujukan untuk **Agen/Klien**. Karena proses transaksi topup pulsa berjalan secara _Asynchronous_ (di belakang layar/antrean), Agen **wajib** menyediakan endpoint (URL) penerima Callback agar sistem Agen bisa mengetahui hasil akhir dari transaksi (Berhasil atau Gagal).

## 1. Cara Kerja Callback

1. Agen mengirim request topup ke `/api/topup` dan mendapatkan `tx_id` dengan status `PENDING`.
2. Sistem Distributor memproses transaksi tersebut di _background worker_.
3. Setelah selesai, Sistem Distributor akan mengirim HTTP `POST` ke URL Callback milik Agen.
4. Sistem Agen menerima data, lalu mengubah status transaksi (dari `PENDING` menjadi `SUCCESS` atau `FAILED`) di database Agen.

## 2. Persyaratan untuk Agen

- Agen harus memiliki endpoint publik yang bisa diakses oleh server Distributor (contoh: `https://sistem-agen.com/api/distributor-callback`).
- URL tersebut harus didaftarkan ke Admin Distributor (via CMS Admin atau manual di database).
- Endpoint Agen harus merespons dengan HTTP Status `200 OK` dalam waktu kurang dari 5 detik.
- Jika endpoint Agen mati atau merespons error, sistem Distributor akan mencoba mengirim ulang (Retry) sebanyak **3 kali** dengan jeda waktu bertahap.

---

## 3. Spesifikasi Payload (Data yang Dikirim)

Sistem Distributor akan mengirim data dengan format **JSON**.

**Headers Request:**

```http
Content-Type: application/json
X-Callback-Timestamp: 1793311200
X-Callback-Signature: sha256=<64 karakter hex>
```

Setiap agen memiliki secret callback sendiri. Superadmin membuat atau merotasi
secret melalui:

```http
POST /api/admin/agents/{id}/callback-secret
Authorization: Bearer <admin-jwt>
```

Nilai `data.callback_secret` hanya ditampilkan pada respons rotasi tersebut.
Simpan di secret manager milik penerima; rotasi secret membuat signature lama
tidak lagi valid. Setelah perubahan skema, agen lama belum mempunyai secret dan
callback tidak akan dikirim sampai secret dibuat.

Signature adalah HMAC-SHA256 menggunakan teks secret sebagai key, dengan pesan
persis `timestamp + "." + raw_request_body`. Digest dikodekan sebagai hex
lowercase dan diberi awalan `sha256=`. Penerima harus:

1. Memastikan timestamp berupa Unix seconds dan berbeda dari waktu server tidak
   lebih dari 5 menit.
2. Menghitung ulang HMAC atas timestamp dan byte body mentah, lalu
   membandingkannya dengan `hmac.Equal` (jangan memakai perbandingan string
   biasa).
3. Menolak signature/timestamp tidak valid sebelum memproses payload, serta
   memproses `tx_id` secara idempoten untuk menghindari efek ganda pada retry.

Timestamp dan signature baru dibuat pada setiap percobaan pengiriman. Jangan
mencatat atau membagikan secret callback.

Contoh payload:

```json
{
  "tx_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
  "status": "SUCCESS",
  "customer_no": "081234567890",
  "product_id": "TSEL10",
  "amount": 10000,
  "reason": "",
  "timestamp": "2026-10-29T14:00:00Z"
}
```

Untuk database yang sudah ada, jalankan
`database/migrations/20261029_add_agent_callback_secret.sql` sebelum
menjalankan versi aplikasi ini. Database baru memperoleh kolom tersebut dari
`database/migration.sql`.
