# Fitur Sistem & Panduan Troubleshooting

Dokumen ini ditujukan untuk **Developer** dan **Admin Internal Distributor**. Berisi penjelasan mengenai fitur teknis di balik layar serta panduan jika terjadi masalah (error) pada sistem.

---

## 🚀 1. Fitur Utama Sistem

Sistem backend ini dirancang untuk menangani transaksi pulsa berskala besar (multi-agen) dengan performa tinggi:

- **High Concurrency (Async Worker Pool):** Transaksi tidak diproses secara langsung di endpoint HTTP. API hanya memasukkan data ke antrean (Channel), lalu 20 Worker (Goroutine) memprosesnya di _background_. Ini mencegah server _bottleneck_ atau _timeout_ saat ada ribuan request bersamaan.
- **ACID Transaction & Row Locking:** Proses pemotongan saldo menggunakan `SELECT ... FOR UPDATE` di MySQL. Ini memastikan jika ada 2 request topup dari agen yang sama masuk bersamaan, saldo tidak akan minus (Race Condition).
- **Keamanan Berlapis:**
  - JWT (JSON Web Token) untuk autentikasi stateless.
  - Bcrypt untuk enkripsi password di database.
  - Rate Limiting (Anti-Bruteforce & Spam) per IP Address.
- **Audit Trail (Balance Ledger):** Setiap pergerakan saldo (potong transaksi, deposit, koreksi admin) **wajib** tercatat di tabel `balance_ledger`. Tidak ada saldo yang bisa berubah tanpa meninggalkan jejak.
- **Async Callback (Webhook):** Sistem mengirim status akhir transaksi ke URL Agen secara asinkron, dilengkapi dengan fitur _Retry_ otomatis jika server Agen sedang mati.

---

## 🛠️ 2. Troubleshooting (Panduan Perbaikan)

Jika terjadi error, ikuti panduan di bawah ini untuk melakukan _debugging_.

### A. Transaksi Berstatus `PENDING` Terus Menerus

**Gejala:** Agen sudah menembak API topup, dapat `tx_id`, tapi setelah beberapa menit status di database masih `PENDING` dan saldo tidak terpotong.
**Penyebab & Solusi:**

1. **Worker Pool Mati:** Cek log server Golang (terminal). Apakah ada tulisan `[WORKER] pool started with 20 workers`? Jika tidak, pastikan server berjalan normal.
2. **Antrean Penuh (Queue Overflow):** Cek endpoint `/health` dan nilai `queue`. Permintaan top-up yang sudah tercatat tetap berstatus `PENDING`; worker memindai database secara berkala dan mengambil transaksi pending ketika antrean tersedia. Jangan kirim ulang transaksi hanya karena antrean sedang penuh.
3. **Error di Background:** Cek log Golang, cari prefix `[TOPUP]` atau `[WORKER]`. Worker juga memindai ulang transaksi pending setelah aplikasi restart. Pastikan database tetap tersedia dan tabel menggunakan skema terbaru.

### B. Status Transaksi `SUCCESS`, Tapi Saldo Agen Tidak Berkurang

**Gejala:** Callback sudah terkirim, status transaksi sukses, tapi saldo agen tetap.
**Penyebab & Solusi:**
Ini haram terjadi karena kita menggunakan Database Transaction (ACID). Jika status `SUCCESS`, maka saldo _pasti_ terpotong. Jika aneh, ikuti langkah ini:

1. Cek tabel `balance_ledger` untuk `agent_id` tersebut.
2. Cari data dengan `ref_type = 'TRANSACTION'` dan `ref_id` (sesuai ID transaksi).
3. Lihat kolom `amount` (harus minus, misal `-10500`) dan `balance_after`.
4. Bandingkan `balance_after` terakhir di ledger dengan saldo di tabel `agents`. Jika beda, jalankan query koreksi: `UPDATE agents SET balance = (SELECT balance_after FROM balance_ledger WHERE agent_id = X ORDER BY id DESC LIMIT 1) WHERE id = X;`

### C. Agen Komplain Tidak Menerima Callback

**Gejala:** Transaksi di sistem kita `SUCCESS`, tapi dashboard agen tetap `PENDING` (karena mereka tidak menerima callback).
**Penyebab & Solusi:**

1. Buka tabel `callback_logs` di database.
2. Cari transaksi tersebut berdasarkan `tx_id`.
3. Perhatikan kolom `success` dan `http_status`:
   - Jika `http_status = 0`: Server agen mati, tidak bisa dihubungi (Connection Refused / Timeout).
   - Jika `http_status = 500`: Ada error di kode server agen mereka.
   - Jika `http_status = 404`: URL callback yang agen berikan salah.
4. **Solusi:** Hubungi agen, beritahu error di atas. Sistem saat ini tidak menyediakan endpoint untuk mengirim ulang callback secara manual; implementasikan retry administratif jika fitur tersebut dibutuhkan.

### D. Muncul Error `429 Too Many Requests`

**Gejala:** Agen tidak bisa melakukan request, muncul error 429.
**Penyebab & Solusi:**
Sistem mendeteksi IP agen mengirim lebih dari 10 request per detik (Rate Limit).

1. Beritahu agen untuk memberi jeda (delay) atau _queue_ di sistem mereka.
2. Jika memang traffic-nya wajar (agen besar), Anda bisa menaikkan limit di file `internal/middleware/ratelimit.go` pada baris `middleware.NewRateLimiter(10, 20)`. Ubah angka 10 menjadi 50 (misal).

### E. Error Database: `Lock wait timeout exceeded`

**Gejala:** Di log muncul error ini saat memproses topup.
**Penyebab & Solusi:**
Ada transaksi database yang menggantung terlalu lama dan mengunci baris (`FOR UPDATE`). Biasanya karena koneksi ke MySQL terputus tiba-tiba.

1. Cek proses di MySQL: `SHOW PROCESSLIST;`
2. Matikan proses yang menggantung (_Kill_ thread id-nya).
3. Pastikan setting `SetConnMaxLifetime(5 * time.Minute)` di `database/mysql.go` sudah benar agar koneksi _zombie_ tidak menumpuk.

---

## 💡 3. Tips Maintenance Rutin

- **Backup Database:** Lakukan _dump_ database MySQL (`mysqldump`) secara harian (cronjob), terutama tabel `agents`, `transactions`, dan `balance_ledger`.
- **Pemantauan Log:** Pantau terminal server atau file log secara berkala. Waspadai munculnya kata `[CALLBACK] FAIL` atau `[PANIC] recovered`.
- **Cek Konsistensi Data (Mingguan):**
  Jalankan query ini untuk memastikan saldo di tabel `agents` cocok dengan akumulasi di `balance_ledger`:

  ```sql
  SELECT a.id, a.balance, SUM(l.amount) AS ledger_sum, (a.balance - SUM(l.amount)) AS selisih
  FROM agents a
  LEFT JOIN balance_ledger l ON l.agent_id = a.id
  GROUP BY a.id
  HAVING selisih != 0;



  ### Kenapa file ini penting dimasukkan ke Git?
  ```

1. **Buku Sakti Developer:** Jika Anda mengerjakan proyek ini sendirian, 3 bulan dari sekarang Anda mungkin lupa cara kerja _Worker Pool_ atau bagaimana mengecek konsistensi saldo. File ini adalah pengingat Anda sendiri.
2. **Handover (Serah Terima):** Jika suatu hari Anda merekrut developer baru untuk menggantikan atau membantu Anda, mereka tinggal membaca file `05` ini, dan mereka langsung paham cara _debugging_-nya tanpa harus bertanya banyak.
3. **Cegah Kepanikan:** Jika suatu saat sistem error atau agen komplain, Anda tinggal buka file ini, ikuti langkahnya (seperti _step-by-step_ di atas), dan masalah selesai dengan tenang.
