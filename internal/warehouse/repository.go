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
