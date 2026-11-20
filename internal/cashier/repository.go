package cashier

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Repository menjadi tempat query SQL untuk modul kasir.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// escapeLike mencegah karakter % dan _ dari pengguna dianggap wildcard.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// CariObat mencari obat aktif berdasarkan nama atau kode,
// beserta satuan jual yang aktif. Stok belum diisi di sini.
func (r *Repository) CariObat(ctx context.Context, kata string, limit int) ([]Obat, error) {
	pola := "%" + escapeLike(kata) + "%"

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, kode, nama, satuan_dasar
		   FROM medicines
		  WHERE is_active = 1 AND (nama LIKE ? OR kode LIKE ?)
		  ORDER BY nama
		  LIMIT ?`,
		pola, pola, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mencari obat: %w", err)
	}

	hasil := make([]Obat, 0)
	indeks := make(map[int64]int)
	ids := make([]any, 0)
	for rows.Next() {
		var o Obat
		if err := rows.Scan(&o.ID, &o.Kode, &o.Nama, &o.SatuanDasar); err != nil {
			rows.Close()
			return nil, fmt.Errorf("gagal membaca obat: %w", err)
		}
		o.Satuan = make([]SatuanJual, 0)
		indeks[o.ID] = len(hasil)
		hasil = append(hasil, o)
		ids = append(ids, o.ID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("gagal membaca obat: %w", err)
	}
	rows.Close()

	if len(hasil) == 0 {
		return hasil, nil
	}

	// Tanda tanya dibuat dari jumlah id; nilainya tetap lewat parameter.
	ph := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	urows, err := r.db.QueryContext(ctx,
		`SELECT id, medicine_id, nama_satuan, isi_per_satuan_dasar, harga_jual
		   FROM medicine_units
		  WHERE is_active = 1 AND medicine_id IN (`+ph+`)
		  ORDER BY medicine_id, isi_per_satuan_dasar, id`,
		ids...,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil satuan obat: %w", err)
	}
	defer urows.Close()

	for urows.Next() {
		var s SatuanJual
		var medicineID int64
		if err := urows.Scan(&s.ID, &medicineID, &s.Nama, &s.IsiPerSatuanDasar, &s.HargaJual); err != nil {
			return nil, fmt.Errorf("gagal membaca satuan obat: %w", err)
		}
		i := indeks[medicineID]
		hasil[i].Satuan = append(hasil[i].Satuan, s)
	}
	if err := urows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca satuan obat: %w", err)
	}
	return hasil, nil
}

// DBTX dipenuhi oleh *sql.DB maupun *sql.Tx. Fungsi stok (stok_kasir.go)
// memakainya supaya bisa berjalan di dalam transaksi penjualan yang sama.
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// TxRepository menjalankan query di dalam satu transaksi database.
type TxRepository struct {
	tx *sql.Tx
}

// DB mengembalikan koneksi transaksi untuk diberikan ke fungsi stok.
func (t *TxRepository) DB() DBTX { return t.tx }

// WithTx menjalankan fn di dalam satu transaksi.
// Jika fn mengembalikan error atau panic, semua perubahan di-rollback.
func (r *Repository) WithTx(ctx context.Context, fn func(tx *TxRepository) error) error {
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

// FindUnit mengembalikan nil jika satuan atau obatnya tidak ada / tidak aktif.
func (t *TxRepository) FindUnit(ctx context.Context, unitID int64) (*UnitObat, error) {
	var u UnitObat
	err := t.tx.QueryRowContext(ctx,
		`SELECT u.id, u.medicine_id, m.nama, u.nama_satuan,
		        u.isi_per_satuan_dasar, u.harga_jual
		   FROM medicine_units u
		   JOIN medicines m ON m.id = u.medicine_id
		  WHERE u.id = ? AND u.is_active = 1 AND m.is_active = 1`,
		unitID,
	).Scan(&u.ID, &u.MedicineID, &u.NamaObat, &u.Nama, &u.IsiPerSatuanDasar, &u.HargaJual)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil satuan obat: %w", err)
	}
	return &u, nil
}

func (t *TxRepository) InsertSale(
	ctx context.Context,
	userID int64, tanggal time.Time, total float64, metode string, jumlahBayar float64,
) (int64, error) {
	res, err := t.tx.ExecContext(ctx,
		`INSERT INTO sales (user_id, tanggal, total, metode_bayar, jumlah_bayar)
		 VALUES (?, ?, ?, ?, ?)`,
		userID, tanggal, total, metode, jumlahBayar,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan penjualan: %w", err)
	}
	return res.LastInsertId()
}

func (t *TxRepository) InsertSaleItem(
	ctx context.Context,
	saleID, unitID int64, qtyInput, qtyDasar, hargaJualSnapshot float64,
) (int64, error) {
	res, err := t.tx.ExecContext(ctx,
		`INSERT INTO sale_items (sale_id, unit_id, qty_input, qty_dasar, harga_jual_snapshot)
		 VALUES (?, ?, ?, ?, ?)`,
		saleID, unitID, qtyInput, qtyDasar, hargaJualSnapshot,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan rincian penjualan: %w", err)
	}
	return res.LastInsertId()
}

func (t *TxRepository) InsertSaleItemBatch(
	ctx context.Context,
	saleItemID, receiptID int64, qtyDasar, hargaBeliSnapshot float64,
) (int64, error) {
	res, err := t.tx.ExecContext(ctx,
		`INSERT INTO sale_item_batches (sale_item_id, batch_receipt_id, qty_dasar, harga_beli_snapshot)
		 VALUES (?, ?, ?, ?)`,
		saleItemID, receiptID, qtyDasar, hargaBeliSnapshot,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan alokasi batch: %w", err)
	}
	return res.LastInsertId()
}