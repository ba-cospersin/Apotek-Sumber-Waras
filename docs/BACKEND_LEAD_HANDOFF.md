# Paket Kerja Backend Lead — Apotek Sumber Waras

Dokumen ini menjelaskan perubahan pada paket repository Backend Lead dan pekerjaan integrasi yang perlu dilanjutkan setelah modul Backend Owner, Backend Kasir, dan Backend Gudang selesai.

## 1. Perubahan yang sudah dimasukkan ke ZIP ini

| File | Perubahan | Tujuan |
|---|---|---|
| `cmd/server/main.go` | Tidak lagi memanggil `router.New` secara langsung. Inisialisasi aplikasi dipanggil melalui `app.New(db)`. | Menjaga `main.go` tetap sederhana sebagai entry point server. |
| `internal/app/app.go` | Diisi fungsi `app.New(db)` untuk membuat renderer dan router. | Menjadi titik dependency wiring aplikasi. |
| `internal/router/router.go` | Semua halaman memakai URL permanen seperti `/dashboard`, `/kasir`, dan `/cek-stok`. Tidak lagi membuat URL `/__preview/...` atau query `?role=...`. | Menyamakan URL aplikasi dengan kontrak frontend. |
| `internal/view/templates.go` | URL pada wrapper `RenderDashboard` juga diganti menjadi URL permanen. | Tidak menyisakan link lama pada jalur render kompatibilitas. |
| `web/static/js/kasir.js` | Fallback URL struk diubah menjadi `/struk-penjualan`. | Tombol checkout tidak mengarah ke route lama. |
| `tests/router_test.go` | Ditambahkan pengujian route halaman, penolakan route lama, dan method selain GET. | Mengurangi risiko route lama muncul kembali. |

### Daftar URL halaman

Seluruh route halaman di bawah didaftarkan sebagai `GET`:

| Halaman | URL |
|---|---|
| Dashboard | `/dashboard` |
| Kasir | `/kasir` |
| Penerimaan Barang | `/penerimaan` |
| Cek Stok & Batch | `/cek-stok` |
| Retur Konsumen | `/retur` |
| Stock Opname | `/stock-opname` |
| Kedaluwarsa & Blokir | `/kedaluwarsa` |
| Laporan | `/laporan` |
| Data Master | `/data-master` |
| Manajemen Pengguna | `/manajemen-pengguna` |
| Struk Penjualan | `/struk-penjualan` |

Pemetaan nama halaman ke URL berada di `pagePaths` pada `internal/router/router.go`. Sidebar tetap menggunakan `.Routes` yang dikirim oleh renderer, jadi template tidak perlu menulis query role.

## 2. Catatan penting tentang status autentikasi

**Penghapusan `__preview` pada paket ini tidak berarti autentikasi sudah selesai.** Untuk sementara, route halaman langsung menampilkan identitas demo berdasarkan halaman yang dibuka. Ini hanya untuk pemeriksaan tampilan dan integrasi frontend.

Pada status repository ini:

- `internal/middleware/auth.go` sudah menyediakan cookie session, kontrak `SessionValidator`, `RequireLogin`, `SetSessionCookie`, `ClearSessionCookie`, dan `CurrentUser`.
- `internal/middleware/role.go` sudah menyediakan `RequireRole` untuk `OWNER`, `KASIR`, dan `GUDANG`.
- `internal/model/user.go` sudah menyediakan model `User` serta `ErrInvalidSession`.
- `internal/auth/handler.go` baru menampilkan halaman login. Request `POST /login` masih mengembalikan HTTP 501 sampai service autentikasi dihubungkan.
- `internal/auth/repository.go` dan `internal/auth/service.go` masih placeholder.
- Router pada paket ini **belum memasang** middleware session dan role pada route halaman.
- Handler bisnis di `internal/owner/`, `internal/cashier/`, dan `internal/warehouse/` masih placeholder.

Dengan demikian, jangan menganggap route `/dashboard` dan route lainnya sudah memiliki pembatasan akses. Sebelum aplikasi dipakai untuk operasional, Backend Owner harus menyelesaikan repository/service session dan Backend Lead harus mengganti handler demo dengan handler modul serta memasang middleware login dan role.

### Target role untuk route halaman

| Role | Halaman utama yang menjadi hak akses |
|---|---|
| `OWNER` | Semua halaman operasional, termasuk dashboard, kasir, penerimaan, persediaan, retur, laporan, data master, dan manajemen pengguna. |
| `KASIR` | `/kasir`, `/retur`, `/struk-penjualan` |
| `GUDANG` | `/penerimaan`, `/cek-stok`, `/stock-opname`, `/kedaluwarsa` |

Tabel tersebut mengikuti pembagian fitur pada rancangan sistem: OWNER memiliki akses pemantauan/pengelolaan, KASIR menangani transaksi dan retur konsumen, sedangkan GUDANG menangani penerimaan dan persediaan. Link Retur Konsumen pada sidebar juga dibatasi untuk OWNER dan KASIR. Aturan final tetap harus mengikuti kesepakatan hak akses sistem dan validasi middleware di backend. Menyembunyikan link sidebar saja tidak cukup; setiap route harus diperiksa di server.

## 3. Kontrak endpoint awal untuk koordinasi modul

Daftar berikut merupakan kontrak awal untuk disepakati bersama. Route halaman `GET` sudah didaftarkan sebagai tampilan demo pada paket ini. Endpoint aksi `POST` di bawah **belum diimplementasikan**.

| Kebutuhan | Method | URL | Modul utama |
|---|---|---|---|
| Menampilkan dashboard | GET | `/dashboard` | Owner |
| Menampilkan halaman kasir | GET | `/kasir` | Kasir |
| Membuat transaksi | POST | `/kasir/transaksi` | Kasir |
| Menampilkan struk | GET | `/struk-penjualan/{id}` atau route detail yang disepakati | Kasir |
| Melihat penerimaan | GET | `/penerimaan` | Gudang |
| Mencatat penerimaan | POST | `/penerimaan` | Gudang |
| Melihat batch | GET | `/cek-stok` | Gudang |
| Mencatat retur konsumen | POST | `/retur` | Kasir |
| Meninjau retur | POST | `/retur/{id}/review` | Kasir/Owner sesuai keputusan tim |
| Mencatat stock opname | POST | `/stock-opname` | Gudang |
| Memblokir batch | POST | `/kedaluwarsa/{id}/blokir` | Gudang |
| Melihat laporan | GET | `/laporan` | Owner |
| Melihat data master | GET | `/data-master` | Owner |
| Mengelola pengguna | GET | `/manajemen-pengguna` | Owner |

Endpoint update, delete, pencarian, filter, pagination, dan format payload harus ditetapkan sebelum frontend dan backend mengimplementasikannya. Untuk URL detail seperti `{id}`, tentukan apakah akan memakai pola route Go `/{id}` atau query parameter berdasarkan versi Go dan rancangan handler tim.

## 4. Pekerjaan yang menjadi tanggung jawab Backend Lead

### A. Fondasi aplikasi

- Menjaga `cmd/server/main.go` tetap sebagai entry point.
- Menjaga `internal/app/app.go` sebagai tempat dependency wiring.
- Menjaga satu connection pool `*sql.DB` untuk seluruh modul.
- Menghubungkan renderer, repository, service, handler, middleware, dan router saat modul tersedia.
- Menjaga konfigurasi aplikasi dan penanganan error startup.

### B. Router dan middleware

- Menjadi pemilik kontrak URL dan HTTP method yang disepakati tim.
- Mendaftarkan handler modul ke URL yang sesuai.
- Memasang `RequireLogin` pada route yang membutuhkan akun.
- Memasang `RequireRole` sesuai hak akses setiap route.
- Menetapkan perilaku pengguna yang belum login (redirect ke `/login`) dan role yang tidak sesuai (HTTP 403).
- Menghapus handler identitas demo setelah handler autentikasi dan modul bisnis siap.

### C. Kontrak integrasi

- Menyamakan nama field form, tipe data, format response, pesan validasi, dan URL.
- Menentukan kontrak data bersama yang dipakai lebih dari satu modul.
- Memastikan alur stok, batch, dan FEFO tidak dibuat dengan aturan berbeda antara Kasir dan Gudang.
- Meninjau perubahan migration dari anggota backend sebelum digabungkan.

### D. Database dan kualitas data

- Menjaga migration dan seed tetap dapat dijalankan berurutan.
- Menjaga foreign key serta konsistensi model bersama.
- Memastikan proses yang mengubah stok memakai transaksi database.
- Menghindari query bisnis di handler.
- Menyediakan data dummy yang konsisten untuk pengujian lintas modul.

### E. Integrasi dan pengujian

- Menguji akses role dan session setelah autentikasi selesai.
- Menguji alur penerimaan sampai batch tersedia untuk penjualan.
- Menguji penjualan menggunakan FEFO serta pengurangan stok yang konsisten.
- Menguji retur konsumen, stock opname, pemblokiran, dan laporan.
- Menguji logout dan session kedaluwarsa.
- Memperbarui dokumentasi jika kontrak route, migration, atau alur bisnis berubah.

## 5. Pembagian kerja agar tidak tumpang tindih

| Bagian | Fokus pekerjaan |
|---|---|
| Backend Lead | `cmd/`, `internal/app/`, `internal/config/`, `internal/database/`, `internal/model/` bersama, `internal/middleware/`, `internal/router/`, migration/seed bersama, kontrak dan integration test. |
| Backend Owner | `internal/auth/` (login, logout, bcrypt, session repository/service), `internal/owner/` (dashboard, data master, manajemen pengguna). |
| Backend Kasir | `internal/cashier/` (transaksi, pembayaran, alokasi FEFO, struk, retur konsumen). |
| Backend Gudang | `internal/warehouse/` (penerimaan, batch, persediaan, stock opname, blokir, pemusnahan/retur PBF). |
| Frontend | `web/templates/` dan `web/static/`; koordinasi nama field serta URL dengan backend pemilik fitur. |

Backend Lead tidak harus mengambil alih query dan business logic dari semua modul. Fokusnya memastikan kontrak dan dependency antar modul konsisten, melakukan review, lalu menguji alur gabungan.

## 6. Urutan kerja setelah paket ini

1. Salin perubahan fondasi dari paket ini ke branch kerja Backend Lead.
2. Jalankan aplikasi dan cek seluruh URL halaman untuk memastikan frontend ter-render.
3. Sepakati kontrak endpoint aksi dan field form dengan tiap anggota backend/frontend.
4. Minta Backend Owner menyelesaikan implementasi session yang memenuhi `middleware.SessionValidator`.
5. Pasang `RequireLogin` dan `RequireRole` pada router; pada tahap ini handler demo harus diganti handler modul yang sebenarnya.
6. Hubungkan handler Owner, Kasir, dan Gudang melalui `internal/app` dan `internal/router`.
7. Jalankan unit test dan integration test, terutama transaksi stok dan FEFO.

## 7. Cara menjalankan dan menguji

Dari root repository:

```bash
# Siapkan konfigurasi lokal dari template
cp .env.example .env

# Unduh dependency
go mod download

# Jalankan migration dan data dummy
 go run ./cmd/migrate
 go run ./cmd/seed

# Jalankan aplikasi
 go run ./cmd/server
```

Pada Windows PowerShell, gunakan `Copy-Item .env.example .env` untuk menyalin konfigurasi.

Buka `http://localhost:8080/dashboard` atau URL halaman lain yang tertera pada tabel. Ganti port sesuai `APP_PORT` pada `.env`.

Pemeriksaan kode:

```bash
go test ./...
go build ./...
```

Catatan validasi paket: lingkungan penyusunan ZIP ini hanya memiliki Go 1.23.2, sedangkan `go.mod` proyek meminta Go 1.27.0. Toolchain Go 1.27.0 tidak dapat diunduh dari lingkungan ini karena akses jaringan tidak tersedia. File Go sudah dirapikan dengan `gofmt`, tetapi `go test` dan `go build` belum dapat diklaim berhasil dijalankan. Jalankan kedua perintah tersebut pada komputer yang memiliki Go 1.27.0.

## 8. Catatan distribusi

- ZIP ini tidak menyertakan `.env` maupun folder `.git`.
- `.env.example` tetap disertakan sebagai template konfigurasi.
- Buat `.env` sendiri berdasarkan database lokal masing-masing.
- Jangan menimpa branch kerja yang memiliki perubahan baru tanpa membuat backup atau memeriksa diff terlebih dahulu.
- Modul bisnis yang masih placeholder sengaja tidak diisi agar pekerjaan anggota Backend Owner, Kasir, dan Gudang tidak tertimpa.
