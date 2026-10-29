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
X-Callback-Signature: <HMAC-SHA256-Signature> (Opsional, untuk keamanan ekstra)

{
  "tx_id": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
  "status": "SUCCESS",
  "customer_no": "081234567890",
  "product_id": "TSEL10",
  "amount": 10500,
  "reason": "",
  "timestamp": "2026-10-29T14:00:00Z"
}
```
