-- Seed file: 002_master_data.sql
-- Urutan file dikendalikan berdasarkan awalan nomor pada nama file.
-- Jangan mengubah file yang sudah diterapkan pada database bersama;
-- tambahkan file seed bernomor baru untuk data tambahan.

-- 2. SUPPLIERS: nama PBF fiktif.
INSERT INTO suppliers (id, nama_pbf, alamat, kontak)
VALUES
    (1, 'PT Sehat Farma Distribusi (DUMMY)', 'Jl. Industri Sehat No. 10, Semarang', '024-555-0101'),
    (2, 'PT Medika Sentosa Abadi (DUMMY)', 'Jl. Niaga Medika No. 25, Semarang', '024-555-0202');

-- 3. MEDICINES: stok tidak disimpan di tabel ini.
INSERT INTO medicines
    (id, kode, nama, kategori, satuan_dasar, stok_minimum, is_active)
VALUES
    (1, 'PAR001', 'Paracetamol 500 mg Tablet', 'Analgesik dan Antipiretik', 'tablet', 100, 1),
    (2, 'AMX001', 'Amoxicillin 500 mg Kapsul', 'Antibiotik', 'kapsul', 50, 1),
    (3, 'CTM001', 'CTM 4 mg Tablet', 'Antihistamin', 'tablet', 100, 1),
    (4, 'ORS001', 'Oralit Sachet', 'Rehidrasi Oral', 'sachet', 120, 1),
    (5, 'VTC001', 'Vitamin C 500 mg Tablet', 'Vitamin', 'tablet', 50, 1),
    (6, 'SIR001', 'Parasetamol Sirup 120 mg/5 mL', 'Obat Sirup', 'mL', 100, 1);

-- 4. MEDICINE_UNITS: isi_per_satuan_dasar selalu dikonversi ke satuan dasar.
INSERT INTO medicine_units
    (id, medicine_id, nama_satuan, isi_per_satuan_dasar, harga_jual, is_active)
VALUES
    (1, 1, 'tablet', 1, 1000, 1),
    (2, 1, 'strip', 10, 9000, 1),
    (3, 1, 'box', 100, 85000, 1),
    (4, 2, 'kapsul', 1, 2500, 1),
    (5, 2, 'strip', 10, 22000, 1),
    (6, 2, 'box', 100, 200000, 1),
    (7, 3, 'tablet', 1, 500, 1),
    (8, 3, 'strip', 10, 4500, 1),
    (9, 4, 'sachet', 1, 3000, 1),
    (10, 4, 'box', 25, 70000, 1),
    (11, 5, 'tablet', 1, 1000, 1),
    (12, 5, 'strip', 10, 9000, 1),
    (13, 6, 'mL', 1, 250, 1),
    (14, 6, 'botol', 60, 15000, 1);
