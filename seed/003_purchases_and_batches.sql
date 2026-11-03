-- Seed file: 003_purchases_and_batches.sql
-- Urutan file dikendalikan berdasarkan awalan nomor pada nama file.
-- Jangan mengubah file yang sudah diterapkan pada database bersama;
-- tambahkan file seed bernomor baru untuk data tambahan.

-- 5. PURCHASES: tiga faktur dummy dari dua PBF.
-- Total purchase = jumlah qty_diterima x harga_beli_per_dasar pada setiap receipt.
INSERT INTO purchases
    (id, supplier_id, user_id, no_faktur, tgl_faktur, total)
VALUES
    (1, 1, 3, 'DUMMY-FKT-001', DATE_SUB(CURDATE(), INTERVAL 14 DAY), 1185000.00),
    (2, 2, 3, 'DUMMY-FKT-002', DATE_SUB(CURDATE(), INTERVAL 9 DAY), 870000.00),
    (3, 1, 3, 'DUMMY-FKT-003', DATE_SUB(CURDATE(), INTERVAL 6 DAY), 156000.00);

-- 6. BATCHES: ada batch aktif, batch mendekati kedaluwarsa, dan batch diblokir.
-- Tanggal dibuat relatif terhadap tanggal seed dijalankan agar skenario tetap relevan.
INSERT INTO batches
    (id, medicine_id, no_batch, tgl_kadaluarsa, status_batch, alasan_blokir, tanggal_blokir)
VALUES
    (1, 1, 'PAR-A-1026', DATE_ADD(CURDATE(), INTERVAL 420 DAY), 'AKTIF', NULL, NULL),
    (2, 2, 'AMX-A-1026', DATE_ADD(CURDATE(), INTERVAL 240 DAY), 'AKTIF', NULL, NULL),
    (3, 3, 'CTM-A-1026', DATE_ADD(CURDATE(), INTERVAL 550 DAY), 'AKTIF', NULL, NULL),
    (4, 1, 'PAR-B-1026', DATE_ADD(CURDATE(), INTERVAL 600 DAY), 'AKTIF', NULL, NULL),
    (5, 4, 'ORS-A-1026', DATE_ADD(CURDATE(), INTERVAL 365 DAY), 'AKTIF', NULL, NULL),
    (6, 5, 'VTC-A-1026', DATE_ADD(CURDATE(), INTERVAL 500 DAY), 'AKTIF', NULL, NULL),
    (7, 6, 'SIR-A-1026', DATE_ADD(CURDATE(), INTERVAL 300 DAY), 'AKTIF', NULL, NULL),
    (8, 3, 'CTM-NEAR-EXP', DATE_ADD(CURDATE(), INTERVAL 20 DAY), 'AKTIF', NULL, NULL),
    (9, 2, 'AMX-BLOCK-01', DATE_ADD(CURDATE(), INTERVAL 450 DAY), 'DIBLOKIR', 'Contoh batch penarikan untuk pengujian.', NOW());

-- 7. BATCH_RECEIPTS: setiap baris adalah satu lapisan penerimaan.
-- Batch PAR-A (id 1) sengaja diterima dua kali dengan harga beli berbeda.
-- qty_sisa sudah mencerminkan transaksi, retur layak jual, disposal, dan opname di bawah.
INSERT INTO batch_receipts
    (id, batch_id, purchase_id, qty_diterima, qty_sisa, harga_beli_per_dasar, tanggal_terima)
VALUES
    (1, 1, 1, 1000, 899, 500.00, DATE_SUB(NOW(), INTERVAL 14 DAY)),
    (2, 2, 1, 500, 491, 1250.00, DATE_SUB(NOW(), INTERVAL 14 DAY)),
    (3, 3, 1, 300, 300, 200.00, DATE_SUB(NOW(), INTERVAL 14 DAY)),
    (4, 4, 2, 500, 500, 550.00, DATE_SUB(NOW(), INTERVAL 9 DAY)),
    (5, 5, 2, 100, 98, 1700.00, DATE_SUB(NOW(), INTERVAL 9 DAY)),
    (6, 6, 2, 400, 400, 500.00, DATE_SUB(NOW(), INTERVAL 9 DAY)),
    (7, 7, 2, 600, 600, 150.00, DATE_SUB(NOW(), INTERVAL 9 DAY)),
    (8, 8, 2, 50, 44, 200.00, DATE_SUB(NOW(), INTERVAL 9 DAY)),
    (9, 9, 2, 100, 90, 1250.00, DATE_SUB(NOW(), INTERVAL 9 DAY)),
    (10, 1, 3, 300, 290, 520.00, DATE_SUB(NOW(), INTERVAL 6 DAY));
