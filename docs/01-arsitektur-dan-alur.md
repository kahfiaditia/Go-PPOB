# Arsitektur & Alur Bisnis

## 1. Peran Sistem

- **Distributor (Sistem Golang)**: Menyediakan API, mengelola saldo deposit agen, dan memproses transaksi pulsa.
- **Agen (Klien)**: Pihak yang mendaftar ke Distributor. Memiliki sistem sendiri (Laravel, Node.js, dll) untuk melayani konter/pengguna akhir mereka.
- **CMS Admin (Akan Datang)**: Dashboard khusus untuk Distributor (Anda) untuk memvalidasi transfer deposit manual dari Agen.

## 2. Alur Bisnis (End-to-End)

1. **Deposit Saldo**:
   - Agen melakukan transfer bank ke rekening Distributor.
   - Admin Distributor membuka CMS Admin, memvalidasi transfer, dan menambahkan saldo ke akun Agen di database Golang.
2. **Transaksi Pulsa**:
   - Agen mengirim request ke API Distributor (membawa JWT Token hasil login, Nomor HP, dan ID Produk).
   - Sistem Golang memvalidasi saldo Agen.
   - Jika saldo cukup: Saldo dipotong, status transaksi di-set `SUCCESS` (atau masuk antrean worker), dan sistem mengirim _Callback_ ke URL sistem Agen.
   - Jika saldo kurang: Status transaksi `FAILED`, sistem tetap mengirim _Callback_ ke URL Agen.
3. **Penerimaan Status**:
   - Sistem Agen menerima Callback dari Distributor, lalu mengubah status transaksi di dashboard mereka dan menampilkannya ke pengguna akhir.
