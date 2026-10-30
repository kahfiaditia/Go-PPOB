# Arsitektur & Alur Bisnis

## Peran sistem dan autentikasi

- **Distributor (sistem Golang)** menyediakan API untuk agen, mengelola akun
  owner/pegawai, profil, produk, saldo dan ledger, transaksi top-up, serta
  callback.
- **Sistem agen** adalah dashboard terpisah yang memakai API distributor.
  Backend agen menyimpan JWT dan meneruskannya sesuai sesi pengguna; token dan
  password tidak boleh disimpan di browser.
- Setiap agen memiliki satu akun **owner**. Akun **pegawai** disimpan terpisah
  dan ditautkan ke agen. Login menghasilkan JWT per pengguna dengan role
  `owner` atau `employee`; API menggunakan role pada token, bukan role dari
  request.
- Admin distributor membuat akun agen/owner dan membuat atau menonaktifkan
  akun pegawai. Owner dan pegawai dapat melihat saldo, riwayat akun agen, dan
  membuat transaksi. Hanya owner dapat mengubah password owner, PIN, dan
  keuntungan default.
- Data pendaftaran agen mencakup email, username, nama, telepon, alamat, serta
  nama dan nomor rekening bank. CMS admin dan frontend agen belum tersedia.

## Harga dan keuntungan

- `price` adalah harga modal distributor. `harga_jual` adalah harga distributor
  yang didebit dari saldo agen.
- Default keuntungan ditetapkan **per produk untuk setiap agen**. Nilai
  default maupun nilai yang dikirim saat transaksi harus Rp500–Rp2.000.
- Default untuk produk yang belum dikonfigurasi adalah Rp500. Owner dapat
  mengubah default melalui API pricing.
- Harga konsumen adalah `harga_jual + keuntungan`. Saldo agen hanya dipotong
  sebesar `harga_jual`; keuntungan tidak menambah debit distributor.

## Pemilihan operator

Operator dipilih manual sebelum transaksi. Prefix nomor tidak digunakan untuk
menebak atau memvalidasi operator; pengguna memilih produk/operator dari
katalog aktif melalui `GET /api/products` (opsional filter `operator`).

## Alur bisnis

1. **Pendaftaran agen**: admin distributor membuat akun owner melalui
   `POST /api/admin/agents`. Profil bisnis dan informasi rekening tersimpan di
   database distributor.
2. **Pengelolaan pegawai**: admin distributor membuat, melihat, dan
   menonaktifkan akun pegawai melalui API admin.
3. **Deposit saldo**:
   - Owner atau pegawai membuat permintaan deposit setelah transfer manual.
   - Admin memeriksa transfer, lalu menyetujui atau menolak permintaan.
   - Saldo hanya bertambah saat permintaan disetujui; perubahan saldo dan
     ledger dicatat dalam transaksi database yang sama.
4. **Transaksi top-up**:
   - Owner atau pegawai login dengan akun masing-masing; backend agen memakai
     JWT pengguna tersebut untuk memanggil API.
   - Pengguna memilih operator/produk, memasukkan nomor tujuan, lalu PIN
     transaksi. Backend agen mengirim ID produk, nomor tujuan, PIN, dan
     keuntungan opsional.
   - API memvalidasi JWT, PIN, produk, keuntungan, dan saldo, lalu menyimpan
     transaksi `PENDING`.
   - Worker memproses antrean. Jika saldo cukup, sistem memotong `harga_jual`,
     mencatat ledger, lalu memperbarui status.
   - `SUCCESS` pada versi ini berarti simulasi internal berhasil; belum ada
     koneksi ke provider/operator pulsa.
   - Jika antrean penuh atau server restart, transaksi pending tersimpan dan
     worker mengambilnya kembali dari database.
5. **Status transaksi**: API mengirim callback jika URL dan secret callback
   agen sudah dikonfigurasi. Sistem agen memverifikasi callback dan
   memperbarui status pada dashboard.

## Dashboard dan laporan

`GET /api/dashboard/summary` menyediakan saldo, jumlah transaksi hari ini,
jumlah transaksi sukses, keuntungan dari transaksi sukses, saldo yang
terpotong pada transaksi sukses, dan operator paling sering untuk transaksi
sukses. Hari dihitung menurut zona waktu server. Riwayat transaksi agen dapat
difilter dengan tanggal, status, dan operator.

Dashboard dan pencetakan struk tetap menjadi tanggung jawab frontend/backend
sistem agen. Harga konsumen untuk struk tersedia sebagai
`customer_price` pada transaksi.

## Batasan yang masih ada

- Frontend agen dan CMS admin belum dibuat; alur tersedia melalui API.
- Logout dilakukan pada sistem agen dengan menghapus sesi/JWT yang disimpan.
  JWT berlaku sesuai konfigurasi masa token; akun pegawai yang dinonaktifkan
  langsung ditolak pada request API berikutnya.
- Deposit dan transaksi pulsa masih berupa alur transfer manual dan simulasi;
  belum ada integrasi bank atau provider/operator pulsa.
