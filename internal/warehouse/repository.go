package warehouse

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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
) (int64, error) {
	res, err := t.tx.ExecContext(
		ctx,
		`INSERT INTO batch_receipts
		   (batch_id, purchase_id, qty_diterima, qty_sisa, harga_beli_per_dasar)
		 VALUES (?, ?, ?, ?, ?)`,
		batchID, purchaseID, qtyDasar, qtyDasar, hargaBeliPerDasar,
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
