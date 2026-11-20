package cashier

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// AlokasiJual adalah satu baris sale_item_batches beserta harga jual per satuan dasar dari rincian penjualannya.
type AlokasiJual struct {
	ID                int64
	ReceiptID         int64
	QtyDasar          float64
	HargaJualPerDasar float64
}

// ReturRow adalah data retur yang dibutuhkan untuk menetapkan kelayakan.
type ReturRow struct {
	ID        int64
	ReceiptID int64
	QtyDasar  float64
	Status    string
}

// ReturBaru adalah data retur yang akan disimpan.
type ReturBaru struct {
	SaleItemBatchID int64
	UserID          int64
	Jenis           string
	QtyDasar        float64
	Alasan          string
	Kondisi         string
	Status          string
	NilaiRefund     float64
	Tanggal         time.Time
}

// nullIfEmpty menyimpan teks kosong sebagai NULL di database.
func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// AmbilPenjualanRetur mengambil transaksi asal beserta alokasi batch dan total retur sebelumnya.
// Mengembalikan ErrTidakDitemukan jika tidak ada.
func (r *Repository) AmbilPenjualanRetur(ctx context.Context, saleID int64) (*PenjualanRetur, error) {
	var p PenjualanRetur
	err := r.db.QueryRowContext(ctx,
		`SELECT s.id, s.tanggal, s.total, u.nama
		   FROM sales s
		   JOIN users u ON u.id = s.user_id
		  WHERE s.id = ?`,
		saleID,
	).Scan(&p.SaleID, &p.Tanggal, &p.Total, &p.Kasir)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakDitemukan
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil penjualan: %w", err)
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT si.id, m.nama, un.nama_satuan, m.satuan_dasar,
		        si.qty_input, si.harga_jual_snapshot,
		        sib.id, b.no_batch, b.tgl_kadaluarsa, sib.qty_dasar,
		        COALESCE((SELECT SUM(rt.qty_dasar)
		                    FROM returns rt
		                   WHERE rt.sale_item_batch_id = sib.id), 0)
		   FROM sale_items si
		   JOIN medicine_units un ON un.id = si.unit_id
		   JOIN medicines m ON m.id = un.medicine_id
		   JOIN sale_item_batches sib ON sib.sale_item_id = si.id
		   JOIN batch_receipts br ON br.id = sib.batch_receipt_id
		   JOIN batches b ON b.id = br.batch_id
		  WHERE si.sale_id = ?
		  ORDER BY si.id, sib.id`,
		saleID,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil rincian penjualan: %w", err)
	}
	defer rows.Close()

	p.Items = make([]ReturItem, 0)
	var itemTerakhir int64
	for rows.Next() {
		var (
			itemID                  int64
			nama, satuan, dasar     string
			qtyInput, hargaPerDasar float64
			b                       ReturBatch
		)
		if err := rows.Scan(
			&itemID, &nama, &satuan, &dasar, &qtyInput, &hargaPerDasar,
			&b.SaleItemBatchID, &b.NoBatch, &b.TglKadaluarsa, &b.QtyDasar, &b.SudahDiretur,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca rincian penjualan: %w", err)
		}

		// Baris diurutkan per rincian, jadi item baru dibuat saat id-nya berganti.
		if len(p.Items) == 0 || itemID != itemTerakhir {
			p.Items = append(p.Items, ReturItem{
				NamaObat:          nama,
				NamaSatuan:        satuan,
				QtyDibeli:         qtyInput,
				SatuanDasar:       dasar,
				HargaJualPerDasar: hargaPerDasar,
				Batches:           make([]ReturBatch, 0),
			})
			itemTerakhir = itemID
		}
		item := &p.Items[len(p.Items)-1]
		item.Batches = append(item.Batches, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca rincian penjualan: %w", err)
	}
	return &p, nil
}

// KunciAlokasi membaca dan mengunci satu alokasi penjualan (FOR UPDATE).
// Dua retur yang bersamaan untuk alokasi yang sama jadi antre di sini.
// Mengembalikan nil jika alokasi tidak ada.
func (t *TxRepository) KunciAlokasi(ctx context.Context, id int64) (*AlokasiJual, error) {
	var a AlokasiJual
	err := t.tx.QueryRowContext(ctx,
		`SELECT sib.id, sib.batch_receipt_id, sib.qty_dasar, si.harga_jual_snapshot
		   FROM sale_item_batches sib
		   JOIN sale_items si ON si.id = sib.sale_item_id
		  WHERE sib.id = ?
		    FOR UPDATE`,
		id,
	).Scan(&a.ID, &a.ReceiptID, &a.QtyDasar, &a.HargaJualPerDasar)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengunci alokasi penjualan: %w", err)
	}
	return &a, nil
}

// TotalRetur menjumlahkan semua retur (apa pun statusnya) pada satu alokasi.
// FOR UPDATE memastikan yang dibaca adalah data terbaru, bukan snapshot lama.
func (t *TxRepository) TotalRetur(ctx context.Context, saleItemBatchID int64) (float64, error) {
	var total float64
	err := t.tx.QueryRowContext(ctx,
		`SELECT COALESCE(SUM(qty_dasar), 0)
		   FROM returns
		  WHERE sale_item_batch_id = ?
		    FOR UPDATE`,
		saleItemBatchID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung total retur: %w", err)
	}
	return total, nil
}

func (t *TxRepository) InsertRetur(ctx context.Context, r ReturBaru) (int64, error) {
	res, err := t.tx.ExecContext(ctx,
		`INSERT INTO returns
		    (sale_item_batch_id, user_id, jenis_pengembalian, qty_dasar,
		     alasan, kondisi_obat, status_kelayakan, nilai_refund, tanggal)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.SaleItemBatchID, r.UserID, r.Jenis, r.QtyDasar,
		nullIfEmpty(r.Alasan), nullIfEmpty(r.Kondisi), r.Status, r.NilaiRefund, r.Tanggal,
	)
	if err != nil {
		return 0, fmt.Errorf("gagal menyimpan retur: %w", err)
	}
	return res.LastInsertId()
}

// KunciRetur membaca dan mengunci satu retur beserta lapisan penerimaan asalnya.
// Mengembalikan nil jika retur tidak ada.
func (t *TxRepository) KunciRetur(ctx context.Context, id int64) (*ReturRow, error) {
	var r ReturRow
	err := t.tx.QueryRowContext(ctx,
		`SELECT rt.id, sib.batch_receipt_id, rt.qty_dasar, rt.status_kelayakan
		   FROM returns rt
		   JOIN sale_item_batches sib ON sib.id = rt.sale_item_batch_id
		  WHERE rt.id = ?
		    FOR UPDATE`,
		id,
	).Scan(&r.ID, &r.ReceiptID, &r.QtyDasar, &r.Status)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("gagal mengunci retur: %w", err)
	}
	return &r, nil
}

// UbahKelayakan mengubah status dari MENUNGGU_PEMERIKSAAN. Syarat status lama
// ikut diperiksa di SQL sebagai pengaman kedua terhadap perubahan ganda.
func (t *TxRepository) UbahKelayakan(ctx context.Context, id int64, statusBaru string) error {
	res, err := t.tx.ExecContext(ctx,
		`UPDATE returns
		    SET status_kelayakan = ?
		  WHERE id = ? AND status_kelayakan = 'MENUNGGU_PEMERIKSAAN'`,
		statusBaru, id,
	)
	if err != nil {
		return fmt.Errorf("gagal mengubah status kelayakan: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa perubahan status: %w", err)
	}
	if n == 0 {
		return ErrStatusSudahFinal
	}
	return nil
}