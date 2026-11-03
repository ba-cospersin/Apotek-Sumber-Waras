-- Seed file: 005_returns.sql
-- Urutan file dikendalikan berdasarkan awalan nomor pada nama file.
-- Jangan mengubah file yang sudah diterapkan pada database bersama;
-- tambahkan file seed bernomor baru untuk data tambahan.

-- 11. RETURNS: ada retur menunggu pemeriksaan dan retur layak jual.
-- Retur MENUNGGU_PEMERIKSAAN tidak menambah qty_sisa.
-- Retur LAYAK_JUAL sudah diperhitungkan dalam qty_sisa batch_receipts id 2.
INSERT INTO returns
    (id, sale_item_batch_id, user_id, jenis_pengembalian, qty_dasar,
     alasan, kondisi_obat, status_kelayakan, nilai_refund, tanggal)
VALUES
    (1, 1, 2, 'SALAH_MEMBELI', 2,
     'Contoh retur yang masih menunggu pemeriksaan.',
     'Kemasan belum diperiksa.', 'MENUNGGU_PEMERIKSAAN', 1800.00,
     DATE_SUB(NOW(), INTERVAL 2 DAY)),
    (2, 3, 2, 'SALAH_MEMBELI', 1,
     'Contoh retur yang telah dinyatakan layak jual.',
     'Kemasan utuh dan kondisi baik.', 'LAYAK_JUAL', 2200.00,
     DATE_SUB(NOW(), INTERVAL 1 DAY));
