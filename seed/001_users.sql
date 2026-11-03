-- Seed file: 001_users.sql
-- Urutan file dikendalikan berdasarkan awalan nomor pada nama file.
-- Jangan mengubah file yang sudah diterapkan pada database bersama;
-- tambahkan file seed bernomor baru untuk data tambahan.

-- 1. USERS: satu akun untuk setiap role yang ada di rancangan.
-- Password ketiga akun demo: Apotek123!
-- Hash menggunakan bcrypt cost 10.
INSERT INTO users
    (id, nama, username, password_hash, role, is_active, created_at)
VALUES
    (1, 'Owner Demo', 'owner', '$2a$10$kWDHsZqqeC1/MYZkpXsAuOHrqxfIVPd4OfjSUiq2P7X/gF45WLyFe', 'OWNER', 1, NOW()),
    (2, 'Kasir Demo', 'kasir', '$2a$10$kWDHsZqqeC1/MYZkpXsAuOHrqxfIVPd4OfjSUiq2P7X/gF45WLyFe', 'KASIR', 1, NOW()),
    (3, 'Petugas Gudang Demo', 'gudang', '$2a$10$kWDHsZqqeC1/MYZkpXsAuOHrqxfIVPd4OfjSUiq2P7X/gF45WLyFe', 'GUDANG', 1, NOW());
