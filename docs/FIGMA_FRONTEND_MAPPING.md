# Pemetaan ekspor layer Figma ke frontend

File TXT dari Figma merupakan ekspor properti layer (Auto Layout, ukuran, warna, font, dan teks), bukan stylesheet CSS yang memiliki selector. Karena itu propertinya diterjemahkan ke HTML semantik dan CSS per halaman.

| Ekspor Figma | Template | CSS | JavaScript / Interaksi |
|---|---|---|---|
| Login - Apotek Sumber Waras.txt | `web/templates/auth/login.html` | `web/static/css/login.css` | `web/static/js/login.js` |
| Dashboard Apotek Sumber Waras - Re.txt | `web/templates/dashboard.html` | `web/static/css/dashboard.css` | `web/static/js/interactions.js` |
| Kasir Apotek sumber waras.txt | `web/templates/kasir.html` | `web/static/css/kasir.css` | `web/static/js/kasir.js` |
| Penerimaan Barang dari PBF - Apotek Sumber waras.txt | `web/templates/penerimaan.html` | `web/static/css/penerimaan.css` | `web/static/js/interactions.js` |
| Cek Stok & Penelusuran Batch - Apotek Sumber Waras.txt | `web/templates/cek_stok.html` | `web/static/css/cek_stok.css` | `web/static/js/interactions.js` |
| Retur Konsumen - Apotek Sumber Waras.txt | `web/templates/retur.html` | `web/static/css/retur.css` | `web/static/js/interactions.js` |
| Stock Opname Apotek Sumber Waras.txt | `web/templates/stock_opname.html` | `web/static/css/stock_opname.css` | `web/static/js/stock_opname.js` |
| Obat Kedaluwarsa dan Batch Diblokir.txt | `web/templates/kedaluwarsa.html` | `web/static/css/kedaluwarsa.css` | `web/static/js/interactions.js` |
| Laporan (Stok per Batch) - Apotek.txt | `web/templates/laporan.html` | `web/static/css/laporan.css` | `web/static/js/interactions.js` |
| Data Master - Apotek Sumber Waras.txt | `web/templates/data_master.html` | `web/static/css/data_master.css` | `web/static/js/data_master.js` |
| Manajemen Pengguna - Apotek Sumber.txt | `web/templates/manajemen_pengguna.html` | `web/static/css/manajemen_pengguna.css` | `web/static/js/interactions.js` |
| Struk Penjualan Apotek Sumber Waras.txt | `web/templates/cashier/struk_penjualan.html` | `web/static/css/struk_penjualan.css` | `window.print()` dari `interactions.js` |

Header, sidebar, dan struktur shell digunakan bersama melalui `layouts/base.html` dan `partials/`. File CSS per halaman mengatur isi spesifik, sementara `base.css` dan `layout.css` mengatur token desain serta shell aplikasi.
