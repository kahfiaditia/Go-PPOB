# Arsitektur & Alur Bisnis

## 1. Peran Sistem

- **Distributor (Sistem Golang)**: Menyediakan API, mengelola saldo deposit agen, dan memproses transaksi pulsa.
- **Agen (Klien)**: Pihak yang mendaftar ke Distributor. Bebas membangun sistem dengan Java/Spring, Node.js, Go, PHP/Laravel, Python, atau teknologi lain yang mendukung HTTP/JSON.
- **Frontend agen**: Dapat memakai React atau framework UI lain. Untuk produksi, frontend sebaiknya memanggil backend milik agen; backend tersebut menyimpan kredensial API dan JWT dengan aman.
- **CMS Admin (Akan Datang)**: Dashboard khusus untuk Distributor (Anda) untuk memvalidasi transfer deposit manual dari Agen.

## 2. Alur Bisnis (End-to-End)

1. **Deposit Saldo**:
   - Agen melakukan transfer bank ke rekening Distributor.
   - Admin Distributor membuka CMS Admin, memvalidasi transfer, dan menambahkan saldo ke akun Agen di database Golang.
2. **Transaksi pulsa (simulasi)**:
   - Sistem Agen login ke API Distributor menggunakan email/password API yang disimpan di environment server Agen. Kredensial ini bukan password pengguna dashboard Agen.
   - Backend Agen menyimpan JWT di cache server-side dan mengirimkannya sebagai token Bearer.
   - Karyawan memasukkan PIN transaksi melalui dashboard Agen. Backend Agen mengirim PIN enam angka itu bersama nomor tujuan dan ID produk; password API tidak dikirim ulang untuk transaksi.
   - Sistem Golang memvalidasi JWT, PIN, produk, dan saldo. Worker memproses antrean, memotong harga jual final dari saldo, mencatat ledger, lalu mengirim callback jika dikonfigurasi.
   - Status `SUCCESS` pada versi belajar ini berarti simulasi internal berhasil. Belum ada koneksi ke provider/operator pulsa.
   - Jika antrean penuh atau server restart, transaksi `PENDING` tetap tersimpan dan worker akan mencoba mengambilnya kembali dari database.
3. **Penerimaan Status**:
   - Sistem Agen menerima Callback dari Distributor, lalu mengubah status transaksi di dashboard mereka dan menampilkannya ke pengguna akhir.
