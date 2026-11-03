CREATE TABLE IF NOT EXISTS users (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    nama VARCHAR(100) NOT NULL,
    username VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role ENUM('OWNER', 'KASIR', 'GUDANG') NOT NULL,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    UNIQUE KEY uq_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS suppliers (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    nama_pbf VARCHAR(150) NOT NULL,
    alamat TEXT NULL,
    kontak VARCHAR(50) NULL,
    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS medicines (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    kode VARCHAR(50) NOT NULL,
    nama VARCHAR(150) NOT NULL,
    kategori VARCHAR(100) NULL,
    satuan_dasar VARCHAR(50) NOT NULL,
    stok_minimum DECIMAL(15,3) NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uq_medicines_kode (kode)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS medicine_units (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    medicine_id BIGINT UNSIGNED NOT NULL,
    nama_satuan VARCHAR(50) NOT NULL,
    isi_per_satuan_dasar DECIMAL(15,3) NOT NULL,
    harga_jual DECIMAL(15,2) NOT NULL DEFAULT 0,
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    PRIMARY KEY (id),
    UNIQUE KEY uq_medicine_units_medicine_name (medicine_id, nama_satuan),
    KEY idx_medicine_units_medicine_id (medicine_id),
    CONSTRAINT fk_medicine_units_medicine
        FOREIGN KEY (medicine_id) REFERENCES medicines (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS purchases (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    supplier_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    no_faktur VARCHAR(100) NOT NULL,
    tgl_faktur DATE NOT NULL,
    total DECIMAL(15,2) NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    UNIQUE KEY uq_purchases_supplier_invoice (supplier_id, no_faktur),
    KEY idx_purchases_user_id (user_id),
    KEY idx_purchases_tgl_faktur (tgl_faktur),
    CONSTRAINT fk_purchases_supplier
        FOREIGN KEY (supplier_id) REFERENCES suppliers (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_purchases_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS batches (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    medicine_id BIGINT UNSIGNED NOT NULL,
    no_batch VARCHAR(100) NOT NULL,
    tgl_kadaluarsa DATE NOT NULL,
    status_batch ENUM('AKTIF', 'DIBLOKIR') NOT NULL DEFAULT 'AKTIF',
    alasan_blokir TEXT NULL,
    tanggal_blokir DATETIME NULL,
    PRIMARY KEY (id),
    UNIQUE KEY uq_batches_medicine_no_batch (medicine_id, no_batch),
    KEY idx_batches_expiry_status (tgl_kadaluarsa, status_batch),
    CONSTRAINT fk_batches_medicine
        FOREIGN KEY (medicine_id) REFERENCES medicines (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS batch_receipts (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    batch_id BIGINT UNSIGNED NOT NULL,
    purchase_id BIGINT UNSIGNED NOT NULL,
    qty_diterima DECIMAL(15,3) NOT NULL,
    qty_sisa DECIMAL(15,3) NOT NULL DEFAULT 0,
    harga_beli_per_dasar DECIMAL(15,2) NOT NULL,
    tanggal_terima DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_batch_receipts_batch_id (batch_id),
    KEY idx_batch_receipts_purchase_id (purchase_id),
    KEY idx_batch_receipts_fefo (batch_id, tanggal_terima),
    CONSTRAINT fk_batch_receipts_batch
        FOREIGN KEY (batch_id) REFERENCES batches (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_batch_receipts_purchase
        FOREIGN KEY (purchase_id) REFERENCES purchases (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sales (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    tanggal DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    total DECIMAL(15,2) NOT NULL DEFAULT 0,
    metode_bayar ENUM('TUNAI', 'NONTUNAI') NOT NULL,
    jumlah_bayar DECIMAL(15,2) NOT NULL DEFAULT 0,
    PRIMARY KEY (id),
    KEY idx_sales_user_id (user_id),
    KEY idx_sales_tanggal (tanggal),
    CONSTRAINT fk_sales_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sale_items (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    sale_id BIGINT UNSIGNED NOT NULL,
    unit_id BIGINT UNSIGNED NOT NULL,
    qty_input DECIMAL(15,3) NOT NULL,
    qty_dasar DECIMAL(15,3) NOT NULL,
    harga_jual_snapshot DECIMAL(15,2) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_sale_items_sale_id (sale_id),
    KEY idx_sale_items_unit_id (unit_id),
    CONSTRAINT fk_sale_items_sale
        FOREIGN KEY (sale_id) REFERENCES sales (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_sale_items_unit
        FOREIGN KEY (unit_id) REFERENCES medicine_units (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS sale_item_batches (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    sale_item_id BIGINT UNSIGNED NOT NULL,
    batch_receipt_id BIGINT UNSIGNED NOT NULL,
    qty_dasar DECIMAL(15,3) NOT NULL,
    harga_beli_snapshot DECIMAL(15,2) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_sale_item_batches_sale_item_id (sale_item_id),
    KEY idx_sale_item_batches_receipt_id (batch_receipt_id),
    CONSTRAINT fk_sale_item_batches_sale_item
        FOREIGN KEY (sale_item_id) REFERENCES sale_items (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_sale_item_batches_receipt
        FOREIGN KEY (batch_receipt_id) REFERENCES batch_receipts (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS returns (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    sale_item_batch_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    jenis_pengembalian VARCHAR(50) NOT NULL,
    qty_dasar DECIMAL(15,3) NOT NULL,
    alasan TEXT NULL,
    kondisi_obat VARCHAR(100) NULL,
    status_kelayakan ENUM('MENUNGGU_PEMERIKSAAN', 'LAYAK_JUAL', 'TIDAK_LAYAK_JUAL') NOT NULL DEFAULT 'MENUNGGU_PEMERIKSAAN',
    nilai_refund DECIMAL(15,2) NOT NULL DEFAULT 0,
    tanggal DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_returns_sale_item_batch_id (sale_item_batch_id),
    KEY idx_returns_user_id (user_id),
    KEY idx_returns_status_tanggal (status_kelayakan, tanggal),
    CONSTRAINT fk_returns_sale_item_batch
        FOREIGN KEY (sale_item_batch_id) REFERENCES sale_item_batches (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_returns_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS stock_opnames (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    batch_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    qty_sistem DECIMAL(15,3) NOT NULL,
    qty_fisik DECIMAL(15,3) NOT NULL,
    catatan TEXT NULL,
    tanggal DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_stock_opnames_batch_id (batch_id),
    KEY idx_stock_opnames_user_id (user_id),
    KEY idx_stock_opnames_tanggal (tanggal),
    CONSTRAINT fk_stock_opnames_batch
        FOREIGN KEY (batch_id) REFERENCES batches (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_stock_opnames_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS stock_opname_details (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    stock_opname_id BIGINT UNSIGNED NOT NULL,
    batch_receipt_id BIGINT UNSIGNED NOT NULL,
    qty_penyesuaian DECIMAL(15,3) NOT NULL,
    qty_sisa_sebelum DECIMAL(15,3) NOT NULL,
    qty_sisa_sesudah DECIMAL(15,3) NOT NULL,
    PRIMARY KEY (id),
    KEY idx_stock_opname_details_opname_id (stock_opname_id),
    KEY idx_stock_opname_details_receipt_id (batch_receipt_id),
    CONSTRAINT fk_stock_opname_details_opname
        FOREIGN KEY (stock_opname_id) REFERENCES stock_opnames (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_stock_opname_details_receipt
        FOREIGN KEY (batch_receipt_id) REFERENCES batch_receipts (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS batch_disposals (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    batch_receipt_id BIGINT UNSIGNED NOT NULL,
    user_id BIGINT UNSIGNED NOT NULL,
    jenis ENUM('PEMUSNAHAN', 'RETUR_PBF') NOT NULL,
    sumber_tindak_lanjut ENUM('INTERNAL_APOTEK', 'BPOM', 'PRODUSEN_PBF') NOT NULL,
    alasan VARCHAR(100) NOT NULL,
    qty_dasar DECIMAL(15,3) NOT NULL,
    nilai_kompensasi DECIMAL(15,2) NOT NULL DEFAULT 0,
    catatan TEXT NULL,
    tanggal DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_batch_disposals_receipt_id (batch_receipt_id),
    KEY idx_batch_disposals_user_id (user_id),
    KEY idx_batch_disposals_tanggal (tanggal),
    CONSTRAINT fk_batch_disposals_receipt
        FOREIGN KEY (batch_receipt_id) REFERENCES batch_receipts (id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_batch_disposals_user
        FOREIGN KEY (user_id) REFERENCES users (id)
        ON UPDATE CASCADE ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
