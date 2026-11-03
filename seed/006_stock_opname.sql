-- Seed file: 006_stock_opname.sql
-- Urutan file dikendalikan berdasarkan awalan nomor pada nama file.
-- Jangan mengubah file yang sudah diterapkan pada database bersama;
-- tambahkan file seed bernomor baru untuk data tambahan.

-- 12. STOCK_OPNAMES: opname batch PAR-A dan CTM mendekati kedaluwarsa.
INSERT INTO stock_opnames
    (id, batch_id, user_id, qty_sistem, qty_fisik, catatan, tanggal)
VALUES
    (1, 1, 3, 1190, 1189,
     'Contoh selisih kurang satu tablet pada batch PAR-A.',
     DATE_SUB(NOW(), INTERVAL 1 DAY)),
    (2, 8, 3, 45, 44,
     'Contoh selisih kurang satu tablet pada batch CTM mendekati kedaluwarsa.',
     DATE_SUB(NOW(), INTERVAL 1 DAY));

-- 13. STOCK_OPNAME_DETAILS: total penyesuaian sama dengan qty_fisik - qty_sistem.
INSERT INTO stock_opname_details
    (id, stock_opname_id, batch_receipt_id, qty_penyesuaian,
     qty_sisa_sebelum, qty_sisa_sesudah)
VALUES
    (1, 1, 1, -1, 900, 899),
    (2, 2, 8, -1, 45, 44);
