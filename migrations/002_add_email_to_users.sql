
-- Menambahkan email untuk login menggunakan username atau email.

ALTER TABLE users
    ADD COLUMN email VARCHAR(255) NULL AFTER username;

-- Mengisi email sementara untuk akun yang sudah ada.
-- Domain .test hanya digunakan untuk pengembangan.
UPDATE users
SET email = CASE
    WHEN username = 'owner' THEN 'owner@example.test'
    WHEN username = 'kasir' THEN 'kasir@example.test'
    WHEN username = 'gudang' THEN 'gudang@example.test'
    ELSE CONCAT('user', id, '@example.test')
END
WHERE email IS NULL OR TRIM(email) = '';

-- Email wajib diisi.
ALTER TABLE users
    MODIFY COLUMN email VARCHAR(255) NOT NULL;

-- Email harus unik.
ALTER TABLE users
    ADD UNIQUE KEY uq_users_email (email);
