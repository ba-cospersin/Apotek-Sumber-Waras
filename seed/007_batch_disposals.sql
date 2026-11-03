-- Seed file: 007_batch_disposals.sql
-- Urutan file dikendalikan berdasarkan awalan nomor pada nama file.
-- Jangan mengubah file yang sudah diterapkan pada database bersama;
-- tambahkan file seed bernomor baru untuk data tambahan.

-- 14. BATCH_DISPOSALS: contoh pemusnahan internal dan retur PBF karena penarikan.
INSERT INTO batch_disposals
    (id, batch_receipt_id, user_id, jenis, sumber_tindak_lanjut,
     alasan, qty_dasar, nilai_kompensasi, catatan, tanggal)
VALUES
    (1, 8, 3, 'PEMUSNAHAN', 'INTERNAL_APOTEK',
     'RUSAK', 5, 0.00,
     'Contoh lima tablet rusak yang dimusnahkan oleh apotek.',
     DATE_SUB(NOW(), INTERVAL 2 DAY)),
    (2, 9, 3, 'RETUR_PBF', 'PRODUSEN_PBF',
     'PENARIKAN', 10, 12500.00,
     'Contoh sepuluh kapsul batch diblokir yang dikembalikan ke PBF karena penarikan.',
     DATE_SUB(NOW(), INTERVAL 1 DAY));
