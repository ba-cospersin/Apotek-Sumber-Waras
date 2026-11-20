package warehouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// Repository menjadi tempat query SQL untuk modul gudang.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// TxRepository menjalankan query di dalam satu transaksi database.
// Hanya bisa dibuat lewat Repository.WithTx.
type TxRepository struct {
	tx *sql.Tx
}

// WithTx menjalankan fn di dalam transaksi.
// Jika fn mengembalikan error (atau panic), semua perubahan di-rollback.
// Jika fn sukses, transaksi di-commit.
func (r *Repository) WithTx(
	ctx context.Context,
	fn func(tx *TxRepository) error,
) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("gagal memulai transaksi: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(&TxRepository{tx: tx}); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("gagal commit transaksi: %w", err)
	}
	return nil
}

// SupplierExists memeriksa apakah PBF dengan id tersebut ada.
func (t *TxRepository) SupplierExists(
	ctx context.Context,
	id int64,
) (bool, error) {
	var n int
	err := t.tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM suppliers WHERE id = ?`,
		id,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("gagal memeriksa supplier: %w", err)
	}
	return n > 0, nil
}

// FindMedicine mengembalikan nil jika obat tidak ditemukan.
func (t *TxRepository) FindMedicine(
	ctx context.Context,
	id int64,
) (*Medicine, error) {
	var m Medicine
	err := t.tx.QueryRowContext(
		ctx,
		`SELECT id, is_active FROM medicines WHERE id = ?`,
		id,
	).Scan(&m.ID, &m.IsActive)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data obat: %w", err)
	}
	return &m, nil
}

// FindUnit mengembalikan nil jika satuan tidak ditemukan
// atau bukan milik obat tersebut.
func (t *TxRepository) FindUnit(
	ctx context.Context,
	medicineID, unitID int64,
) (*Unit, error) {
	var u Unit
	err := t.tx.QueryRowContext(
		ctx,
		`SELECT id, isi_per_satuan_dasar, is_active
		   FROM medicine_units
		  WHERE id = ? AND medicine_id = ?`,
		unitID, medicineID,
	).Scan(&u.ID, &u.IsiPerSatuanDasar, &u.IsActive)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data satuan: %w", err)
	}
	return &u, nil
}

// FindBatchForUpdate mencari batch fisik berdasarkan obat dan nomor batch.
// Baris dikunci (FOR UPDATE) sampai transaksi selesai.
// Mengembalikan nil jika batch belum ada.
func (t *TxRepository) FindBatchForUpdate(
	ctx context.Context,
	medicineID int64,
	noBatch string,
) (*Batch, error) {
	var b Batch
	err := t.tx.QueryRowContext(
		ctx,
		`SELECT id, tgl_kadaluarsa, status_batch
		   FROM batches
		  WHERE medicine_id = ? AND no_batch = ?
		    FOR UPDATE`,
		medicineID, noBatch,
	).Scan(&b.ID, &b.TglKadaluarsa, &b.Status)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data batch: %w", err)
	}
	return &b, nil
}

// InsertBatch membuat identitas batch fisik baru (status AKTIF).
func (t *TxRepository) InsertBatch(
	ctx context.Context,
	medicineID int64,
	noBatch string,
	tglKadaluarsa time.Time,
) (int64, error) {
	res, err := t.tx.ExecContext(
		ctx,
		`INSERT INTO batches (medicine_id, no_batch, tgl_kadaluarsa)
		 VALUES (?, ?, ?)`,
		medicineID, noBatch, tglKadaluarsa.Format(layoutTanggal),
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan batch: %w", err)
	}
	return res.LastInsertId()
}

// InsertPurchase menyimpan faktur PBF dengan total awal 0.
// Mengembalikan ErrFakturDuplikat jika PBF dan nomor faktur sudah ada.
func (t *TxRepository) InsertPurchase(
	ctx context.Context,
	supplierID, userID int64,
	noFaktur string,
	tglFaktur time.Time,
) (int64, error) {
	res, err := t.tx.ExecContext(
		ctx,
		`INSERT INTO purchases (supplier_id, user_id, no_faktur, tgl_faktur, total)
		 VALUES (?, ?, ?, ?, 0)`,
		supplierID, userID, noFaktur, tglFaktur.Format(layoutTanggal),
	)
	if err != nil {
		var me *mysql.MySQLError
		// 1062 = duplicate entry (unique key supplier_id + no_faktur).
		if errors.As(err, &me) && me.Number == 1062 {
			return 0, ErrFakturDuplikat
		}
		return 0, fmt.Errorf("gagal menyimpan faktur: %w", err)
	}
	return res.LastInsertId()
}

// InsertReceipt membuat lapisan penerimaan baru.
// qty_sisa awalnya sama dengan qty_diterima. Semua dalam satuan dasar.
func (t *TxRepository) InsertReceipt(
	ctx context.Context,
	batchID, purchaseID int64,
	qtyDasar, hargaBeliPerDasar float64,
	tanggalTerima time.Time,
) (int64, error) {
	res, err := t.tx.ExecContext(
		ctx,
		`INSERT INTO batch_receipts
		   (batch_id, purchase_id, qty_diterima, qty_sisa, harga_beli_per_dasar, tanggal_terima)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		batchID, purchaseID, qtyDasar, qtyDasar, hargaBeliPerDasar,
		tanggalTerima.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan penerimaan batch: %w", err)
	}
	return res.LastInsertId()
}

// UpdatePurchaseTotal mengisi total faktur setelah semua item tersimpan.
func (t *TxRepository) UpdatePurchaseTotal(
	ctx context.Context,
	purchaseID int64,
	total float64,
) error {
	_, err := t.tx.ExecContext(
		ctx,
		`UPDATE purchases SET total = ? WHERE id = ?`,
		total, purchaseID,
	)
	if err != nil {
		return fmt.Errorf("gagal memperbarui total faktur: %w", err)
	}
	return nil
}

// ---------- Riwayat penerimaan (hanya membaca) ----------

// ListPenerimaan mengambil satu halaman daftar faktur beserta total barisnya.
// Urutan: tanggal faktur terbaru dulu.
func (r *Repository) ListPenerimaan(
	ctx context.Context,
	q QueryPenerimaan,
) ([]RingkasanPenerimaan, int, error) {
	// Filter disusun dinamis, tetapi nilainya SELALU lewat tanda tanya (?)
	// supaya aman dari SQL injection.
	where := " WHERE 1=1"
	var args []any

	if q.SupplierID > 0 {
		where += " AND p.supplier_id = ?"
		args = append(args, q.SupplierID)
	}
	if q.Dari != "" {
		where += " AND p.tgl_faktur >= ?"
		args = append(args, q.Dari)
	}
	if q.Sampai != "" {
		where += " AND p.tgl_faktur <= ?"
		args = append(args, q.Sampai)
	}

	var total int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM purchases p`+where,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung daftar faktur: %w", err)
	}

	argsHalaman := append(append([]any{}, args...), q.Limit, q.Offset)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT p.id, p.supplier_id, s.nama_pbf, p.no_faktur, p.tgl_faktur,
		        p.total, u.nama,
		        (SELECT COUNT(*) FROM batch_receipts br WHERE br.purchase_id = p.id)
		   FROM purchases p
		   JOIN suppliers s ON s.id = p.supplier_id
		   JOIN users u ON u.id = p.user_id`+where+`
		  ORDER BY p.tgl_faktur DESC, p.id DESC
		  LIMIT ? OFFSET ?`,
		argsHalaman...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal mengambil daftar faktur: %w", err)
	}
	defer rows.Close()

	hasil := make([]RingkasanPenerimaan, 0)
	for rows.Next() {
		var it RingkasanPenerimaan
		var tgl time.Time
		if err := rows.Scan(
			&it.PurchaseID, &it.SupplierID, &it.NamaPBF, &it.NoFaktur,
			&tgl, &it.Total, &it.DicatatOleh, &it.JumlahItem,
		); err != nil {
			return nil, 0, fmt.Errorf("gagal membaca daftar faktur: %w", err)
		}
		it.TglFaktur = tgl.Format(layoutTanggal)
		hasil = append(hasil, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("gagal membaca daftar faktur: %w", err)
	}

	return hasil, total, nil
}

// GetPenerimaan mengambil satu faktur beserta semua lapisan penerimaannya.
// Mengembalikan nil jika faktur tidak ditemukan.
func (r *Repository) GetPenerimaan(
	ctx context.Context,
	id int64,
) (*DetailPenerimaan, error) {
	var d DetailPenerimaan
	var tgl time.Time

	err := r.db.QueryRowContext(
		ctx,
		`SELECT p.id, p.supplier_id, s.nama_pbf, p.no_faktur, p.tgl_faktur,
		        p.total, u.nama
		   FROM purchases p
		   JOIN suppliers s ON s.id = p.supplier_id
		   JOIN users u ON u.id = p.user_id
		  WHERE p.id = ?`,
		id,
	).Scan(
		&d.PurchaseID, &d.SupplierID, &d.NamaPBF, &d.NoFaktur,
		&tgl, &d.Total, &d.DicatatOleh,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil faktur: %w", err)
	}
	d.TglFaktur = tgl.Format(layoutTanggal)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT br.id, br.batch_id, b.medicine_id, m.kode, m.nama, m.satuan_dasar,
		        b.no_batch, b.tgl_kadaluarsa, b.status_batch,
		        br.qty_diterima, br.qty_sisa, br.harga_beli_per_dasar,
		        br.tanggal_terima
		   FROM batch_receipts br
		   JOIN batches b ON b.id = br.batch_id
		   JOIN medicines m ON m.id = b.medicine_id
		  WHERE br.purchase_id = ?
		  ORDER BY br.id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil item faktur: %w", err)
	}
	defer rows.Close()

	d.Items = make([]ItemDetailPenerimaan, 0)
	for rows.Next() {
		var it ItemDetailPenerimaan
		var kadaluarsa, terima time.Time
		if err := rows.Scan(
			&it.ReceiptID, &it.BatchID, &it.MedicineID, &it.KodeObat,
			&it.NamaObat, &it.SatuanDasar, &it.NoBatch, &kadaluarsa,
			&it.StatusBatch, &it.QtyDiterima, &it.QtySisa,
			&it.HargaBeliPerDasar, &terima,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca item faktur: %w", err)
		}
		it.TglKadaluarsa = kadaluarsa.Format(layoutTanggal)
		it.TanggalTerima = terima.Format("2006-01-02 15:04:05")
		d.Items = append(d.Items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca item faktur: %w", err)
	}

	return &d, nil
}

// ---------- Persediaan ----------

// escapeLike menetralkan karakter khusus LIKE (% dan _) pada kata pencarian.
// Karakter escape yang dipakai adalah tanda seru, dan harus dipasangkan
// dengan ESCAPE '!' pada query.
func escapeLike(s string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s)
}

// SumStokDapatDijual menjumlahkan qty_sisa semua lapisan obat yang boleh dijual:
// batch AKTIF, kedaluwarsa LEBIH dari batas (hari ini + 1 bulan), dan masih ada sisa.
// Syarat ini HARUS sama dengan query lapisan yang dipakai Kasir saat menjual.
func (r *Repository) SumStokDapatDijual(
	ctx context.Context,
	medicineID int64,
	batas string,
) (float64, error) {
	var total float64
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COALESCE(SUM(r.qty_sisa), 0)
		   FROM batch_receipts r
		   JOIN batches b ON b.id = r.batch_id
		  WHERE b.medicine_id = ?
		    AND b.status_batch = ?
		    AND b.tgl_kadaluarsa > ?
		    AND r.qty_sisa > 0`,
		medicineID, statusBatchAktif, batas,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung stok tersedia: %w", err)
	}
	return total, nil
}

// RingkasanStok menghitung jumlah batch per kategori (hanya yang masih punya stok).
func (r *Repository) RingkasanStok(
	ctx context.Context,
	hariIni, batas string,
) (*RingkasanStok, error) {
	var rs RingkasanStok
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN b.status_batch = ? THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN b.status_batch = ? AND b.tgl_kadaluarsa <= ? THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN b.status_batch = ? AND b.tgl_kadaluarsa > ? AND b.tgl_kadaluarsa <= ? THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN b.status_batch = ? AND b.tgl_kadaluarsa > ? THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN b.status_batch = ? AND b.tgl_kadaluarsa > ? THEN t.total ELSE 0 END), 0)
		   FROM batches b
		   JOIN (SELECT batch_id, SUM(qty_sisa) AS total
		           FROM batch_receipts
		          GROUP BY batch_id
		         HAVING SUM(qty_sisa) > 0) t ON t.batch_id = b.id`,
		statusBatchDiblokir,
		statusBatchAktif, hariIni,
		statusBatchAktif, hariIni, batas,
		statusBatchAktif, batas,
		statusBatchAktif, batas,
	).Scan(
		&rs.TotalBatch, &rs.BatchDiblokir, &rs.BatchKedaluwarsa,
		&rs.BatchMendekatiKedaluwarsa, &rs.BatchBolehDijual, &rs.StokBolehDijual,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung ringkasan stok: %w", err)
	}
	return &rs, nil
}

// ListBatch mengambil satu halaman daftar batch beserta total stoknya.
// Urutan: kedaluwarsa terdekat dulu.
func (r *Repository) ListBatch(
	ctx context.Context,
	q QueryBatch,
) ([]BatchStok, int, error) {
	where := " WHERE 1=1"
	var args []any

	if q.Kata != "" {
		pola := "%" + escapeLike(q.Kata) + "%"
		where += " AND (m.nama LIKE ? ESCAPE '!' OR m.kode LIKE ? ESCAPE '!' OR b.no_batch LIKE ? ESCAPE '!')"
		args = append(args, pola, pola, pola)
	}

	switch q.Status {
	case statusJualDiblokir:
		where += " AND b.status_batch = ?"
		args = append(args, statusBatchDiblokir)
	case statusJualKedaluwarsa:
		where += " AND b.status_batch = ? AND b.tgl_kadaluarsa <= ?"
		args = append(args, statusBatchAktif, q.HariIni)
	case statusJualMendekati:
		where += " AND b.status_batch = ? AND b.tgl_kadaluarsa > ? AND b.tgl_kadaluarsa <= ?"
		args = append(args, statusBatchAktif, q.HariIni, q.Batas)
	case statusJualBoleh:
		where += " AND b.status_batch = ? AND b.tgl_kadaluarsa > ?"
		args = append(args, statusBatchAktif, q.Batas)
	}

	if q.HanyaAdaStok {
		where += ` AND EXISTS (SELECT 1 FROM batch_receipts rs
		                        WHERE rs.batch_id = b.id AND rs.qty_sisa > 0)`
	}

	var total int
	err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		   FROM batches b
		   JOIN medicines m ON m.id = b.medicine_id`+where,
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung daftar batch: %w", err)
	}

	argsHalaman := append(append([]any{}, args...), q.Limit, q.Offset)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT b.id, b.medicine_id, m.kode, m.nama, m.satuan_dasar,
		        b.no_batch, b.tgl_kadaluarsa, b.status_batch,
		        COALESCE(SUM(r.qty_sisa), 0), COUNT(r.id)
		   FROM batches b
		   JOIN medicines m ON m.id = b.medicine_id
		   LEFT JOIN batch_receipts r ON r.batch_id = b.id`+where+`
		  GROUP BY b.id, b.medicine_id, m.kode, m.nama, m.satuan_dasar,
		           b.no_batch, b.tgl_kadaluarsa, b.status_batch
		  ORDER BY b.tgl_kadaluarsa ASC, m.nama ASC, b.id ASC
		  LIMIT ? OFFSET ?`,
		argsHalaman...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal mengambil daftar batch: %w", err)
	}
	defer rows.Close()

	hasil := make([]BatchStok, 0)
	for rows.Next() {
		var b BatchStok
		var kadaluarsa time.Time
		if err := rows.Scan(
			&b.BatchID, &b.MedicineID, &b.KodeObat, &b.NamaObat, &b.SatuanDasar,
			&b.NoBatch, &kadaluarsa, &b.StatusBatch, &b.TotalStok, &b.JumlahLapisan,
		); err != nil {
			return nil, 0, fmt.Errorf("gagal membaca daftar batch: %w", err)
		}
		b.TglKadaluarsa = kadaluarsa.Format(layoutTanggal)
		hasil = append(hasil, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("gagal membaca daftar batch: %w", err)
	}

	return hasil, total, nil
}

// GetBatch mengambil satu batch fisik beserta semua lapisan penerimaannya.
// Mengembalikan nil jika batch tidak ditemukan.
func (r *Repository) GetBatch(
	ctx context.Context,
	id int64,
) (*DetailBatch, error) {
	var d DetailBatch
	var kadaluarsa time.Time
	var alasan sql.NullString
	var tglBlokir sql.NullTime

	err := r.db.QueryRowContext(
		ctx,
		`SELECT b.id, b.medicine_id, m.kode, m.nama, m.satuan_dasar,
		        b.no_batch, b.tgl_kadaluarsa, b.status_batch,
		        b.alasan_blokir, b.tanggal_blokir
		   FROM batches b
		   JOIN medicines m ON m.id = b.medicine_id
		  WHERE b.id = ?`,
		id,
	).Scan(
		&d.BatchID, &d.MedicineID, &d.KodeObat, &d.NamaObat, &d.SatuanDasar,
		&d.NoBatch, &kadaluarsa, &d.StatusBatch, &alasan, &tglBlokir,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil batch: %w", err)
	}
	d.TglKadaluarsa = kadaluarsa.Format(layoutTanggal)
	if alasan.Valid {
		d.AlasanBlokir = alasan.String
	}
	if tglBlokir.Valid {
		d.TanggalBlokir = tglBlokir.Time.Format("2006-01-02 15:04:05")
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT r.id, r.purchase_id, p.no_faktur, p.tgl_faktur, s.nama_pbf,
		        r.tanggal_terima, r.qty_diterima, r.qty_sisa, r.harga_beli_per_dasar
		   FROM batch_receipts r
		   JOIN purchases p ON p.id = r.purchase_id
		   JOIN suppliers s ON s.id = p.supplier_id
		  WHERE r.batch_id = ?
		  ORDER BY r.tanggal_terima ASC, r.id ASC`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil lapisan batch: %w", err)
	}
	defer rows.Close()

	d.Lapisan = make([]LapisanBatch, 0)
	for rows.Next() {
		var l LapisanBatch
		var tglFaktur, terima time.Time
		if err := rows.Scan(
			&l.ReceiptID, &l.PurchaseID, &l.NoFaktur, &tglFaktur, &l.NamaPBF,
			&terima, &l.QtyDiterima, &l.QtySisa, &l.HargaBeliPerDasar,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca lapisan batch: %w", err)
		}
		l.TglFaktur = tglFaktur.Format(layoutTanggal)
		l.TanggalTerima = terima.Format("2006-01-02 15:04:05")
		d.TotalStok += l.QtySisa
		d.Lapisan = append(d.Lapisan, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca lapisan batch: %w", err)
	}
	d.JumlahLapisan = len(d.Lapisan)

	return &d, nil
}

// FindBatchByID mengunci dan mengambil satu batch (FOR UPDATE).
// Mengembalikan nil jika batch tidak ditemukan.
func (t *TxRepository) FindBatchByID(
	ctx context.Context,
	id int64,
) (*Batch, error) {
	var b Batch
	err := t.tx.QueryRowContext(
		ctx,
		`SELECT id, tgl_kadaluarsa, status_batch
		   FROM batches
		  WHERE id = ?
		    FOR UPDATE`,
		id,
	).Scan(&b.ID, &b.TglKadaluarsa, &b.Status)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil data batch: %w", err)
	}
	return &b, nil
}

// SetBatchDiblokir menandai batch sebagai diblokir beserta alasan dan waktunya.
func (t *TxRepository) SetBatchDiblokir(
	ctx context.Context,
	id int64,
	alasan string,
	waktu time.Time,
) error {
	_, err := t.tx.ExecContext(
		ctx,
		`UPDATE batches
		    SET status_batch = ?, alasan_blokir = ?, tanggal_blokir = ?
		  WHERE id = ?`,
		statusBatchDiblokir, alasan, waktu.Format("2006-01-02 15:04:05"), id,
	)
	if err != nil {
		return fmt.Errorf("gagal memblokir batch: %w", err)
	}
	return nil
}
