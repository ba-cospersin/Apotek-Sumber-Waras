-- Seed file: 004_sales.sql
-- Urutan file dikendalikan berdasarkan awalan nomor pada nama file.
-- Jangan mengubah file yang sudah diterapkan pada database bersama;
-- tambahkan file seed bernomor baru untuk data tambahan.

-- 8. SALES: contoh pembayaran tunai dan nontunai.
INSERT INTO sales
    (id, user_id, tanggal, total, metode_bayar, jumlah_bayar)
VALUES
    (1, 2, DATE_SUB(NOW(), INTERVAL 4 DAY), 99000.00, 'TUNAI', 100000.00),
    (2, 2, DATE_SUB(NOW(), INTERVAL 3 DAY), 22000.00, 'NONTUNAI', 22000.00),
    (3, 2, DATE_SUB(NOW(), INTERVAL 2 DAY), 6000.00, 'TUNAI', 10000.00);

-- 9. SALE_ITEMS: satu detail dapat dialokasikan ke beberapa lapisan penerimaan.
INSERT INTO sale_items
    (id, sale_id, unit_id, qty_input, qty_dasar, harga_jual_snapshot)
VALUES
    (1, 1, 2, 11, 110, 900.00),
    (2, 2, 5, 1, 10, 2200.00),
    (3, 3, 9, 2, 2, 3000.00);

-- 10. SALE_ITEM_BATCHES: penjualan paracetamol mengambil stok dari dua receipt PAR-A.
INSERT INTO sale_item_batches
    (id, sale_item_id, batch_receipt_id, qty_dasar, harga_beli_snapshot)
VALUES
    (1, 1, 1, 100, 500.00),
    (2, 1, 10, 10, 520.00),
    (3, 2, 2, 10, 1250.00),
    (4, 3, 5, 2, 1700.00);
