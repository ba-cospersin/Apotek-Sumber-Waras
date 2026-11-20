# Sistem Stok Apotek Berbasis Batch — Apotek Sumber Waras

Aplikasi web untuk membantu pengelolaan persediaan obat di Apotek Sumber Waras. Sistem dirancang dengan pencatatan berdasarkan identitas batch fisik, lapisan penerimaan, tanggal kedaluwarsa, dan riwayat pergerakan stok.

**Status:** Dalam pengembangan  
**Jenis Proyek:** Proyek Kelompok — Pemrograman Web

> **Status repository:** Fondasi backend saat ini mencakup konfigurasi, koneksi database, migration runner, seed runner, middleware dasar, dan endpoint pemeriksaan `/health`. Modul autentikasi dan fitur bisnis masih dalam tahap pengembangan. Daftar fitur pada README ini menjelaskan ruang lingkup dan target sistem, bukan klaim bahwa seluruh fitur sudah berjalan.

---

## Daftar Isi

1. [Tentang Proyek](#1-tentang-proyek)
2. [Fitur Sistem](#2-fitur-sistem)
3. [Hak Akses Pengguna](#3-hak-akses-pengguna)
4. [Aturan Bisnis Utama](#4-aturan-bisnis-utama)
5. [Gambaran Database](#5-gambaran-database)
6. [Teknologi dan Library](#6-teknologi-dan-library)
7. [Persiapan dan Instalasi](#7-persiapan-dan-instalasi)
8. [Migration Database](#8-migration-database)
9. [Seed Data Dummy](#9-seed-data-dummy)
10. [Menjalankan Aplikasi](#10-menjalankan-aplikasi)
11. [Struktur Folder Proyek](#11-struktur-folder-proyek)
12. [Pembagian Tanggung Jawab dan Pedoman Pengembangan](#12-pembagian-tanggung-jawab-dan-pedoman-pengembangan)
13. [Dokumentasi Tambahan](#13-dokumentasi-tambahan)
14. [Batasan Sistem](#14-batasan-sistem)

---

## 1. Tentang Proyek

Apotek Sumber Waras masih menghadapi kendala dalam pencatatan persediaan obat karena pengelolaan stok dilakukan melalui buku dan spreadsheet. Cara tersebut berisiko menimbulkan ketidaksesuaian antara stok yang tercatat dan stok fisik, serta menyulitkan pemantauan tanggal kedaluwarsa dan penelusuran asal stok obat.

Proyek ini bertujuan membangun sistem informasi berbasis web untuk membantu proses pengelolaan obat, mulai dari penerimaan barang dari Pedagang Besar Farmasi (PBF), penjualan, pengembalian obat, pemeriksaan stok, hingga penyusunan laporan.

Sistem menggunakan pendekatan **batch-based inventory**. Stok ditelusuri berdasarkan identitas batch fisik dan lapisan penerimaannya. Dengan pendekatan ini, obat yang sama dapat memiliki beberapa catatan penerimaan dengan jumlah, harga beli, dan waktu penerimaan yang berbeda, meskipun berasal dari batch fisik yang sama.

## 2. Fitur Sistem

Bagian ini menjelaskan ruang lingkup fitur yang dikembangkan. Tidak seluruh fitur sudah tersedia pada server saat ini.

### 2.1 Data Master

- Mengelola data obat.
- Mengelola satuan obat dan konversi ke satuan dasar.
- Mengatur harga jual berdasarkan satuan.
- Mengelola data PBF atau pemasok.
- Mengelola akun pengguna sesuai hak akses OWNER.

### 2.2 Penerimaan Barang

- Mencatat penerimaan obat berdasarkan faktur PBF.
- Mencatat nomor batch dan tanggal kedaluwarsa.
- Mencatat kuantitas dan harga beli.
- Memisahkan identitas batch fisik dari lapisan penerimaan.
- Menambah stok berdasarkan jumlah yang benar-benar diterima.

### 2.3 Penjualan

- Menampilkan obat dan batch yang dapat dijual.
- Mengonversi jumlah penjualan ke satuan dasar.
- Mengambil stok menggunakan prinsip FEFO (*First Expired, First Out*).
- Mencatat alokasi stok berdasarkan lapisan penerimaan.
- Mencatat pembayaran tunai maupun nontunai secara manual.
- Menghitung total pembayaran dan kembalian.
- Mencetak struk transaksi.

### 2.4 Retur Konsumen

- Mencatat pengembalian obat dengan mengacu pada transaksi penjualan asli.
- Menelusuri batch dan lapisan penerimaan asal obat.
- Mencatat jumlah, alasan, dan kondisi obat yang dikembalikan.
- Memisahkan obat yang menunggu pemeriksaan, layak jual, dan tidak layak jual.
- Mengembalikan obat ke stok yang dapat dijual hanya setelah dinyatakan layak jual.

### 2.5 Persediaan dan Stock Opname

- Melihat jumlah stok berdasarkan obat dan batch.
- Menelusuri asal penerimaan stok.
- Memantau batch yang diblokir.
- Melakukan pencatatan stock opname.
- Mencatat penyesuaian stok berdasarkan lapisan penerimaan.
- Mencatat pemusnahan atau retur obat kepada PBF.

### 2.6 Dashboard dan Laporan

- Menampilkan ringkasan kondisi persediaan.
- Menampilkan obat dengan stok minimum.
- Menampilkan batch yang mendekati kedaluwarsa.
- Menampilkan batch yang diblokir.
- Menampilkan riwayat stock opname.
- Menampilkan laporan penjualan, keuntungan kotor, dan kerugian persediaan.

## 3. Hak Akses Pengguna

Sistem memiliki tiga role pengguna.

| Role | Tanggung Jawab |
|---|---|
| **OWNER** | Mengelola akun pengguna dan data master, memantau dashboard, serta melihat laporan. |
| **KASIR** | Melakukan transaksi penjualan dan mencatat retur konsumen. |
| **GUDANG** | Mengelola penerimaan barang, memantau persediaan, melakukan stock opname, serta mencatat pemusnahan dan retur ke PBF. |

Setiap pengguna harus login untuk mengakses fitur sesuai role-nya. Sistem tidak menyediakan registrasi akun secara publik. Pengelolaan akun dilakukan oleh OWNER.

Tidak ada role `SUPERADMIN` terpisah dalam rancangan database. Akun `OWNER` digunakan untuk pengelolaan sistem sesuai hak akses yang ditetapkan.

## 4. Aturan Bisnis Utama

1. **Satuan dasar persediaan**  
   Stok disimpan dalam satuan dasar masing-masing obat. Penjualan dapat dilakukan menggunakan satuan lain yang memiliki konversi ke satuan dasar.

2. **Pencatatan batch**  
   Satu identitas batch fisik dapat memiliki beberapa lapisan penerimaan apabila diterima melalui faktur atau waktu penerimaan yang berbeda. Setiap lapisan penerimaan menyimpan jumlah dan harga beli masing-masing.

3. **FEFO**  
   Pengeluaran stok mengikuti tanggal kedaluwarsa terdekat. Apabila tanggal kedaluwarsa sama, lapisan penerimaan yang lebih dahulu diterima diprioritaskan.

4. **Pembatasan penjualan**  
   Obat dengan tanggal kedaluwarsa satu bulan atau kurang dari tanggal transaksi tidak dapat dijual. Batch yang diblokir juga tidak dapat digunakan untuk penjualan.

5. **Retur konsumen**  
   Retur harus mengacu pada transaksi penjualan yang berhasil. Jumlah kumulatif obat yang dikembalikan tidak boleh melebihi jumlah yang dibeli pada alokasi penjualan terkait.

6. **Pemeriksaan obat retur**  
   Obat yang dikembalikan tidak otomatis menjadi stok yang dapat dijual. Obat harus melalui pemeriksaan dan ditetapkan status kelayakannya terlebih dahulu.

7. **Retur ke PBF dan pemusnahan**  
   Retur ke PBF dan pemusnahan dicatat sebagai proses terpisah dari retur konsumen. Jumlah yang dicatat tidak boleh melebihi stok yang masih tersedia pada lapisan penerimaan terkait.

8. **Riwayat dan perhitungan keuntungan**  
   Harga jual dan harga beli pada transaksi disimpan sebagai snapshot agar laporan historis tetap konsisten meskipun harga master berubah.

9. **Konsistensi stok**  
   Setiap transaksi yang memengaruhi stok harus diproses menggunakan transaksi database dan validasi ulang agar dua kasir tidak menggunakan stok yang sama secara bersamaan.

## 5. Gambaran Database

Database menggunakan MySQL/MariaDB dengan 14 tabel inti sesuai ERD proyek.

| Kelompok | Tabel |
|---|---|
| Pengguna | `users` |
| Data master | `suppliers`, `medicines`, `medicine_units` |
| Penerimaan dan batch | `purchases`, `batches`, `batch_receipts` |
| Penjualan | `sales`, `sale_items`, `sale_item_batches` |
| Retur konsumen | `returns` |
| Stock opname | `stock_opnames`, `stock_opname_details` |
| Tindak lanjut stok | `batch_disposals` |

### 5.1 Tabel Pengguna

Tabel `users` menyimpan informasi akun dengan kolom utama berikut:

| Kolom | Keterangan |
|---|---|
| `nama` | Nama pengguna. |
| `username` | Identitas username dan harus unik. |
| `email` | Alamat email dan harus unik. |
| `password_hash` | Hash password menggunakan bcrypt, bukan password teks biasa. |
| `role` | Role pengguna: `OWNER`, `KASIR`, atau `GUDANG`. |
| `is_active` | Status aktif akun. |

Migration `002_add_email_to_users.sql` menambahkan kolom `email`, mengisi email sementara untuk akun lama, menjadikannya wajib diisi, dan menambahkan unique key.

Rancangan form login menggunakan satu field `identifier` yang dapat menerima username atau email, bersama field password. Backend membedakan pencarian berdasarkan format nilai `identifier`. Username tidak boleh menggunakan karakter `@` agar tidak menimbulkan ambiguitas.

Email demo menggunakan domain `.test` dan tidak ditujukan untuk pengiriman email sungguhan.

### 5.2 Tabel Teknis

Selain 14 tabel inti, terdapat tabel teknis berikut:

- `schema_migrations`: dibuat oleh migration runner untuk mencatat file migration yang berhasil diterapkan.
- `seed_runs`: dibuat oleh seed runner untuk mencatat file seed yang berhasil dijalankan.
- `sessions`: dibuat melalui migration `003_add_sessions.sql` untuk mendukung rancangan autentikasi server-side.

Tabel teknis bukan bagian dari 14 tabel bisnis pada ERD. Tabel `sessions` tetap diperlukan pada rancangan autentikasi karena token session akan disimpan dan diperiksa di sisi server melalui database.

### 5.3 Prinsip Struktur Database

- Stok tidak disimpan langsung pada tabel `medicines`; jumlah stok ditelusuri dari lapisan penerimaan pada `batch_receipts`.
- `batches` menyimpan identitas batch fisik. `batch_receipts` menyimpan penerimaan batch pada faktur tertentu.
- Satu batch fisik dapat memiliki beberapa lapisan penerimaan dengan jumlah dan harga beli yang berbeda.
- `sale_items` menyimpan rincian obat pada transaksi penjualan, sedangkan `sale_item_batches` mencatat alokasi stok dari lapisan penerimaan.
- `returns` merujuk pada alokasi batch dari transaksi penjualan asal.
- `stock_opname_details` mencatat penyesuaian pada setiap lapisan penerimaan yang terdampak.
- `batch_disposals` mencatat pemusnahan atau retur ke PBF, terpisah dari retur konsumen.
- Tabel `sessions` direncanakan untuk menghubungkan token session dengan akun pengguna serta menyimpan informasi waktu yang dibutuhkan untuk validasi masa berlaku session.

Perubahan struktur tabel harus dilakukan melalui file migration baru dan dikoordinasikan dengan Backend Lead. Jangan mengubah struktur database secara manual tanpa memperbarui migration.

## 6. Teknologi dan Library

### 6.1 Teknologi Utama

| Teknologi | Kegunaan |
|---|---|
| Go (Golang) | Backend dan server aplikasi. |
| HTML | Struktur halaman web. |
| CSS | Tampilan antarmuka. |
| JavaScript | Interaksi pada halaman web. |
| MySQL/MariaDB | Penyimpanan data relasional. |
| Git dan GitHub | Version control dan kolaborasi tim. |

### 6.2 Library Eksternal Go

| Modul | Package | Kegunaan |
|---|---|---|
| `github.com/go-sql-driver/mysql` | `github.com/go-sql-driver/mysql` | Driver MySQL untuk `database/sql`. |
| `github.com/joho/godotenv` | `github.com/joho/godotenv` | Membaca konfigurasi dari file `.env`. |
| `golang.org/x/crypto` | `golang.org/x/crypto/bcrypt` | Hashing dan verifikasi password menggunakan bcrypt pada implementasi autentikasi. |

`bcrypt` merupakan package di dalam modul `golang.org/x/crypto`, bukan modul terpisah. Modul tersebut sudah tercantum pada `go.mod`, tetapi pemanggilan bcrypt belum tersedia pada modul autentikasi yang masih berupa placeholder. Apabila `go mod tidy` dijalankan sebelum package digunakan, dependency yang belum di-import dapat dihapus. Periksa perubahan `go.mod` dan `go.sum` sebelum commit.

### 6.3 Package Bawaan Go

Package bawaan Go yang digunakan atau direncanakan dalam proyek antara lain:

- `net/http` — server, HTTP handler, dan cookie.
- `html/template` — menampilkan halaman HTML.
- `database/sql` — antarmuka interaksi database.
- `context` — pengaturan batas waktu dan pembatalan operasi.
- `os` — membaca environment variable dan file.
- `log` — pencatatan log aplikasi.
- `time` — pengelolaan tanggal dan waktu.
- `crypto/rand` — menghasilkan token acak untuk session.
- `encoding/hex` — mengubah token acak ke format hexadecimal.
- `testing` — pengujian aplikasi.

Package bawaan Go tidak perlu diinstal menggunakan `go get`.

## 7. Persiapan dan Instalasi

### 7.1 Persyaratan

Pastikan perangkat telah memiliki:

- Go versi sesuai direktif `go` pada `go.mod` (repository saat ini mencantumkan Go `1.27.0`).
- MySQL atau MariaDB. Untuk pengembangan lokal, database dapat dijalankan melalui XAMPP.
- Git.
- Browser untuk mengakses aplikasi.

### 7.2 Clone Repository

```bash
git clone https://github.com/ba-cospersin/Apotek-Sumber-Waras.git
cd Apotek-Sumber-Waras
```

Jika alamat repository berubah, gunakan URL clone yang ditampilkan pada halaman GitHub repository.

### 7.3 Mengunduh Dependency

Setelah melakukan clone repository, jalankan:

```bash
go mod download
```

Perintah tersebut mengunduh dependency yang tercantum pada `go.mod`.

`go mod tidy` tidak perlu dijalankan setiap kali membuat atau mengubah satu file `.go`. Jalankan ketika terdapat perubahan import atau dependency, lalu periksa perubahan pada `go.mod` dan `go.sum` sebelum melakukan commit.

Setiap perubahan dependency harus dikoordinasikan agar seluruh anggota menggunakan versi yang konsisten.

### 7.4 Konfigurasi Environment

Salin file `.env.example` menjadi `.env`.

**PowerShell:**

```powershell
Copy-Item .env.example .env
```

**Linux/macOS:**

```bash
cp .env.example .env
```

File `.env.example` pada repository saat ini berisi nilai placeholder:

```env
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=PORT_DB
DB_USER=USERNAME
DB_PASSWORD=PASSWORD
DB_NAME=namaDB
```

Setelah menyalin file tersebut, ganti nilai placeholder sesuai konfigurasi MySQL/MariaDB lokal. Contoh konfigurasi untuk MySQL lokal:

```env
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=apotek_sumber_waras
```

Keterangan:

- Pastikan MySQL/MariaDB sudah berjalan.
- `DB_HOST` berisi host database.
- `DB_PORT` berisi port MySQL/MariaDB, umumnya `3306`.
- `DB_USER` dan `DB_PASSWORD` disesuaikan dengan akun database lokal.
- `DB_NAME` harus sesuai dengan nama database yang dibuat.
- **`SESSION_KEY` tidak digunakan dan tidak perlu ditambahkan.** Rancangan autentikasi menggunakan token session acak yang disimpan di database, bukan cookie bertanda tangan yang memerlukan secret key aplikasi.
- Jangan menggunakan kredensial contoh untuk lingkungan produksi.
- File `.env` berisi konfigurasi lokal dan tidak boleh diunggah ke GitHub.
- Gunakan `.env.example` sebagai acuan konfigurasi bersama.
- Pastikan `.env` tidak ter-commit. Jika sebelumnya sudah masuk pelacakan Git, keluarkan dari index menggunakan `git rm --cached .env`.

> **Catatan repository:** ZIP pengembangan dapat saja memuat file `.env` lokal meskipun `.gitignore` sudah mengabaikannya. Sebelum membagikan ZIP kepada anggota lain atau mengunggahnya ke tempat publik, keluarkan `.env` dari arsip.

### 7.5 Membuat Database Kosong

Buat database kosong dengan nama yang sama seperti `DB_NAME` pada `.env`.

Contoh melalui phpMyAdmin atau MySQL client:

```sql
CREATE DATABASE apotek_sumber_waras
CHARACTER SET utf8mb4
COLLATE utf8mb4_unicode_ci;
```

Pembuatan database kosong dilakukan satu kali pada setiap lingkungan lokal. Tabel dan relasi tidak perlu dibuat secara manual karena akan dibuat oleh migration.

## 8. Migration Database

Migration digunakan untuk membuat dan memperbarui struktur database, termasuk tabel, kolom, primary key, foreign key, unique key, dan indeks.

### 8.1 File Migration yang Tersedia

Migration yang tersedia pada ZIP repository saat ini:

```text
migrations/
├── 001_init.sql
├── 002_add_email_to_users.sql
└── 003_add_sessions.sql
```

Fungsi masing-masing migration:

| Migration | Fungsi |
|---|---|
| `001_init.sql` | Membuat 14 tabel inti berdasarkan ERD. |
| `002_add_email_to_users.sql` | Menambahkan kolom `email` pada tabel `users`, mengisi email sementara untuk akun yang sudah ada, menjadikan email wajib diisi, dan menambahkan unique key. |
| `003_add_sessions.sql` | Membuat tabel teknis `sessions` untuk rancangan autentikasi server-side. |

Migration `002_add_email_to_users.sql` diperlukan agar struktur database mendukung login menggunakan username atau email. Untuk database yang sudah berisi akun dari migration awal, migration ini mengisi email sementara berdasarkan username. Domain `.test` hanya digunakan untuk pengembangan.

### 8.2 Menjalankan Migration

Pastikan database kosong sudah dibuat dan konfigurasi `.env` sudah benar. Jalankan perintah berikut dari direktori utama repository:

```bash
go run ./cmd/migrate
```

Migration runner saat ini:

1. Membaca konfigurasi `.env`.
2. Menghubungkan aplikasi ke MySQL/MariaDB.
3. Membuat tabel teknis `schema_migrations` jika belum tersedia.
4. Membaca seluruh file `migrations/*.sql`.
5. Mengurutkan file berdasarkan nama.
6. Memeriksa apakah setiap migration sudah pernah diterapkan.
7. Menjalankan migration yang belum tercatat.
8. Mencatat migration yang berhasil diterapkan pada `schema_migrations`.

Dengan penamaan berurutan, `001_init.sql` dijalankan sebelum `002_add_email_to_users.sql`. Ketika perintah dijalankan kembali, migration yang sudah tercatat akan dilewati.

**Catatan teknis:** Migration runner memisahkan perintah SQL berdasarkan tanda titik koma (`;`). Karena itu, file migration saat ini ditujukan untuk statement SQL biasa dan tidak boleh mengandalkan titik koma di dalam string literal atau stored procedure. Perintah DDL MySQL dapat melakukan implicit commit; apabila migration gagal di tengah proses, periksa kondisi tabel dan kolom sebelum mencoba menjalankan ulang file tersebut.

### 8.3 Migration Session untuk Autentikasi

File `migrations/003_add_sessions.sql` **sudah tersedia pada repository saat ini**. Migration ini membuat tabel teknis `sessions` untuk rancangan autentikasi server-side. Tabel tersebut bukan bagian dari 14 tabel inti ERD.

Kolom utama yang disediakan:

- `user_id`: relasi ke akun pada tabel `users`; satu akun dibatasi memiliki satu baris session aktif melalui unique key.
- `token_hash`: hash token session yang disimpan di database, bukan token asli dari cookie.
- `created_at`: waktu pembuatan session.
- `last_activity_at`: waktu aktivitas terakhir.
- `expires_at`: batas maksimum berlakunya session.

Migration runner membaca seluruh `migrations/*.sql` secara berurutan sehingga migration ini diproses setelah migration `001` dan `002`. Namun, tersedianya tabel belum berarti login/session telah berfungsi. Repository session, service autentikasi, handler login/logout, dan integrasi middleware masih perlu diselesaikan.

### 8.4 Menambahkan Migration Berikutnya

Apabila ada perubahan struktur database, buat file migration baru dengan nomor berurutan. Contoh setelah migration session ditambahkan:

```text
migrations/
├── 001_init.sql
├── 002_add_email_to_users.sql
├── 003_add_sessions.sql
└── 004_nama_perubahan_berikutnya.sql
```

Setiap migration baru harus:

- Memiliki nama dan nomor urut yang belum pernah digunakan.
- Dikoordinasikan dengan Backend Lead.
- Diuji pada database pengembangan sebelum digabungkan.
- Tidak mengubah file migration lama yang sudah diterapkan anggota lain.

### 8.5 Mengulang Database dari Awal

Saat ini belum tersedia perintah `migrate fresh` seperti pada Laravel. Untuk mengulang database pengembangan lokal dari kondisi kosong:

1. Pastikan database hanya berisi data pengembangan dan tidak memiliki data penting.
2. Hapus database lokal melalui phpMyAdmin atau MySQL client.
3. Buat kembali database kosong dengan nama yang sama.
4. Jalankan migration.
5. Jalankan seed.

Contoh perintah SQL untuk menghapus database pengembangan:

```sql
DROP DATABASE apotek_sumber_waras;
```

Kemudian buat kembali database menggunakan perintah pada bagian 7.5.

> **Peringatan:** `DROP DATABASE` menghapus seluruh tabel dan data di dalam database. Jangan menjalankan perintah ini pada database operasional atau database yang berisi data penting.

## 9. Seed Data Dummy

Seed digunakan untuk memasukkan data dummy ke tabel yang sudah dibuat oleh migration. Seed tidak membuat tabel inti atau kolom database; pembuatan struktur tetap menjadi tanggung jawab migration.

### 9.1 File Seed yang Tersedia

```text
seed/
├── 001_users.sql
├── 002_master_data.sql
├── 003_purchases_and_batches.sql
├── 004_sales.sql
├── 005_returns.sql
├── 006_stock_opname.sql
└── 007_batch_disposals.sql
```

Isi masing-masing file:

| File | Isi |
|---|---|
| `001_users.sql` | Akun OWNER, KASIR, dan GUDANG, termasuk username dan email demo. |
| `002_master_data.sql` | PBF, data obat, dan satuan obat. |
| `003_purchases_and_batches.sql` | Faktur penerimaan, identitas batch, dan lapisan penerimaan. |
| `004_sales.sql` | Transaksi penjualan, rincian, dan alokasi batch. |
| `005_returns.sql` | Contoh retur konsumen dengan status kelayakan berbeda. |
| `006_stock_opname.sql` | Hasil stock opname dan detail penyesuaian. |
| `007_batch_disposals.sql` | Contoh pemusnahan dan retur ke PBF. |

Data dummy disiapkan agar anggota tim dapat menguji fitur menggunakan data awal yang seragam. Skenario data mencakup batch aktif dan mendekati kedaluwarsa, batch yang diblokir, penerimaan batch berlapis, penjualan dari beberapa lapisan penerimaan, retur konsumen, stock opname, pemusnahan, dan retur kepada PBF.

### 9.2 Menjalankan Seed

Pastikan migration sudah berhasil dijalankan. Seed runner mensyaratkan migration berikut sudah tercatat pada `schema_migrations`:

- `001_init`
- `002_add_email_to_users`

Jalankan dari direktori utama repository:

```bash
go run ./cmd/seed
```

Seed runner akan:

1. Membaca konfigurasi `.env`.
2. Memeriksa bahwa migration `001_init` dan `002_add_email_to_users` sudah diterapkan.
3. Membuat tabel teknis `seed_runs` jika belum tersedia.
4. Memeriksa apakah database masih menggunakan seed lama tunggal `001_demo_data`.
5. Membaca seluruh file `seed/*.sql` dan mengurutkannya berdasarkan nama.
6. Menjalankan setiap file seed dalam transaksi tersendiri.
7. Mencatat file yang berhasil diterapkan pada `seed_runs`.
8. Melewati file yang sudah tercatat ketika perintah dijalankan kembali.

Migration `003_add_sessions.sql` nantinya tidak menjadi prasyarat seed selama file seed tidak membutuhkan tabel `sessions`. Session dibuat ketika pengguna melakukan login, bukan sebagai data dummy persediaan.

Jika salah satu file gagal, transaksi untuk file tersebut dibatalkan. File yang gagal dapat dicoba kembali setelah penyebabnya diperbaiki. File seed sebelumnya yang sudah berhasil tetap tercatat.

### 9.3 Akun Demo

Seed menyediakan tiga akun untuk pengujian hak akses.

| Role | Username | Email | Password |
|---|---|---|---|
| OWNER | `owner` | `owner@example.test` | `Apotek123!` |
| KASIR | `kasir` | `kasir@example.test` | `Apotek123!` |
| GUDANG | `gudang` | `gudang@example.test` | `Apotek123!` |

Password disimpan dalam bentuk bcrypt hash pada database. Akun tersebut hanya untuk pengembangan dan pengujian lokal. Email dengan domain `.test` bukan alamat email untuk penggunaan nyata.

### 9.4 Menambahkan atau Mengubah Data Seed

File seed yang sudah berhasil diterapkan tercatat di `seed_runs`. Mengubah isi file yang sudah diterapkan tidak membuat perubahan otomatis pada database karena runner akan melewati file tersebut.

Untuk menambah data dummy, buat file baru dengan nomor berikutnya. Contoh:

```text
seed/008_additional_demo_data.sql
```

Kemudian jalankan kembali:

```bash
go run ./cmd/seed
```

Seed runner akan melewati file lama dan menjalankan file baru yang belum tercatat. Jangan mengubah file seed yang sudah dibagikan dan diterapkan anggota lain dengan harapan perubahan tersebut otomatis masuk ke database mereka.

### 9.5 Reset Data Seed

Untuk mengulang seluruh data dari kondisi kosong, reset database pengembangan lokal menggunakan prosedur pada bagian 8.5, kemudian jalankan migration dan seed kembali.

Jika database pernah menggunakan seed versi lama tunggal (`001_demo_data`), jangan langsung menjalankan seed terpisah pada database yang sama. Reset hanya database pengembangan lokal terlebih dahulu untuk menghindari duplikasi data.

## 10. Menjalankan Aplikasi

Setelah database dibuat, migration diterapkan, dan seed berhasil dijalankan, mulai server dari direktori utama repository:

```bash
go run ./cmd/server
```

Port aplikasi mengikuti nilai `APP_PORT` pada file `.env`. Dengan konfigurasi contoh, alamat lokalnya adalah:

```text
http://localhost:8080
```

### 10.1 Endpoint Pemeriksaan

Router menyediakan endpoint `GET /health` untuk memeriksa koneksi database serta route `GET` halaman frontend berikut: `/dashboard`, `/kasir`, `/penerimaan`, `/cek-stok`, `/retur`, `/stock-opname`, `/kedaluwarsa`, `/laporan`, `/data-master`, `/manajemen-pengguna`, dan `/struk-penjualan`.

Route halaman tersebut saat ini masih menggunakan identitas demo dan belum dipasangi middleware session/role. Route final ini disediakan agar tampilan dapat diuji tanpa prefix `__preview`; jangan menganggapnya sebagai perlindungan akses untuk aplikasi operasional. Lihat `docs/BACKEND_LEAD_HANDOFF.md` untuk status dan tahapan integrasi autentikasi.

Buka alamat:

```text
http://localhost:8080/health
```

Jika server dan database berjalan dengan baik, response yang diharapkan:

```text
Server and database are running
```

Endpoint tersebut hanya digunakan untuk pemeriksaan kondisi server dan koneksi database, bukan sebagai endpoint autentikasi.

### 10.2 Pemeriksaan Build dan Test

Untuk memeriksa apakah seluruh package dapat dikompilasi, jalankan:

```bash
go build ./...
```

Untuk menjalankan pengujian yang tersedia:

```bash
go test ./...
```

### 10.3 Urutan Persiapan Awal Anggota

Untuk anggota yang baru melakukan clone dan menggunakan database lokal yang masih kosong, urutannya adalah:

1. Nyalakan MySQL/MariaDB.
2. Buat database kosong sesuai bagian 7.5.
3. Salin `.env.example` menjadi `.env`, lalu ganti seluruh nilai placeholder sesuai konfigurasi lokal.
4. Unduh dependency.
5. Jalankan migration.
6. Jalankan seed.
7. Jalankan server.

Perintahnya:

```bash
go mod download
go run ./cmd/migrate
go run ./cmd/seed
go run ./cmd/server
```

Setiap anggota menggunakan database lokal masing-masing dengan struktur dan data dummy yang sama.

## 11. Struktur Folder Proyek

Struktur proyek dibedakan menjadi struktur yang benar-benar tersedia pada ZIP saat ini dan gambaran struktur pengembangan berikutnya. File yang masih berupa placeholder atau belum tersedia ditandai agar tidak disalahartikan sebagai fitur yang sudah selesai.

### 11.1 Struktur Repository pada ZIP Saat Ini

```text
Apotek-Sumber-Waras/
│
├── cmd/
│   ├── server/
│   │   └── main.go
│   ├── migrate/
│   │   └── main.go
│   └── seed/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   └── app.go
│   ├── auth/
│   │   ├── handler.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── cashier/
│   │   ├── handler.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── db.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── role.go
│   ├── model/
│   │   └── user.go
│   ├── owner/
│   │   ├── handler.go
│   │   ├── repository.go
│   │   └── service.go
│   ├── router/
│   │   └── router.go
│   ├── view/
│   │   └── templates.go
│   └── warehouse/
│       ├── handler.go
│       ├── repository.go
│       └── service.go
│
├── migrations/
│   ├── 001_init.sql
│   ├── 002_add_email_to_users.sql
│   └── 003_add_sessions.sql
│
├── seed/
│   ├── 001_users.sql
│   ├── 002_master_data.sql
│   ├── 003_purchases_and_batches.sql
│   ├── 004_sales.sql
│   ├── 005_returns.sql
│   ├── 006_stock_opname.sql
│   └── 007_batch_disposals.sql
│
├── docs/                       # Dokumentasi teknis dan mapping frontend
├── tests/                      # Tempat pengujian
│
├── web/
│   ├── templates/
│   │   ├── layouts/
│   │   └── partials/
│   └── static/
│       ├── css/
│       ├── img/
│       └── js/
│
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

File `.env` bersifat lokal dan tidak disertakan dalam ZIP distribusi. Setiap anggota membuat file tersebut dari `.env.example` dan tidak boleh meng-commit kredensialnya.

Sebagian file pada folder fitur masih berupa placeholder. Keberadaan file tersebut menunjukkan tempat modul akan dikembangkan, bukan berarti login, dashboard, penjualan, penerimaan, atau fitur lain sudah selesai.

| File/Folder | Fungsi Saat Ini atau Target | Penanggung Jawab Utama |
|---|---|---|
| `cmd/server/main.go` | Titik awal server; membaca konfigurasi, membuka koneksi database, memanggil `app.New`, dan menjalankan HTTP server. | Backend Lead |
| `cmd/migrate/main.go` | Menjalankan seluruh migration SQL yang belum tercatat. | Backend Lead |
| `cmd/seed/main.go` | Menjalankan file seed yang belum tercatat. | Backend Lead |
| `internal/app/app.go` | Menginisialisasi renderer dan router melalui `app.New`; menjadi titik awal dependency wiring modul berikutnya. | Backend Lead |
| `internal/config/` | Membaca konfigurasi aplikasi dari `.env` atau environment variable. | Backend Lead |
| `internal/database/` | Membuat dan mengelola connection pool database. | Backend Lead |
| `internal/model/` | Menyimpan struct data dan error bersama. | Backend Lead dan Backend Developer |
| `internal/middleware/` | Fondasi pemeriksaan session dan pembatasan akses berdasarkan role. | Backend Lead |
| `internal/router/` | Mendaftarkan URL halaman permanen dan endpoint `/health`. Halaman masih memakai identitas demo sementara, belum memakai middleware session/role. | Backend Lead |
| `internal/auth/` | Handler halaman login sudah ada, tetapi POST login belum terhubung; repository dan service masih placeholder. | Backend Owner |
| `internal/owner/` | Target modul manajemen pengguna, data master, dan dashboard. File saat ini masih placeholder. | Backend Owner |
| `internal/cashier/` | Target modul penjualan dan retur konsumen. File saat ini masih placeholder. | Backend Kasir |
| `internal/warehouse/` | Target modul penerimaan, batch, persediaan, dan stock opname. File saat ini masih placeholder. | Backend Gudang |
| `internal/view/` | Memuat template HTML dan merender halaman menggunakan `PageData`. | Integrasi Backend dan Frontend |
| `web/templates/` | Tempat template HTML yang akan dibuat oleh Frontend. | Frontend 1 dan Frontend 2 |
| `web/static/` | Tempat CSS, JavaScript, dan gambar antarmuka. | Frontend 1 dan Frontend 2 |
| `migrations/` | Menyimpan perubahan struktur database. | Backend Lead dan pengusul fitur |
| `seed/` | Menyimpan data dummy standar untuk pengujian. | Backend Lead dan tim |

**Catatan integrasi saat ini:** `cmd/server/main.go` sudah memanggil `app.New(db)`. Package `internal/app` membuat renderer dan router, sedangkan satu koneksi database diteruskan ke router. Route halaman saat ini sudah menggunakan URL permanen seperti `/dashboard` dan `/cek-stok`, tetapi handler halaman masih memakai identitas demo sementara dan belum dilindungi middleware autentikasi. Rincian pembagian kerja ada pada `docs/BACKEND_LEAD_HANDOFF.md`.

### 11.2 Target Struktur Pengembangan

Berikut gambaran struktur setelah fitur backend dan frontend dikembangkan:

```text
Apotek-Sumber-Waras/
│
├── cmd/
│   ├── server/
│   │   └── main.go
│   ├── migrate/
│   │   └── main.go
│   └── seed/
│       └── main.go
│
├── internal/
│   ├── app/
│   │   └── app.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── db.go
│   ├── model/
│   │   ├── user.go
│   │   ├── medicine.go
│   │   ├── batch.go
│   │   └── sale.go
│   ├── middleware/
│   │   ├── auth.go
│   │   └── role.go
│   ├── router/
│   │   └── router.go
│   ├── auth/                  # Backend Owner
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── owner/                 # Backend Owner
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── cashier/               # Backend Kasir
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   ├── warehouse/             # Backend Gudang
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   └── view/
│       └── templates.go
│
├── web/
│   ├── templates/
│   │   ├── layouts/
│   │   │   └── base.html
│   │   ├── partials/
│   │   │   ├── sidebar.html
│   │   │   └── header.html
│   │   ├── login.html
│   │   ├── dashboard.html
│   │   ├── data_master.html
│   │   ├── penerimaan.html
│   │   ├── kasir.html
│   │   ├── retur.html
│   │   ├── cek_stok.html
│   │   ├── stock_opname.html
│   │   ├── kedaluwarsa.html
│   │   ├── laporan.html
│   │   └── manajemen_pengguna.html
│   └── static/
│       ├── css/
│       ├── js/
│       └── img/
│
├── migrations/
│   ├── 001_init.sql
│   ├── 002_add_email_to_users.sql
│   └── 003_add_sessions.sql 
│
├── seed/
│   ├── 001_users.sql
│   ├── 002_master_data.sql
│   ├── 003_purchases_and_batches.sql
│   ├── 004_sales.sql
│   ├── 005_returns.sql
│   ├── 006_stock_opname.sql
│   └── 007_batch_disposals.sql
│
├── docs/
│   ├── ERD.png
│   ├── DATABASE.md
│   └── DEVELOPMENT.md
│
├── tests/
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

Struktur di atas merupakan target, bukan daftar file yang harus dibuat kosong sejak awal. Folder atau file tambahan dibuat ketika fitur terkait mulai dikerjakan.

### 11.3 Pola Arsitektur Backend

Pengembangan backend mengikuti pemisahan tanggung jawab berdasarkan lapisan:

```text
HTTP Request
     |
     v
  Handler  ------ urusan HTTP request/response
     |
     v
  Service  ------ aturan bisnis dan validasi
     |
     v
 Repository ------ query dan interaksi database
     |
     v
   MySQL
```

`Model` digunakan bersama oleh repository, service, dan handler sebagai struktur data aplikasi.

| Lapisan | Tanggung Jawab |
|---|---|
| Model | Menyimpan struct data dan error yang digunakan bersama. Tidak berisi query SQL atau business logic. |
| Handler | Menerima HTTP request, membaca input, memanggil service, lalu mengirim response, redirect, atau merender halaman. Tidak menjalankan query SQL. |
| Service | Menjalankan validasi dan aturan bisnis. Tidak bergantung pada `http.Request` dan tidak menjalankan query SQL langsung. |
| Repository | Menjadi tempat query SQL dan interaksi database. Mengembalikan struct/model, bukan objek `*sql.Rows` ke lapisan atas. |
| Middleware | Membungkus handler untuk menjalankan pemeriksaan umum sebelum request diteruskan, seperti autentikasi dan role. |
| Router | Mendaftarkan URL, HTTP method, handler, dan middleware. |

Proyek menggunakan pengelompokan modul berdasarkan fitur (*feature-based*). Dengan demikian, handler, service, dan repository untuk fitur autentikasi berada pada `internal/auth/`; fitur Owner pada `internal/owner/`; Kasir pada `internal/cashier/`; dan Gudang pada `internal/warehouse/`.

Contoh struktur modul:

```text
internal/warehouse/
├── handler.go
├── service.go
└── repository.go
```

Pola tersebut merupakan pedoman, bukan kewajiban mutlak. File tambahan dapat dibuat apabila suatu logika cukup kompleks untuk dipisahkan, misalnya `fefo.go` untuk aturan FEFO. Jangan membuat file kosong hanya untuk mengikuti pola.

### 11.4 Fungsi `internal/app`

`internal/app/app.go` dipertahankan sebagai tempat inisialisasi dan penghubung dependency aplikasi (*application initialization/dependency wiring*).

Target tanggung jawabnya adalah menghubungkan komponen seperti:

- Koneksi database.
- Repository.
- Service.
- Handler.
- Renderer/view.
- Middleware.
- Router.

Dengan pemisahan tersebut, `cmd/server/main.go` nantinya cukup membaca konfigurasi, membuat aplikasi, mengatur HTTP server, dan menjalankannya. Pada implementasi saat ini, `app.go` sudah membuat renderer dan router. Dependency repository, service, handler bisnis, dan middleware akan ditambahkan ke wiring ini ketika modul masing-masing selesai.

### 11.5 Perbedaan `internal/view` dan `web`

- `web/templates/` berisi file tampilan HTML yang dibuat dan dikelola Frontend.
- `web/static/` berisi CSS, JavaScript, dan gambar.
- `internal/view/` berisi kode Go untuk memuat dan merender template HTML ketika diperlukan dalam integrasi backend dan frontend.

Dengan demikian, Frontend mengerjakan isi dan desain halaman, sedangkan kode Go pada `internal/view/` menghubungkan template dengan handler. Penempatan dan cara pemuatan template perlu disepakati sebelum integrasi.

## 12. Pembagian Tanggung Jawab dan Pedoman Pengembangan

### 12.1 Pembagian Tanggung Jawab

| Anggota/Bagian | Tanggung Jawab |
|---|---|
| **Backend Lead** | Menyiapkan fondasi aplikasi, konfigurasi, koneksi database, model bersama awal, middleware, migration, seed, standar integrasi, dan review perubahan skema. |
| **Backend Owner** | Mengembangkan autentikasi login/logout, manajemen pengguna, data master, dan agregasi dashboard. |
| **Backend Kasir** | Mengembangkan penjualan, FEFO, alokasi stok, pembayaran, retur konsumen, dan perhitungan keuntungan. |
| **Backend Gudang** | Mengembangkan penerimaan, batch, persediaan, stock opname, pemblokiran batch, pemusnahan, dan retur ke PBF. |
| **Frontend 1** | Mengembangkan login, layout bersama, data master, penerimaan, persediaan, stock opname, kedaluwarsa, dan manajemen pengguna. |
| **Frontend 2** | Mengembangkan dashboard, halaman kasir, retur konsumen, dan laporan. |

Pembagian tersebut merupakan tanggung jawab utama masing-masing anggota. Integrasi antarmodul tetap dilakukan bersama.

### 12.2 Kepemilikan Folder dan File

- Backend Lead menjaga fondasi bersama pada `cmd/`, `internal/app/`, `internal/config/`, `internal/database/`, `internal/model/`, `internal/middleware/`, dan `internal/router/`.
- Backend Owner membuat dan mengelola `internal/auth/` serta `internal/owner/`.
- Backend Kasir membuat dan mengelola `internal/cashier/`.
- Backend Gudang membuat dan mengelola `internal/warehouse/`.
- Frontend 1 dan Frontend 2 membuat serta mengelola halaman dan aset pada `web/` sesuai pembagian fitur.
- `internal/view/` digunakan ketika proses render HTML perlu diintegrasikan.
- `migrations/` dikoordinasikan Backend Lead. Developer fitur dapat mengusulkan perubahan skema melalui migration baru.
- `seed/` menjadi data dummy standar tim. Penambahan data dummy dilakukan melalui file baru dengan nomor berurutan.
- `docs/` berisi dokumentasi teknis yang disusun dan diperbarui bersama sesuai kebutuhan.

### 12.3 Dependency dan Integrasi Backend

Alur dependency aplikasi mengikuti prinsip dependency melalui struct, bukan variabel global.

Urutan umum integrasi:

```text
Database
   |
   v
Repository
   |
   v
Service
   |
   v
Handler
   |
   v
Router
```

Aturan integrasi:

- Gunakan satu objek koneksi database (`*sql.DB`) yang dibuat saat aplikasi dimulai dan diberikan kepada modul backend.
- Jangan membuat koneksi database baru di setiap handler atau setiap request.
- Query database diletakkan pada repository.
- Aturan bisnis diletakkan pada service.
- Handler tidak menjalankan query SQL secara langsung.
- Model digunakan bersama oleh lapisan yang membutuhkan.
- Perubahan skema database harus dikoordinasikan dan ditinjau sebelum digabungkan.
- Validasi stok harus dilakukan di dalam transaksi database untuk mencegah ketidaksesuaian data.
- Modul yang berbagi proses persediaan harus menggunakan aturan dan mekanisme stok yang konsisten.

### 12.4 Integrasi Autentikasi dan Role

Pengelolaan autentikasi dibagi antara Backend Lead, Backend Owner, dan Frontend 1.

**Backend Lead:**

- Menyediakan middleware untuk memeriksa session pengguna.
- Menyediakan middleware untuk membatasi akses berdasarkan role.
- Menyediakan fungsi untuk mengambil informasi pengguna dari request context.
- Mengintegrasikan middleware dengan router utama.
- Mengkoordinasikan migration tabel `sessions` bersama Backend Owner.

**Backend Owner:**

- Membuat fitur login dan logout.
- Menerima satu nilai input `identifier` yang dapat berisi username atau email, serta password.
- Memeriksa akun berdasarkan username atau email pada tabel `users`.
- Melakukan verifikasi password menggunakan bcrypt.
- Memeriksa status akun aktif.
- Menghasilkan token session acak menggunakan `crypto/rand`.
- Menyimpan, memeriksa, dan menghapus session melalui repository database.
- Mengimplementasikan kontrak `SessionValidator` yang disediakan middleware.
- Mengintegrasikan proses autentikasi dengan middleware yang disediakan Backend Lead.
- Mengembangkan manajemen akun pengguna sesuai hak akses OWNER.

**Frontend 1:**

- Membuat halaman login.
- Membuat satu field teks bernama `identifier` untuk username atau email, serta field password.
- Mengintegrasikan form dengan endpoint login yang disepakati.

Sistem tidak menyediakan registrasi publik. Akun pengguna dikelola oleh OWNER.

#### Rancangan Session

Autentikasi menggunakan pendekatan **server-side session**. Token session dibuat secara acak oleh backend dan disimpan di database. Browser hanya menyimpan token tersebut dalam cookie untuk dikirim kembali pada request berikutnya.

Dengan pendekatan ini, aplikasi **tidak memerlukan `SESSION_KEY`**. Cookie hanya membawa token session; pemeriksaan validitas session dilakukan oleh backend melalui repository database.

Ketentuan session yang menjadi target implementasi:

- Satu akun hanya memiliki satu session aktif.
- Login dari perangkat baru menggantikan session sebelumnya.
- Session berakhir ketika pengguna tidak aktif selama 30 menit.
- Masa berlaku maksimum session adalah 8 jam.
- Logout menghapus session dari database dan menghapus cookie pada browser.
- Cookie menggunakan `HttpOnly`, `SameSite=Lax`, dan `Path=/`.
- Atribut `Secure` diaktifkan ketika aplikasi berjalan melalui HTTPS.

**Status implementasi pada ZIP saat ini:**

- `internal/middleware/auth.go` sudah menyediakan nama cookie `session`, kontrak `SessionValidator`, fungsi `SetSessionCookie`, `ClearSessionCookie`, `RequireLogin`, dan `CurrentUser`.
- `RequireLogin` sudah mengambil cookie, memanggil `ValidateAndTouch`, memeriksa status aktif pengguna, dan meneruskan user melalui request context.
- `internal/middleware/role.go` sudah menyediakan middleware pembatasan role `OWNER`, `KASIR`, dan `GUDANG`.
- `internal/model/user.go` sudah menyediakan struct `User` dan error `ErrInvalidSession`.
- Implementasi repository session, proses login/logout, dan integrasi autentikasi ke router belum tersedia.
- Migration `003_add_sessions.sql` sudah tersedia dan membuat tabel `sessions`, tetapi repository session dan integrasinya belum tersedia.

Ketentuan satu session per akun, idle timeout 30 menit, dan batas maksimum 8 jam baru dianggap berjalan setelah repository, migration, handler/service autentikasi, dan integrasi middleware selesai dibuat serta diuji.

### 12.5 Integrasi Frontend dan Backend

Sebelum menghubungkan halaman dengan backend, sepakati:

- URL dan HTTP method setiap fitur.
- Nama input form.
- Format data yang dikirim.
- Format response atau pesan hasil proses.
- Aturan validasi.
- Kebutuhan role pengguna.

Frontend dan backend harus menggunakan kesepakatan yang sama agar proses integrasi tidak menimbulkan perbedaan nama field maupun format data.

### 12.6 Aturan Git dan GitHub

Untuk menjaga kestabilan kode:

1. Branch `main` digunakan sebagai versi utama yang stabil.
2. Setiap anggota mengerjakan fitur melalui branch masing-masing.
3. Perubahan dikirim melalui Pull Request.
4. Perubahan pada branch `main` harus melalui proses review.
5. Hindari melakukan push langsung ke `main`.
6. Jangan mengubah file fondasi atau file milik anggota lain tanpa koordinasi.
7. Perubahan pada `go.mod`, `go.sum`, migration, seed, router, konfigurasi, dan koneksi database perlu diperiksa sebelum digabungkan.
8. Jangan commit file `.env`, kredensial, password, atau data rahasia lainnya.

Contoh pembuatan branch:

```bash
git checkout -b feature/nama-fitur
```

Contoh mengirim perubahan:

```bash
git add nama-file-atau-folder
git commit -m "feat: menambahkan fitur penerimaan barang"
git push origin feature/nama-fitur
```

Setelah itu, buat Pull Request ke branch `main`. Aturan review dan perlindungan branch dapat diatur melalui pengaturan repository GitHub. Proyek ini tidak mewajibkan folder atau file `.github`.

## 13. Dokumentasi Tambahan

Dokumentasi teknis proyek dapat disimpan pada folder `docs/`, meliputi:

- ERD dan relasi antartabel (`ERD.png`).
- Struktur database dan penjelasan tabel (`DATABASE.md`).
- Panduan struktur kode, pembagian modul, dan proses integrasi (`DEVELOPMENT.md`).
- Rancangan antarmuka dan alur proses apabila diperlukan.

Folder `docs/` berisi mapping frontend dan panduan kerja Backend Lead. Setiap perubahan penting terhadap database, kontrak endpoint, atau alur bisnis sebaiknya diperbarui pada dokumentasi terkait.

Folder `tests/` disediakan untuk pengujian unit dan integrasi. Pengujian ditambahkan seiring dengan implementasi fitur.

## 14. Batasan Sistem

Sistem dikembangkan sesuai kebutuhan pengelolaan stok Apotek Sumber Waras. Hal-hal berikut tidak termasuk dalam lingkup pengembangan:

- Integrasi langsung dengan sistem BPOM, PBF, atau pihak eksternal lainnya.
- Payment gateway untuk pembayaran nontunai.
- Manajemen data pelanggan.
- Resep elektronik dan integrasi dengan dokter.
- Otomatisasi pemesanan atau pengadaan barang ke PBF.
- Sistem akuntansi penuh.

Pengembangan difokuskan pada pencatatan persediaan berbasis batch, transaksi penjualan, retur, stock opname, dan pelaporan yang dapat ditelusuri.