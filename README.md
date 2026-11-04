# Sistem Stok Apotek Berbasis Batch — Apotek Sumber Waras

Aplikasi web untuk membantu pengelolaan persediaan obat di Apotek Sumber Waras dengan pencatatan berdasarkan batch fisik, lapisan penerimaan, tanggal kedaluwarsa, dan riwayat pergerakan stok.

**Status:** Dalam pengembangan  
**Jenis Proyek:** Proyek Kelompok — Pemrograman Web

---

## Daftar Isi

1. [Tentang Proyek](#1-tentang-proyek)
2. [Fitur Utama](#2-fitur-utama)
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

Proyek ini bertujuan membangun sistem informasi berbasis web yang membantu proses pengelolaan obat, mulai dari penerimaan barang dari Pedagang Besar Farmasi (PBF), penjualan, pengembalian obat, pemeriksaan stok, hingga penyusunan laporan.

Sistem menggunakan pendekatan **batch-based inventory**, yaitu stok obat ditelusuri berdasarkan identitas batch fisik dan lapisan penerimaannya. Dengan demikian, obat yang sama dapat memiliki beberapa catatan penerimaan dengan nomor batch, tanggal kedaluwarsa, jumlah, dan harga beli yang berbeda.

## 2. Fitur Utama

### 2.1 Data Master

- Mengelola data obat.
- Mengelola satuan obat dan konversi ke satuan dasar.
- Mengatur harga jual berdasarkan satuan.
- Mengelola data PBF atau pemasok.

### 2.2 Penerimaan Barang

- Mencatat penerimaan obat berdasarkan faktur PBF.
- Mencatat nomor batch dan tanggal kedaluwarsa.
- Mencatat kuantitas dan harga beli.
- Memisahkan pencatatan batch fisik dan lapisan penerimaan.
- Menambah stok berdasarkan jumlah yang diterima.

### 2.3 Penjualan

- Menampilkan obat yang dapat dijual.
- Mengonversi jumlah penjualan ke satuan dasar.
- Mengambil stok menggunakan prinsip FEFO (*First Expired, First Out*).
- Mencatat alokasi stok berdasarkan lapisan penerimaan.
- Mencatat pembayaran tunai maupun nontunai.
- Menghitung total pembayaran dan kembalian.
- Mencetak struk transaksi.

### 2.4 Retur Konsumen

- Mencatat pengembalian obat dengan mengacu pada transaksi penjualan asli.
- Menelusuri batch dan lapisan penerimaan asal obat.
- Mencatat jumlah, alasan, dan kondisi obat yang dikembalikan.
- Memisahkan obat yang menunggu pemeriksaan, layak jual, dan tidak layak jual.
- Mengembalikan stok ke persediaan yang dapat dijual hanya setelah obat dinyatakan layak jual.

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
| **OWNER** | Mengelola pengguna dan data master, memantau dashboard, serta melihat laporan. |
| **KASIR** | Melakukan transaksi penjualan dan mencatat retur konsumen. |
| **GUDANG** | Mengelola penerimaan barang, memantau persediaan, melakukan stock opname, serta mencatat pemusnahan dan retur ke PBF. |

Setiap pengguna harus melakukan login. Tidak tersedia registrasi akun secara publik. Pengelolaan akun dilakukan oleh OWNER.

Tidak ada role SUPERADMIN terpisah dalam rancangan database. Akun OWNER digunakan untuk pengelolaan sistem sesuai hak aksesnya.

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

   Setiap transaksi yang memengaruhi stok harus diproses secara aman menggunakan transaksi database dan validasi ulang agar dua kasir tidak menggunakan stok yang sama secara bersamaan.

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

### 5.1 Tabel Teknis

Selain 14 tabel inti, terdapat dua tabel teknis yang dibuat untuk membantu pencatatan proses database:

- `schema_migrations`: mencatat migration yang sudah diterapkan.
- `seed_runs`: mencatat file seed yang sudah berhasil dijalankan.

Kedua tabel tersebut bukan bagian dari 14 tabel bisnis pada ERD.

Tabel `sessions` belum tersedia pada migration awal di repository saat ini. Tabel tersebut akan ditambahkan melalui migration lanjutan ketika integrasi autentikasi berbasis database selesai.

### 5.2 Prinsip Struktur Database

- Stok tidak disimpan langsung pada tabel `medicines`; jumlah stok diperoleh dari lapisan penerimaan pada `batch_receipts`.
- `batches` menyimpan identitas batch fisik. `batch_receipts` menyimpan penerimaan batch pada faktur tertentu.
- Satu batch fisik dapat memiliki beberapa lapisan penerimaan dengan jumlah dan harga beli yang berbeda.
- `sale_items` menyimpan rincian pembelian konsumen, sedangkan `sale_item_batches` mencatat alokasi stok dari lapisan penerimaan.
- `returns` selalu merujuk pada alokasi batch dari transaksi penjualan asal.
- `stock_opname_details` mencatat penyesuaian pada setiap lapisan penerimaan yang terdampak.
- `batch_disposals` mencatat pemusnahan atau retur ke PBF, terpisah dari retur konsumen.

Perubahan struktur tabel harus dilakukan melalui migration dan dikoordinasikan dengan Backend Lead. Developer fitur tidak membuat ulang tabel yang sudah tersedia.

## 6. Teknologi dan Library

### 6.1 Teknologi Utama

| Teknologi | Kegunaan |
|---|---|
| Go (Golang) | Backend dan server aplikasi |
| HTML | Struktur halaman web |
| CSS | Tampilan antarmuka |
| JavaScript | Interaksi pada halaman web |
| MySQL/MariaDB | Penyimpanan data relasional |
| Git dan GitHub | Version control dan kolaborasi tim |

### 6.2 Library Eksternal Go

Proyek menggunakan tiga modul eksternal utama.

| Modul | Package yang Digunakan | Kegunaan |
|---|---|---|
| `github.com/go-sql-driver/mysql` | `github.com/go-sql-driver/mysql` | Driver MySQL untuk `database/sql`. |
| `github.com/joho/godotenv` | `github.com/joho/godotenv` | Membaca konfigurasi dari file `.env`. |
| `golang.org/x/crypto` | `golang.org/x/crypto/bcrypt` | Hashing dan verifikasi password menggunakan bcrypt. |

**Catatan:** bcrypt merupakan package di dalam modul `golang.org/x/crypto`, bukan modul terpisah. Package ini digunakan pada implementasi autentikasi. Password tidak disimpan dalam bentuk teks biasa.

### 6.3 Package Bawaan Go

Beberapa package bawaan Go yang digunakan:

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

- Go versi yang sesuai dengan `go.mod` (repository saat ini mencantumkan Go 1.27.0).
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

Perintah tersebut mengunduh dependency yang sudah tercantum pada `go.mod`.

`go mod tidy` tidak perlu dijalankan setiap kali membuat atau mengubah satu file `.go`. Jalankan ketika terdapat perubahan import atau dependency, kemudian periksa perubahan pada `go.mod` dan `go.sum` sebelum melakukan commit.

Perlu diperhatikan bahwa `go mod tidy` dapat menghapus dependency yang belum digunakan oleh kode. Package bcrypt akan benar-benar digunakan ketika implementasi login dibuat oleh Backend Owner. Jika belum ada import bcrypt, dependency tersebut dapat dikeluarkan oleh `go mod tidy` dan perlu ditambahkan kembali ketika implementasinya dimulai.

Setiap perubahan dependency harus dikoordinasikan agar seluruh anggota menggunakan versi yang konsisten.

### 7.4 Konfigurasi Environment

Salin file `.env.example` menjadi `.env`.

PowerShell:

```powershell
Copy-Item .env.example .env
```

Linux/macOS:

```bash
cp .env.example .env
```

Sesuaikan konfigurasi database pada `.env` dengan perangkat masing-masing.

```env
APP_PORT=8080

DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=apotek_sumber_waras

SESSION_KEY=isi-dengan-secret-acak-minimal-32-byte
```

Catatan:

- Pastikan MySQL/MariaDB sudah berjalan.
- Sesuaikan `DB_USER`, `DB_PASSWORD`, dan `DB_PORT` dengan konfigurasi lokal.
- `SESSION_KEY` masih divalidasi oleh konfigurasi aplikasi yang tersedia pada repository saat ini. Penggunaan akhirnya perlu diselaraskan ketika integrasi session database diselesaikan.
- Jangan menggunakan secret contoh untuk lingkungan produksi.
- File `.env` berisi konfigurasi lokal dan rahasia sehingga tidak boleh diunggah ke GitHub.
- Gunakan `.env.example` sebagai acuan konfigurasi bersama.
- Pastikan `.env` benar-benar tidak ter-commit. Jika sebelumnya sudah masuk pelacakan Git, keluarkan dari index menggunakan `git rm --cached .env`.

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

Migration digunakan untuk membuat struktur database, termasuk tabel, kolom, primary key, foreign key, unique key, dan indeks.

File migration awal yang tersedia saat ini:

```text
migrations/
└── 001_init.sql
```

Jalankan dari direktori utama repository:

```bash
go run ./cmd/migrate
```

Migration runner saat ini akan:

1. Membaca konfigurasi `.env`.
2. Menghubungkan aplikasi ke MySQL/MariaDB.
3. Membuat tabel teknis `schema_migrations` jika belum tersedia.
4. Memeriksa apakah migration `001_init` sudah pernah diterapkan.
5. Menjalankan `migrations/001_init.sql` jika belum tercatat.
6. Mencatat migration yang berhasil diterapkan.

Migration awal membuat 14 tabel inti berdasarkan ERD.

Jika migration `001_init` sudah tercatat, perintah berikutnya akan melewatinya dan tidak membuat ulang tabel.

**Batasan migration saat ini:**

Runner yang tersedia masih menjalankan satu file yang ditentukan secara langsung, yaitu `migrations/001_init.sql`. Runner belum membaca seluruh file migration secara otomatis.

Apabila terdapat perubahan struktur database, developer mengusulkan migration baru dan berkoordinasi dengan Backend Lead. Runner perlu dikembangkan agar dapat menjalankan file migration berikutnya secara berurutan, misalnya `002_sessions.sql`.

Jangan menganggap menambahkan file SQL baru saja sudah cukup karena runner saat ini belum otomatis menerapkannya.

### 8.1 Rencana Migration Session

Untuk mendukung autentikasi berbasis session database, sistem nantinya memerlukan tabel `sessions`.

Rencananya tabel ini akan ditambahkan melalui:

```text
migrations/
├── 001_init.sql
└── 002_sessions.sql
```

Tabel `sessions` merupakan tabel teknis, bukan bagian dari 14 tabel inti ERD.

Migration tersebut baru dapat digunakan setelah migration runner mendukung lebih dari satu file SQL dan Backend Owner menyelesaikan integrasi session repository.

### 8.2 Mengulang Database dari Awal

Saat ini belum tersedia perintah `migrate fresh` seperti pada Laravel.

Untuk mengulang database pengembangan lokal dari kondisi kosong:

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

**Peringatan:** `DROP DATABASE` menghapus seluruh tabel dan data di dalam database. Jangan menjalankan perintah ini pada database operasional atau database yang berisi data penting.

## 9. Seed Data Dummy

Seed digunakan untuk memasukkan data dummy ke tabel yang sudah dibuat oleh migration.

Seed tidak membuat tabel inti atau kolom database. Proses pembuatan struktur tetap menjadi tanggung jawab migration.

File seed dipisahkan berdasarkan kelompok data agar lebih mudah dibaca dan diperbarui.

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

### 9.1 Isi Data Seed

| File | Isi |
|---|---|
| `001_users.sql` | Akun OWNER, KASIR, dan GUDANG. |
| `002_master_data.sql` | PBF, data obat, dan satuan obat. |
| `003_purchases_and_batches.sql` | Faktur penerimaan, identitas batch, dan lapisan penerimaan. |
| `004_sales.sql` | Transaksi penjualan, rincian, dan alokasi batch. |
| `005_returns.sql` | Contoh retur konsumen dengan status kelayakan berbeda. |
| `006_stock_opname.sql` | Hasil stock opname dan detail penyesuaian. |
| `007_batch_disposals.sql` | Contoh pemusnahan dan retur ke PBF. |

Data dummy disiapkan agar fitur dapat dicoba dengan data awal yang seragam.

Contoh skenario data yang tersedia:

- Batch aktif dan batch mendekati kedaluwarsa.
- Batch yang diblokir.
- Satu batch fisik yang diterima melalui lebih dari satu lapisan penerimaan dengan harga beli berbeda.
- Transaksi penjualan yang mengambil stok dari beberapa lapisan penerimaan.
- Retur konsumen yang menunggu pemeriksaan dan yang sudah dinyatakan layak jual.
- Stock opname dengan selisih stok.
- Pemusnahan dan retur barang kepada PBF.

### 9.2 Menjalankan Seed

Pastikan migration sudah berhasil dijalankan. Kemudian, dari direktori utama repository, jalankan:

```bash
go run ./cmd/seed
```

Seed runner akan:

1. Membaca konfigurasi `.env`.
2. Memastikan migration `001_init` sudah diterapkan.
3. Membuat tabel teknis `seed_runs` jika belum tersedia.
4. Membaca file `seed/*.sql` berdasarkan urutan nama file.
5. Menjalankan setiap file seed dalam transaksi tersendiri.
6. Mencatat file yang berhasil diterapkan.
7. Melewati file yang sudah tercatat ketika perintah dijalankan kembali.

Jika salah satu file gagal, transaksi untuk file tersebut dibatalkan. File yang gagal dapat dicoba kembali setelah penyebabnya diperbaiki. File seed sebelumnya yang sudah berhasil tetap tercatat.

### 9.3 Akun Demo

Seed menyediakan tiga akun untuk pengujian hak akses.

| Role | Username | Password |
|---|---|---|
| OWNER | `owner` | `Apotek123!` |
| KASIR | `kasir` | `Apotek123!` |
| GUDANG | `gudang` | `Apotek123!` |

Password disimpan dalam bentuk bcrypt hash. Akun tersebut hanya untuk pengembangan dan pengujian lokal.

### 9.4 Menambahkan atau Mengubah Data Seed

File seed yang sudah berhasil diterapkan tercatat di `seed_runs`.

Mengubah isi file yang sudah diterapkan tidak membuat perubahan otomatis pada database karena runner akan melewati file tersebut.

Untuk menambah data dummy, buat file baru dengan nomor berikutnya. Contoh:

```text
seed/008_additional_demo_data.sql
```

Kemudian jalankan kembali:

```bash
go run ./cmd/seed
```

Seed runner akan melewati file lama dan menjalankan file baru yang belum tercatat.

Jangan mengubah file seed yang sudah dibagikan dan diterapkan anggota lain dengan harapan perubahan otomatis masuk.

### 9.5 Reset Data Seed

Untuk mengulang seluruh data dari kondisi kosong, reset database pengembangan lokal menggunakan prosedur pada bagian 8.2, kemudian jalankan migration dan seed kembali.

Jika database pernah menggunakan seed versi lama tunggal (`001_demo_data`), jangan langsung menjalankan seed terpisah pada database yang sama. Reset database pengembangan lokal terlebih dahulu untuk menghindari duplikasi data.

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

Router saat ini menyediakan endpoint `/health` untuk memeriksa apakah server dan database dapat dihubungi.

Buka:

```text
http://localhost:8080/health
```

Jika server dan database berjalan dengan baik, response yang diharapkan adalah:

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
3. Salin `.env.example` menjadi `.env`, lalu sesuaikan konfigurasi lokal.
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

Struktur proyek dibagi menjadi dua tahap: **struktur awal yang disiapkan Backend Lead** dan **struktur pengembangan yang dibuat oleh anggota sesuai tanggung jawabnya**.

Folder fitur tidak perlu dibuat kosong hanya untuk memenuhi contoh struktur.

### 11.1 Struktur Awal Repository

Berikut struktur awal yang menjadi acuan setelah Backend Lead menyelesaikan fondasi aplikasi.

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
│   ├── config/
│   │   └── config.go
│   │
│   ├── database/
│   │   └── db.go
│   │
│   ├── model/
│   │   └── user.go
│   │
│   ├── middleware/
│   │   ├── auth.go
│   │   └── role.go
│   │
│   └── router/
│       └── router.go
│
├── migrations/
│   └── 001_init.sql
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
├── .env                  # Lokal, tidak di-commit
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

Keterangan:

| File/Folder | Fungsi | Penanggung Jawab |
|---|---|---|
| `cmd/server/main.go` | Titik awal untuk menjalankan aplikasi web. | Backend Lead |
| `cmd/migrate/main.go` | Program untuk menerapkan migration database. | Backend Lead |
| `cmd/seed/main.go` | Program untuk memasukkan data dummy. | Backend Lead |
| `internal/config/` | Membaca konfigurasi aplikasi dari `.env`. | Backend Lead |
| `internal/database/` | Membuat dan mengelola connection pool database. | Backend Lead |
| `internal/model/` | Menyimpan struct data bersama yang digunakan oleh beberapa lapisan. | Backend Lead dan Backend Developer |
| `internal/middleware/` | Menyediakan pemeriksaan autentikasi dan pembatasan akses berdasarkan role. | Backend Lead |
| `internal/router/` | Mengatur route utama dan endpoint pemeriksaan `/health`. | Backend Lead |
| `migrations/` | Menyimpan struktur database dalam file SQL. | Backend Lead |
| `seed/` | Menyimpan data dummy standar untuk pengujian. | Backend Lead |

**Catatan:** Struktur ini menampilkan migration yang benar-benar tersedia saat ini. `002_sessions.sql` belum disertakan karena migration runner belum mendukung penerapan file kedua.

Backend Lead tidak perlu membuat folder fitur Owner, Kasir, Gudang, atau halaman frontend kosong. Folder tersebut dibuat oleh anggota yang bertanggung jawab ketika implementasi fiturnya dimulai.

### 11.2 Struktur Pengembangan Setelah Pekerjaan Dibagikan

Berikut gambaran struktur yang dapat terbentuk setelah Backend Owner, Backend Kasir, Backend Gudang, serta Frontend mulai mengerjakan fitur masing-masing.

File di bagian ini merupakan target pengembangan, bukan file yang harus dibuat kosong sejak awal.

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
│   ├── config/
│   │   └── config.go
│   │
│   ├── database/
│   │   └── db.go
│   │
│   ├── model/
│   │   ├── user.go
│   │   ├── medicine.go
│   │   ├── batch.go
│   │   └── sale.go
│   │
│   ├── middleware/
│   │   ├── auth.go
│   │   └── role.go
│   │
│   ├── router/
│   │   └── router.go
│   │
│   ├── auth/                       # Dibuat Backend Owner
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   │
│   ├── owner/                      # Dibuat Backend Owner
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   │
│   ├── cashier/                    # Dibuat Backend Kasir
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   │
│   ├── warehouse/                  # Dibuat Backend Gudang
│   │   ├── handler.go
│   │   ├── service.go
│   │   └── repository.go
│   │
│   └── view/                       # Dibuat saat integrasi render HTML
│       └── templates.go
│
├── web/                            # Dibuat Frontend 1 dan Frontend 2
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
│   │
│   └── static/
│       ├── css/
│       ├── js/
│       └── img/
│
├── migrations/
│   ├── 001_init.sql
│   └── 002_sessions.sql          # Setelah runner mendukung migration berikutnya
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
│
├── .env
├── .env.example
├── .gitignore
├── go.mod
├── go.sum
└── README.md
```

Folder `auth` dibuat oleh Backend Owner karena berisi proses login, logout, dan pengelolaan autentikasi yang terhubung ke database.

Folder `owner`, `cashier`, dan `warehouse` menggunakan pendekatan **feature-based**. Kode backend dikelompokkan berdasarkan fitur atau tanggung jawab, bukan berdasarkan folder global `handler`, `service`, dan `repository`.

### 11.3 Pola Penulisan Kode Backend

Setiap modul backend disarankan memisahkan tanggung jawab menggunakan tiga lapisan:

| Lapisan | Tanggung Jawab |
|---|---|
| Handler | Menerima HTTP request, memanggil service, dan mengirim response atau merender halaman. |
| Service | Menjalankan aturan bisnis, validasi, dan koordinasi transaksi. |
| Repository | Menjalankan query dan interaksi dengan database. |

Contoh struktur modul Gudang:

```text
internal/warehouse/
├── handler.go
├── service.go
└── repository.go
```

Pola tersebut merupakan pedoman, bukan kewajiban mutlak. File tambahan dapat dibuat ketika suatu logika cukup kompleks untuk dipisahkan, misalnya `fefo.go` untuk aturan FEFO.

Jangan membuat file kosong hanya untuk mengikuti pola.

### 11.4 Fungsi Model

Folder `internal/model/` menyimpan struct data yang digunakan bersama oleh beberapa lapisan dan modul backend.

Contohnya:

- `user.go` menyimpan struktur data pengguna.
- `medicine.go` menyimpan struktur data obat.
- `batch.go` menyimpan struktur data batch.
- `sale.go` menyimpan struktur data penjualan.

Model tidak berisi query SQL atau business logic. Repository, service, dan handler menggunakan model yang sama agar struktur data tetap konsisten.

File model tambahan dibuat ketika dibutuhkan oleh implementasi fitur dan dikoordinasikan dengan Backend Lead.

### 11.5 Perbedaan `internal/view` dan `web`

- `web/templates/` berisi file tampilan HTML yang dibuat dan dikelola Frontend.
- `web/static/` berisi CSS, JavaScript, dan gambar.
- `internal/view/` berisi kode Go untuk memuat dan merender template HTML ketika diperlukan dalam integrasi backend dan frontend.

Dengan demikian, Frontend mengerjakan isi dan desain halaman, sedangkan kode Go pada `internal/view/` menghubungkan template dengan handler.

Folder `internal/view/` tidak harus dibuat pada repository awal sebelum mekanisme render template diimplementasikan.

## 12. Pembagian Tanggung Jawab dan Pedoman Pengembangan

### 12.1 Pembagian Tanggung Jawab

| Anggota/Bagian | Tanggung Jawab |
|---|---|
| **Backend Lead** | Menyiapkan fondasi aplikasi, konfigurasi, koneksi database, model bersama awal, middleware, migration awal, seed, standar integrasi, dan review perubahan skema. |
| **Backend Owner** | Mengembangkan autentikasi login/logout, manajemen pengguna, data master, dan agregasi dashboard. |
| **Backend Kasir** | Mengembangkan penjualan, FEFO, alokasi stok, pembayaran, retur konsumen, dan perhitungan keuntungan. |
| **Backend Gudang** | Mengembangkan penerimaan, batch, persediaan, stock opname, pemblokiran batch, pemusnahan, dan retur ke PBF. |
| **Frontend 1** | Mengembangkan login, layout bersama, data master, penerimaan, persediaan, stock opname, kedaluwarsa, dan manajemen pengguna. |
| **Frontend 2** | Mengembangkan dashboard, halaman kasir, retur konsumen, dan laporan. |

Pembagian tersebut merupakan tanggung jawab utama masing-masing anggota. Integrasi antarmodul tetap dilakukan bersama.

### 12.2 Kepemilikan Folder dan File

- Backend Lead menjaga fondasi bersama pada `cmd/`, `internal/config/`, `internal/database/`, `internal/model/`, `internal/middleware/`, dan `internal/router/`.
- Backend Owner membuat dan mengelola `internal/auth/` serta `internal/owner/`.
- Backend Kasir membuat dan mengelola `internal/cashier/`.
- Backend Gudang membuat dan mengelola `internal/warehouse/`.
- Frontend 1 dan Frontend 2 membuat serta mengelola halaman dan asset pada `web/` sesuai pembagian fitur.
- `internal/view/` dibuat ketika proses render HTML perlu diintegrasikan.
- `migrations/` dikoordinasikan Backend Lead. Developer fitur boleh mengusulkan perubahan skema melalui migration baru.
- `seed/` menjadi data dummy standar tim. Penambahan data dummy dilakukan melalui file baru yang berurutan.
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

Pengelolaan autentikasi dibagi antara Backend Lead dan Backend Owner.

**Backend Lead:**

- Menyediakan middleware untuk memeriksa session pengguna.
- Menyediakan middleware untuk membatasi akses berdasarkan role.
- Menyediakan fungsi untuk mengambil informasi pengguna dari request context.
- Mengintegrasikan middleware dengan router utama.

**Backend Owner:**

- Membuat fitur login dan logout.
- Memeriksa username dan password terhadap tabel `users`.
- Melakukan verifikasi password menggunakan bcrypt.
- Memeriksa status akun aktif.
- Menghasilkan token session acak.
- Menyimpan, memeriksa, dan menghapus session melalui repository.
- Mengintegrasikan proses autentikasi dengan middleware yang disediakan Backend Lead.
- Mengembangkan manajemen akun pengguna sesuai hak akses OWNER.

**Frontend 1:**

- Membuat halaman login.
- Membuat form username dan password.
- Mengintegrasikan form dengan endpoint login yang disepakati.

Sistem tidak menyediakan registrasi publik. Akun pengguna dikelola oleh OWNER.

Target mekanisme session yang dikembangkan:

- Satu akun hanya memiliki satu session aktif.
- Login dari perangkat baru menggantikan session sebelumnya.
- Session berakhir ketika pengguna tidak aktif selama 30 menit.
- Masa berlaku maksimum session adalah 8 jam.
- Logout menghapus session dari database.

Ketentuan tersebut baru dianggap tersedia setelah tabel `sessions`, repository session, dan pemeriksaan middleware selesai diimplementasikan serta diuji. Middleware yang tersedia saat ini masih menyediakan kontrak pemeriksaan session dan belum menjadi implementasi autentikasi lengkap.

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
5. Jangan melakukan push langsung ke `main`.
6. Hindari mengubah file fondasi atau file milik anggota lain tanpa koordinasi.
7. Perubahan pada `go.mod`, `go.sum`, migration, seed, router, konfigurasi, dan koneksi database perlu diperiksa sebelum digabungkan.

Contoh pembuatan branch:

```bash
git checkout -b feature/nama-fitur
```

Contoh mengirim perubahan:

```bash
git add .
git commit -m "feat: menambahkan fitur penerimaan barang"
git push origin feature/nama-fitur
```

Setelah itu, buat Pull Request ke branch `main`.

Aturan review dan perlindungan branch dapat diatur melalui pengaturan repository GitHub. Proyek ini tidak mewajibkan folder atau file `.github`.

## 13. Dokumentasi Tambahan

Dokumentasi teknis proyek dapat disimpan pada folder `docs/`, meliputi:

- ERD dan relasi antartabel (`ERD.png`).
- Struktur database dan penjelasan tabel (`DATABASE.md`).
- Panduan struktur kode, pembagian modul, dan proses integrasi (`DEVELOPMENT.md`).
- Rancangan antarmuka dan alur proses apabila diperlukan.

Folder dan file dokumentasi dibuat ketika dokumennya tersedia. Setiap perubahan penting terhadap database atau alur bisnis sebaiknya diperbarui pada dokumentasi terkait.

## 14. Batasan Sistem

Sistem dikembangkan sesuai kebutuhan pengelolaan stok Apotek Sumber Waras. Oleh karena itu, beberapa hal berikut tidak termasuk dalam lingkup pengembangan:

- Integrasi langsung dengan sistem BPOM, PBF, atau pihak eksternal lainnya.
- Payment gateway untuk pembayaran nontunai.
- Manajemen data pelanggan.
- Resep elektronik dan integrasi dengan dokter.
- Otomatisasi pemesanan atau pengadaan barang ke PBF.
- Sistem akuntansi penuh.

Pengembangan difokuskan pada pencatatan persediaan berbasis batch, transaksi penjualan, retur, stock opname, dan pelaporan yang dapat ditelusuri.