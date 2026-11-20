
-- Seed akun demo untuk setiap role.
-- Password ketiga akun: Apotek123!

INSERT INTO users
    (id, nama, username, email, password_hash, role, is_active, created_at)
VALUES
    (
        1,
        'Owner Demo',
        'owner',
        'owner@example.test',
        '$2a$10$kWDHsZqqeC1/MYZkpXsAuOHrqxfIVPd4OfjSUiq2P7X/gF45WLyFe',
        'OWNER',
        1,
        NOW()
    ),
    (
        2,
        'Kasir Demo',
        'kasir',
        'kasir@example.test',
        '$2a$10$kWDHsZqqeC1/MYZkpXsAuOHrqxfIVPd4OfjSUiq2P7X/gF45WLyFe',
        'KASIR',
        1,
        NOW()
    ),
    (
        3,
        'Petugas Gudang Demo',
        'gudang',
        'gudang@example.test',
        '$2a$10$kWDHsZqqeC1/MYZkpXsAuOHrqxfIVPd4OfjSUiq2P7X/gF45WLyFe',
        'GUDANG',
        1,
        NOW()
    );
