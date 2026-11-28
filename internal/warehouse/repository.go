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
// Jika transaksi dibatalkan MySQL karena deadlock, fn dijalankan ulang dari
// awal (maksimal tiga kali), jadi fn tidak boleh punya efek di luar transaksi.
func (r *Repository) WithTx(
	ctx context.Context,
	fn func(tx *TxRepository) error,
) error {
	return ulangiJikaDeadlock(ctx, func() error {
		return r.jalankanTx(ctx, fn)
	})
}

// Jumlah percobaan maksimum jika transaksi dibatalkan karena deadlock.
const maksPercobaanTx = 3

// adalahDeadlock memeriksa apakah err adalah deadlock MySQL (kode 1213).
// MySQL membatalkan salah satu transaksi yang saling menunggu; transaksi itu
// aman diulang dari awal.
func adalahDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1213
}

// ulangiJikaDeadlock menjalankan jalankan() dan mengulanginya (maksimal
// maksPercobaanTx kali) hanya jika kegagalannya deadlock. Error lain
// dikembalikan apa adanya.
func ulangiJikaDeadlock(ctx context.Context, jalankan func() error) error {
	var err error
	for percobaan := 1; percobaan <= maksPercobaanTx; percobaan++ {
		err = jalankan()
		if !adalahDeadlock(err) || percobaan == maksPercobaanTx {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(time.Duration(percobaan) * 20 * time.Millisecond):
		}
	}
	return err
}

// jalankanTx menjalankan satu kali transaksi (tanpa pengulangan).
func (r *Repository) jalankanTx(
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
	case statusJualBermasalah:
		// Diblokir, atau kedaluwarsa/mendekati kedaluwarsa (<= batas).
		where += " AND (b.status_batch = ? OR b.tgl_kadaluarsa <= ?)"
		args = append(args, statusBatchDiblokir, q.Batas)
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

// ---------- Tindak lanjut stok ----------

// FindReceiptForUpdate mengunci dan mengambil satu lapisan penerimaan (FOR UPDATE).
// Mengembalikan nil jika lapisan tidak ditemukan.
func (t *TxRepository) FindReceiptForUpdate(
	ctx context.Context,
	id int64,
) (*LapisanStok, error) {
	var l LapisanStok
	err := t.tx.QueryRowContext(
		ctx,
		`SELECT id, batch_id, qty_sisa, harga_beli_per_dasar
		   FROM batch_receipts
		  WHERE id = ?
		    FOR UPDATE`,
		id,
	).Scan(&l.ID, &l.BatchID, &l.QtySisa, &l.HargaBeliPerDasar)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil lapisan penerimaan: %w", err)
	}
	return &l, nil
}

// KurangiStokLapisan mengurangi qty_sisa sebuah lapisan.
// Penjaga "qty_sisa >= ?" memastikan stok tidak pernah negatif: jika sisa tidak
// cukup, tidak ada baris yang berubah dan ErrStokTidakCukup dikembalikan.
func (t *TxRepository) KurangiStokLapisan(
	ctx context.Context,
	id int64,
	qty float64,
) error {
	res, err := t.tx.ExecContext(
		ctx,
		`UPDATE batch_receipts
		    SET qty_sisa = qty_sisa - ?
		  WHERE id = ? AND qty_sisa >= ?`,
		qty, id, qty,
	)
	if err != nil {
		return fmt.Errorf("gagal mengurangi stok lapisan: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa pengurangan stok: %w", err)
	}
	if n != 1 {
		return ErrStokTidakCukup
	}
	return nil
}

// InsertDisposal menyimpan satu baris tindak lanjut.
func (t *TxRepository) InsertDisposal(
	ctx context.Context,
	d DisposalBaru,
) (int64, error) {
	var catatan sql.NullString
	if d.Catatan != "" {
		catatan = sql.NullString{String: d.Catatan, Valid: true}
	}

	res, err := t.tx.ExecContext(
		ctx,
		`INSERT INTO batch_disposals
		   (batch_receipt_id, user_id, jenis, sumber_tindak_lanjut, alasan,
		    qty_dasar, nilai_kompensasi, catatan, tanggal)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ReceiptID, d.UserID, d.Jenis, d.Sumber, d.Alasan,
		d.Qty, d.Kompensasi, catatan, d.Waktu.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan tindak lanjut: %w", err)
	}
	return res.LastInsertId()
}

// ListTindakLanjut mengambil satu halaman riwayat tindak lanjut.
// Urutan: terbaru dulu.
func (r *Repository) ListTindakLanjut(
	ctx context.Context,
	q QueryTindakLanjut,
) ([]RiwayatTindakLanjut, int, error) {
	const dari = `
	  FROM batch_disposals d
	  JOIN batch_receipts r ON r.id = d.batch_receipt_id
	  JOIN batches b ON b.id = r.batch_id
	  JOIN medicines m ON m.id = b.medicine_id
	  JOIN users u ON u.id = d.user_id`

	where := " WHERE 1=1"
	var args []any

	if q.Jenis != "" {
		where += " AND d.jenis = ?"
		args = append(args, q.Jenis)
	}
	if q.BatchID > 0 {
		where += " AND b.id = ?"
		args = append(args, q.BatchID)
	}
	if q.Dari != "" {
		where += " AND DATE(d.tanggal) >= ?"
		args = append(args, q.Dari)
	}
	if q.Sampai != "" {
		where += " AND DATE(d.tanggal) <= ?"
		args = append(args, q.Sampai)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+dari+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung riwayat tindak lanjut: %w", err)
	}

	argsHalaman := append(append([]any{}, args...), q.Limit, q.Offset)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT d.id, d.batch_receipt_id, b.id, b.medicine_id, m.kode, m.nama,
		        m.satuan_dasar, b.no_batch, d.jenis, d.sumber_tindak_lanjut,
		        d.alasan, d.qty_dasar, d.nilai_kompensasi, r.harga_beli_per_dasar,
		        d.catatan, u.nama, d.tanggal`+dari+where+`
		  ORDER BY d.tanggal DESC, d.id DESC
		  LIMIT ? OFFSET ?`,
		argsHalaman...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal mengambil riwayat tindak lanjut: %w", err)
	}
	defer rows.Close()

	hasil := make([]RiwayatTindakLanjut, 0)
	for rows.Next() {
		var it RiwayatTindakLanjut
		var harga float64
		var catatan sql.NullString
		var tanggal time.Time
		if err := rows.Scan(
			&it.DisposalID, &it.BatchReceiptID, &it.BatchID, &it.MedicineID,
			&it.KodeObat, &it.NamaObat, &it.SatuanDasar, &it.NoBatch, &it.Jenis,
			&it.SumberTindakLanjut, &it.Alasan, &it.QtyDasar, &it.NilaiKompensasi,
			&harga, &catatan, &it.DicatatOleh, &tanggal,
		); err != nil {
			return nil, 0, fmt.Errorf("gagal membaca riwayat tindak lanjut: %w", err)
		}
		if catatan.Valid {
			it.Catatan = catatan.String
		}
		it.Tanggal = tanggal.Format("2006-01-02 15:04:05")
		it.NilaiPerolehan = bulatkan(harga*it.QtyDasar, 2)
		it.NilaiKerugian = bulatkan(it.NilaiPerolehan-it.NilaiKompensasi, 2)
		hasil = append(hasil, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("gagal membaca riwayat tindak lanjut: %w", err)
	}

	return hasil, total, nil
}

// ---------- Stock opname ----------

// BatchAda memeriksa apakah batch dengan id tersebut ada (tanpa mengunci).
func (t *TxRepository) BatchAda(
	ctx context.Context,
	id int64,
) (bool, error) {
	var n int
	err := t.tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM batches WHERE id = ?`,
		id,
	).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("gagal memeriksa batch: %w", err)
	}
	return n > 0, nil
}

// LockLapisanBatch mengunci (FOR UPDATE) dan mengambil SEMUA lapisan penerimaan
// sebuah batch, diurutkan dari yang paling dahulu diterima.
//
// Bentuk query (JOIN batches ... FOR UPDATE) sengaja sama dengan query lapisan
// milik modul Kasir, supaya urutan penguncian baris kedua modul sama dan
// peluang deadlock kecil.
func (t *TxRepository) LockLapisanBatch(
	ctx context.Context,
	batchID int64,
) ([]LapisanStok, error) {
	rows, err := t.tx.QueryContext(
		ctx,
		`SELECT r.id, r.batch_id, r.qty_sisa, r.harga_beli_per_dasar
		   FROM batch_receipts r
		   JOIN batches b ON b.id = r.batch_id
		  WHERE r.batch_id = ?
		  ORDER BY r.tanggal_terima ASC, r.id ASC
		    FOR UPDATE`,
		batchID,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengunci lapisan batch: %w", err)
	}
	defer rows.Close()

	var hasil []LapisanStok
	for rows.Next() {
		var l LapisanStok
		if err := rows.Scan(&l.ID, &l.BatchID, &l.QtySisa, &l.HargaBeliPerDasar); err != nil {
			return nil, fmt.Errorf("gagal membaca lapisan batch: %w", err)
		}
		hasil = append(hasil, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca lapisan batch: %w", err)
	}
	return hasil, nil
}

// TambahStokLapisan menambah qty_sisa sebuah lapisan.
func (t *TxRepository) TambahStokLapisan(
	ctx context.Context,
	id int64,
	qty float64,
) error {
	res, err := t.tx.ExecContext(
		ctx,
		`UPDATE batch_receipts SET qty_sisa = qty_sisa + ? WHERE id = ?`,
		qty, id,
	)
	if err != nil {
		return fmt.Errorf("gagal menambah stok lapisan: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa penambahan stok: %w", err)
	}
	if n != 1 {
		return fmt.Errorf("lapisan penerimaan %d tidak ditemukan saat menambah stok", id)
	}
	return nil
}

// InsertStockOpname menyimpan header hasil opname.
func (t *TxRepository) InsertStockOpname(
	ctx context.Context,
	o OpnameBaru,
) (int64, error) {
	var catatan sql.NullString
	if o.Catatan != "" {
		catatan = sql.NullString{String: o.Catatan, Valid: true}
	}

	res, err := t.tx.ExecContext(
		ctx,
		`INSERT INTO stock_opnames
		   (batch_id, user_id, qty_sistem, qty_fisik, catatan, tanggal)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		o.BatchID, o.UserID, o.QtySistem, o.QtyFisik, catatan,
		o.Waktu.Format("2006-01-02 15:04:05"),
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan stock opname: %w", err)
	}
	return res.LastInsertId()
}

// InsertOpnameDetail menyimpan penyesuaian satu lapisan.
func (t *TxRepository) InsertOpnameDetail(
	ctx context.Context,
	opnameID, receiptID int64,
	penyesuaian, sebelum, sesudah float64,
) error {
	_, err := t.tx.ExecContext(
		ctx,
		`INSERT INTO stock_opname_details
		   (stock_opname_id, batch_receipt_id, qty_penyesuaian, qty_sisa_sebelum, qty_sisa_sesudah)
		 VALUES (?, ?, ?, ?, ?)`,
		opnameID, receiptID, penyesuaian, sebelum, sesudah,
	)
	if err != nil {
		return fmt.Errorf("gagal menyimpan detail stock opname: %w", err)
	}
	return nil
}

// GetStockOpname mengambil satu hasil opname beserta penyesuaiannya.
// Mengembalikan nil jika tidak ditemukan. Field turunan (selisih, status,
// nilai) diisi oleh service.
func (r *Repository) GetStockOpname(
	ctx context.Context,
	id int64,
) (*StockOpname, error) {
	var o StockOpname
	var kadaluarsa, tanggal time.Time
	var catatan sql.NullString

	err := r.db.QueryRowContext(
		ctx,
		`SELECT so.id, so.batch_id, b.medicine_id, m.kode, m.nama, m.satuan_dasar,
		        b.no_batch, b.tgl_kadaluarsa, so.qty_sistem, so.qty_fisik,
		        so.catatan, u.nama, so.tanggal
		   FROM stock_opnames so
		   JOIN batches b ON b.id = so.batch_id
		   JOIN medicines m ON m.id = b.medicine_id
		   JOIN users u ON u.id = so.user_id
		  WHERE so.id = ?`,
		id,
	).Scan(
		&o.OpnameID, &o.BatchID, &o.MedicineID, &o.KodeObat, &o.NamaObat,
		&o.SatuanDasar, &o.NoBatch, &kadaluarsa, &o.QtySistem, &o.QtyFisik,
		&catatan, &o.DicatatOleh, &tanggal,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil stock opname: %w", err)
	}
	o.TglKadaluarsa = kadaluarsa.Format(layoutTanggal)
	o.Tanggal = tanggal.Format("2006-01-02 15:04:05")
	if catatan.Valid {
		o.Catatan = catatan.String
	}

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT d.batch_receipt_id, p.no_faktur, r.tanggal_terima,
		        r.harga_beli_per_dasar, d.qty_penyesuaian,
		        d.qty_sisa_sebelum, d.qty_sisa_sesudah
		   FROM stock_opname_details d
		   JOIN batch_receipts r ON r.id = d.batch_receipt_id
		   JOIN purchases p ON p.id = r.purchase_id
		  WHERE d.stock_opname_id = ?
		  ORDER BY d.id`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil detail stock opname: %w", err)
	}
	defer rows.Close()

	o.Penyesuaian = make([]PenyesuaianLapisan, 0)
	for rows.Next() {
		var p PenyesuaianLapisan
		var terima time.Time
		if err := rows.Scan(
			&p.ReceiptID, &p.NoFaktur, &terima, &p.HargaBeliPerDasar,
			&p.QtyPenyesuaian, &p.QtySisaSebelum, &p.QtySisaSesudah,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca detail stock opname: %w", err)
		}
		p.TanggalTerima = terima.Format("2006-01-02 15:04:05")
		o.Penyesuaian = append(o.Penyesuaian, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca detail stock opname: %w", err)
	}

	return &o, nil
}

// ListStockOpname mengambil satu halaman riwayat opname, terbaru dulu.
func (r *Repository) ListStockOpname(
	ctx context.Context,
	q QueryStockOpname,
) ([]RiwayatStockOpname, int, error) {
	const dari = `
	  FROM stock_opnames so
	  JOIN batches b ON b.id = so.batch_id
	  JOIN medicines m ON m.id = b.medicine_id
	  JOIN users u ON u.id = so.user_id`

	where := " WHERE 1=1"
	var args []any

	if q.BatchID > 0 {
		where += " AND so.batch_id = ?"
		args = append(args, q.BatchID)
	}
	if q.Dari != "" {
		where += " AND DATE(so.tanggal) >= ?"
		args = append(args, q.Dari)
	}
	if q.Sampai != "" {
		where += " AND DATE(so.tanggal) <= ?"
		args = append(args, q.Sampai)
	}
	if q.HanyaSelisih {
		where += " AND so.qty_fisik <> so.qty_sistem"
	}

	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*)`+dari+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("gagal menghitung riwayat stock opname: %w", err)
	}

	argsHalaman := append(append([]any{}, args...), q.Limit, q.Offset)

	rows, err := r.db.QueryContext(
		ctx,
		`SELECT so.id, so.batch_id, b.medicine_id, m.kode, m.nama, m.satuan_dasar,
		        b.no_batch, so.qty_sistem, so.qty_fisik,
		        (SELECT COUNT(*) FROM stock_opname_details d WHERE d.stock_opname_id = so.id),
		        so.catatan, u.nama, so.tanggal`+dari+where+`
		  ORDER BY so.tanggal DESC, so.id DESC
		  LIMIT ? OFFSET ?`,
		argsHalaman...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("gagal mengambil riwayat stock opname: %w", err)
	}
	defer rows.Close()

	hasil := make([]RiwayatStockOpname, 0)
	for rows.Next() {
		var it RiwayatStockOpname
		var catatan sql.NullString
		var tanggal time.Time
		if err := rows.Scan(
			&it.OpnameID, &it.BatchID, &it.MedicineID, &it.KodeObat, &it.NamaObat,
			&it.SatuanDasar, &it.NoBatch, &it.QtySistem, &it.QtyFisik,
			&it.JumlahLapisan, &catatan, &it.DicatatOleh, &tanggal,
		); err != nil {
			return nil, 0, fmt.Errorf("gagal membaca riwayat stock opname: %w", err)
		}
		if catatan.Valid {
			it.Catatan = catatan.String
		}
		it.Tanggal = tanggal.Format("2006-01-02 15:04:05")
		hasil = append(hasil, it)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("gagal membaca riwayat stock opname: %w", err)
	}

	return hasil, total, nil
}

// ---------- Pilihan form penerimaan ----------

// ListOpsiPenerimaan mengambil daftar PBF, serta obat aktif beserta satuannya.
func (r *Repository) ListOpsiPenerimaan(ctx context.Context) (*OpsiPenerimaan, error) {
	hasil := &OpsiPenerimaan{PBF: []PBFOpsi{}, Obat: []ObatOpsi{}}

	pbf, err := r.db.QueryContext(ctx, `SELECT id, nama_pbf FROM suppliers ORDER BY nama_pbf ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar PBF: %w", err)
	}
	defer pbf.Close()
	for pbf.Next() {
		var p PBFOpsi
		if err := pbf.Scan(&p.ID, &p.NamaPBF); err != nil {
			return nil, fmt.Errorf("gagal membaca daftar PBF: %w", err)
		}
		hasil.PBF = append(hasil.PBF, p)
	}
	if err := pbf.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca daftar PBF: %w", err)
	}

	obat, err := r.db.QueryContext(
		ctx,
		`SELECT id, kode, nama, satuan_dasar
		   FROM medicines
		  WHERE is_active = 1
		  ORDER BY nama ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar obat: %w", err)
	}
	defer obat.Close()

	indeks := map[int64]int{}
	for obat.Next() {
		var o ObatOpsi
		if err := obat.Scan(&o.ID, &o.Kode, &o.Nama, &o.SatuanDasar); err != nil {
			return nil, fmt.Errorf("gagal membaca daftar obat: %w", err)
		}
		o.Satuan = []SatuanOpsi{}
		indeks[o.ID] = len(hasil.Obat)
		hasil.Obat = append(hasil.Obat, o)
	}
	if err := obat.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca daftar obat: %w", err)
	}

	satuan, err := r.db.QueryContext(
		ctx,
		`SELECT id, medicine_id, nama_satuan, isi_per_satuan_dasar
		   FROM medicine_units
		  WHERE is_active = 1
		  ORDER BY medicine_id ASC, isi_per_satuan_dasar DESC, nama_satuan ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil daftar satuan: %w", err)
	}
	defer satuan.Close()
	for satuan.Next() {
		var s SatuanOpsi
		var medicineID int64
		if err := satuan.Scan(&s.ID, &medicineID, &s.NamaSatuan, &s.IsiPerSatuanDasar); err != nil {
			return nil, fmt.Errorf("gagal membaca daftar satuan: %w", err)
		}
		if i, ada := indeks[medicineID]; ada {
			hasil.Obat[i].Satuan = append(hasil.Obat[i].Satuan, s)
		}
	}
	if err := satuan.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca daftar satuan: %w", err)
	}

	return hasil, nil
}
