# Rancangan Final — Sistem Stok Apotek Berbasis Batch

Dokumen ini adalah acuan kerja untuk proyek **Apotek Sumber Waras**. Isinya dirangkum dari rancangan final yang sudah ditetapkan pemilik proyek (BAB I–VI) dan **tidak boleh diubah tanpa keputusan eksplisit pemilik proyek**.

Status acuan: 25 November 2026.

---

## 1. Aktor dan Tanggung Jawab

Sistem hanya memiliki **tiga role**. Tidak ada pendaftaran akun publik dan tidak ada role tambahan.

| Aktor | Peran | Hak akses sesuai rancangan final |
|---|---|---|
| **Owner** | Pengambil keputusan dan penentu harga jual | Seluruh fitur, termasuk dashboard, laporan laba kotor dan kerugian, pengaturan harga jual, **peninjauan kelayakan obat retur**, pemblokiran batch, dan manajemen pengguna. |
| **Kasir** | Melayani transaksi penjualan dan pengembalian obat dari konsumen | Penjualan, pemeriksaan stok yang boleh dijual, **pencatatan retur konsumen berdasarkan transaksi asal**, dan pencetakan struk. |
| **Petugas Gudang** | Mengelola penerimaan barang dan persediaan fisik | Penerimaan barang dari PBF, pemeriksaan stok per batch, stock opname, pemblokiran batch, serta pencatatan pemusnahan atau pengembalian stok ke PBF. |

### Aturan penting

- **Kasir tidak menilai kelayakan obat retur.** Kasir hanya mencatat retur.
- **Petugas Gudang tidak menangani retur konsumen.** Gudang menangani batch, stock opname, pemblokiran, dan tindak lanjut ke PBF/pemusnahan.
- **Owner yang meninjau dan menetapkan kelayakan retur.**
- Owner memiliki seluruh fitur, termasuk fitur yang secara operasional dilakukan Kasir/Gudang.

---

## 2. Matriks Authorization (Final)

Legend: ✅ = boleh, ❌ = tidak boleh.

| No | Aksi | Owner | Kasir | Gudang | Catatan |
|---:|---|:---:|:---:|:---:|---|
| 1 | Login dan logout | ✅ | ✅ | ✅ | Semua role. |
| 2 | Lihat profil sendiri | ✅ | ✅ | ✅ | Halaman profil hanya menampilkan data. |
| 3 | Ubah informasi akun sendiri | ❌ | ❌ | ❌ | Rancangan final 1.4: pembuatan/perubahan/penonaktifan akun dilakukan Owner. Belum ada self-edit profil. |
| 4 | Buat akun pengguna | ✅ | ❌ | ❌ | Owner melalui Manajemen Pengguna. |
| 5 | Ubah/nonaktifkan akun pengguna | ✅ | ❌ | ❌ | Gunakan `is_active`; jangan hard delete agar riwayat tetap utuh. |
| 6 | Lihat daftar pengguna | ✅ | ❌ | ❌ | Manajemen Pengguna hanya Owner. |
| 7 | Dashboard | ✅ | ❌ | ❌ | Rancangan final: dashboard dan laporan untuk Owner. |
| 8 | Laporan stok, expiry, laba kotor, kerugian | ✅ | ❌ | ❌ | Laporan hanya Owner. |
| 9 | Lihat menu/data master (obat, satuan, konversi, PBF) | ✅ | ❌ | ❌ | Gudang boleh membaca data obat/PBF sebagai referensi formulir penerimaan, tetapi tidak membuka menu Data Master. |
| 10 | Tambah/ubah data master | ✅ | ❌ | ❌ | Data Master hanya Owner. |
| 11 | Ubah harga jual | ✅ | ❌ | ❌ | Harga jual ditentukan Owner. |
| 12 | Mencatat penjualan | ✅ | ✅ | ❌ | Owner (seluruh fitur) dan Kasir. |
| 13 | Cetak struk | ✅ | ✅ | ❌ | Mengikuti transaksi penjualan. |
| 14 | Melihat seluruh penjualan | ✅ | ❌ | ❌ | Owner dapat melihat semua transaksi. |
| 15 | Melihat penjualan yang ia layani sendiri | ✅ | ✅ | ❌ | Owner melihat semua; Kasir hanya transaksi miliknya. |
| 16 | Mencatat retur konsumen | ✅ | ✅ | ❌ | Retur konsumen berdasarkan transaksi asal. |
| 17 | Melihat riwayat/detail retur konsumen | ✅ | ✅ | ❌ | Kasir perlu melihat retur yang ia catat. |
| 18 | **Meninjau/menetapkan kelayakan retur** | ✅ | ❌ | ❌ | **Hanya Owner.** Koreksi dari matriks lama yang memberi Gudang ✅. |
| 19 | Mencatat penerimaan barang dari PBF | ✅ | ❌ | ✅ | Owner dan Gudang. |
| 20 | Melihat riwayat penerimaan | ✅ | ❌ | ✅ | Owner dan Gudang. |
| 21 | Mencatat stock opname | ✅ | ❌ | ✅ | Owner dan Gudang. |
| 22 | Melihat stok per batch dan penelusuran batch | ✅ | ❌ | ✅ | Kasir melihat stok boleh dijual di halaman Kasir; bukan halaman Cek Stok global. |
| 23 | Blokir batch | ✅ | ❌ | ✅ | Pemblokiran berlaku untuk seluruh lapisan penerimaan batch tersebut. |
| 24 | Tindak lanjut pemusnahan / retur ke PBF | ✅ | ❌ | ✅ | Berbeda dari retur konsumen. |
| 25 | Manajemen pengguna | ✅ | ❌ | ❌ | Hanya Owner. |

### Koreksi terhadap matriks lama

1. **“Meninjau kelayakan obat untuk di retur”**: Gudang ❌, bukan ✅. Hanya Owner ✅. Kasir ❌.
2. **“Login dan ubah informasi akun”**: login/lihat profil ✅ untuk semua; **ubah akun hanya Owner**. Tidak ada self-service perubahan akun di rancangan final.
3. **“Melihat penjualan milik sendiri”**: dipisah menjadi “melihat seluruh penjualan” (Owner) dan “melihat penjualan yang ia layani” (Kasir). Owner tidak punya konsep “penjualan milik sendiri”.
4. **“Melihat dan mengubah penerimaan, opname, dan pemusnahan”**: jangan gunakan kata “mengubah”. Rancangan final melarang perubahan stok tanpa catatan transaksi. Yang benar: **mencatat** dan **melihat riwayat**.
5. **“Buat dan penghapusan akun”**: gunakan **buat, ubah, nonaktifkan**. Tidak ada penghapusan permanen akun.
6. **“Lihat data master”**: Gudang ❌ untuk menu Data Master, tetapi boleh membaca data obat/PBF pada formulir penerimaan. Jangan disamakan.
7. **“Lihat stok per batch”**: Kasir ❌ untuk halaman Cek Stok global. Kasir tetap melihat jumlah stok boleh dijual di halaman Kasir dan batch asal pada alur retur.

---

## 3. Akses Halaman per Role

| Halaman / URL | Owner | Kasir | Gudang |
|---|:---:|:---:|:---:|
| `/dashboard` | ✅ | ❌ | ❌ |
| `/kasir` | ✅ | ✅ | ❌ |
| `/penerimaan` | ✅ | ❌ | ✅ |
| `/cek-stok` | ✅ | ❌ | ✅ |
| `/retur` | ✅ | ✅ | ❌ |
| `/stock-opname` | ✅ | ❌ | ✅ |
| `/kedaluwarsa` | ✅ | ❌ | ✅ |
| `/laporan` | ✅ | ❌ | ❌ |
| `/data-master` | ✅ | ❌ | ❌ |
| `/manajemen-pengguna` | ✅ | ❌ | ❌ |
| `/profil` | ✅ | ✅ | ✅ |
| `/struk-penjualan` | ✅ | ✅ | ❌ |

Catatan: sidebar yang sekarang sudah mengikuti pembagian ini. Namun **route halaman wajib memeriksa role di server**, bukan hanya menyembunyikan menu di sidebar. Saat ini halaman masih memakai `RequireLogin` saja sehingga user yang login bisa membuka URL halaman role lain secara langsung.

---

## 4. Alur Retur Konsumen (Final)

1. **Kasir** mencari transaksi penjualan asal.
2. Kasir memilih alokasi batch asal, mengisi jumlah, jenis pengembalian, alasan, dan kondisi obat.
3. Retur disimpan dengan status **`MENUNGGU_PEMERIKSAAN`**. Stok **belum** ditambahkan.
4. **Owner** memeriksa kondisi fisik dan menetapkan:
   - `LAYAK_JUAL` → jumlah dikembalikan ke lapisan penerimaan asal dan boleh dijual kembali (selama batch tidak diblokir dan belum memasuki batas kedaluwarsa).
   - `TIDAK_LAYAK_JUAL` → stok tidak dikembalikan; biaya perolehan dicatat sebagai kerugian.
5. Status kelayakan hanya boleh ditetapkan **satu kali**.
6. **Gudang tidak terlibat dalam retur konsumen.** Gudang hanya menangani pemblokiran batch dan tindak lanjut pemusnahan/retur PBF.

### Aturan retur

- Retur tidak dibatasi jumlah hari.
- Wajib merujuk transaksi penjualan yang berhasil.
- Jumlah kumulatif retur per alokasi tidak boleh melebihi jumlah yang dibeli.
- Refund memakai harga jual historis (snapshot) dari transaksi asal.
- Obat retur tidak otomatis menjadi stok layak jual.
- `POST /retur` tidak boleh menerima `status_kelayakan` dari client; server harus memaksa `MENUNGGU_PEMERIKSAAN`.
- Endpoint penetapan kelayakan harus **Owner-only**.

---

## 5. Aturan Batch dan Stok

- `batches` = identitas batch fisik (obat + nomor batch + tanggal kedaluwarsa + status blokir).
- `batch_receipts` = lapisan penerimaan dari batch fisik yang sama; harga beli dan sisa stok dicatat per lapisan.
- Stok boleh dijual hanya jika:
  - `batches.status_batch = 'AKTIF'`
  - `batch_receipts.qty_sisa > 0`
  - `tgl_kadaluarsa > hari ini + 1 bulan` (ketat; tanggal yang tepat satu bulan lagi tidak boleh dijual)
- Batch diblokir: seluruh lapisan penerimaan dari batch itu tidak boleh dijual, tetapi stok tetap tercatat dan dapat ditelusuri.
- FEFO: urutkan `tgl_kadaluarsa` paling dekat, lalu `tanggal_terima` paling awal, lalu `id` penerimaan.
- Stok tidak boleh negatif. Pengurangan harus memakai transaksi database, penguncian baris, dan validasi ulang.
- Perubahan stok hanya melalui: penerimaan, penjualan, retur layak jual, stock opname, atau tindak lanjut batch. Tidak ada edit stok langsung.
- Stock opname mencatat `qty_sistem`, `qty_fisik`, dan detail penyesuaian per lapisan penerimaan; selisih tidak menimpa stok secara langsung.
- Tindak lanjut pemusnahan/retur PBF menggunakan `batch_disposals`, bukan `returns`.

---

## 6. Aturan Penjualan dan Pembayaran

- Kasir memilih obat, satuan, dan jumlah; sistem mengonversi ke satuan dasar.
- Sistem menentukan FEFO dan mencatat setiap alokasi batch pada `sale_item_batches`.
- Pembayaran:
  - Tunai: `jumlah_bayar >= total`; selisih menjadi kembalian.
  - Nontunai: `jumlah_bayar == total`.
  - Jika tidak valid: transaksi tidak disimpan dan stok tidak berkurang.
- Kembalian tidak disimpan sebagai kolom; dihitung dari `jumlah_bayar - total`.
- Harga jual dan harga beli disimpan sebagai snapshot historis.
- Laba kotor = harga jual snapshot - harga beli snapshot per alokasi yang benar-benar terjual.
- Retur mengurangi laba sesuai nilai refund dan HPP barang retur; retur tidak layak jual dicatat sebagai kerugian terpisah.

---

## 7. Status Implementasi (per 25 November 2026)

Acuan commit:
- `merge-FE` = `a24923a` (PR #6, sudah termasuk backend + UI fitur kasir).
- `origin/kasir` = `63d812b`.
- Patch authorization/retur nomor 1–4 ada di working tree, belum di-commit.

### Sudah terintegrasi

- Auth login/logout + session guard.
- Backend gudang (`internal/warehouse`) ter-wire ke app dan database.
- Backend kasir (`internal/cashier`) ter-wire ke app dan database.
- FE kasir (`kasir.js`), retur (`retur.js`), dan struk (`struk.js`) memanggil endpoint nyata.
- Integration test kasir dengan database uji: penjualan multi-batch, dua kasir bersamaan, retur kumulatif, retur bersamaan, laba, batas kedaluwarsa — lulus.

### Sudah diperbaiki (25 November 2026)

1. **Route halaman sudah memeriksa role di server** melalui `pageRoles` + `RequireRole`. Gudang tidak bisa lagi membuka `/kasir` atau `/retur`; Kasir tidak bisa membuka `/penerimaan`, `/stock-opname`, `/kedaluwarsa`, atau `/laporan`.
2. **`POST /retur/{id}/kelayakan` sekarang Owner-only.** Kasir dan Gudang mendapat 403.
3. **`POST /retur` selalu memaksa `MENUNGGU_PEMERIKSAAN`.** `status_kelayakan` dari client diabaikan, dan stok tidak ditambah saat retur dicatat.
4. **Tombol “Layak jual / Tidak layak” hanya muncul untuk Owner.** `retur.js` membaca role dari `data-user-role`; Kasir hanya melihat status “Menunggu Pemeriksaan”.
5. **Cache aset statis diperbaiki.** `PageJS`/`PageCSS` memakai versi (`?v=3`) dan handler `/static/` mengirim `Cache-Control: no-cache`, supaya JS demo lama tidak tertahan di browser. Database produksi juga sudah di-reset dari seed pada 25 November 2026.

### Sisa pekerjaan

1. **FE gudang/owner masih statis.** `penerimaan.js`, `cek_stok.js`, `stock_opname.js`, `kedaluwarsa.js`, `laporan.js`, `data_master.js` belum memanggil endpoint. Backend gudang sudah ada, tetapi UI-nya belum terhubung ke database.
2. **`GET /laporan/laba` adalah fitur Owner** yang berada di modul kasir; endpoint berjalan, tetapi `laporan.js` belum memanggilnya.
3. **Kontrak dokumentasi berbeda** dengan implementasi: docs menyebut `POST /retur/{id}/review`, implementasi `POST /retur/{id}/kelayakan`; docs menyebut `/struk-penjualan/{id}`, implementasi JSON `/kasir/struk/{id}` dan halaman `/struk-penjualan?id=`.
4. **Logika stok kasir diduplikasi** dari gudang (`KurangiStokLapisan`/`TambahStokLapisan`). Aturan satu bulan dan status blokir sudah sama, tetapi duplikasi ini berisiko menyimpang di kemudian hari.

---

## 8. Prinsip yang Tidak Boleh Dilanggar

- Tiga role: **OWNER, KASIR, GUDANG**. Jangan menambah role baru.
- Retur konsumen: **Kasir mencatat → Owner menilai kelayakan → stok hanya kembali jika LAYAK_JUAL**.
- Gudang tidak menangani retur konsumen; Gudang menangani batch, stock opname, blokir, dan disposal.
- Halaman role lain tidak boleh diakses hanya karena tahu URL; server harus menolak.
- Perubahan stok harus selalu punya catatan transaksi.
- Jangan mengubah harga historis pada transaksi lama.
- Jangan menjual batch diblokir atau batch yang kedaluwarsanya ≤ 1 bulan dari tanggal transaksi.
