package cashier

import (
	"context"
	"fmt"
	"time"
)

const layoutTanggalSQL = "2006-01-02"

const kondisiBolehDijual = `
	b.status_batch = 'AKTIF'
	AND br.qty_sisa > 0
	AND b.tgl_kadaluarsa > DATE_ADD(?, INTERVAL 1 MONTH)`

// StokKasir membaca dan mengubah stok untuk kebutuhan penjualan dan retur.
type StokKasir struct{}

// StokTersedia menjumlahkan stok yang boleh dijual untuk satu obat (satuan dasar).
func (StokKasir) StokTersedia(
	ctx context.Context, q DBTX, medicineID int64, hariIni time.Time,
) (float64, error) {
	var total float64
	err := q.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(br.qty_sisa), 0)
		  FROM batch_receipts br
		  JOIN batches b ON b.id = br.batch_id
		 WHERE b.medicine_id = ?
		   AND `+kondisiBolehDijual,
		medicineID, hariIni.Format(layoutTanggalSQL),
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("gagal menghitung stok: %w", err)
	}
	return bulatkan(total, 3), nil
}

// LapisanTersedia mengembalikan lapisan yang boleh dijual untuk satu obat,
// terurut FEFO. Jika kunci = true, baris dikunci (FOR UPDATE) sampai transaksi
// selesai, jadi harus dipanggil di dalam transaksi.
func (StokKasir) LapisanTersedia(
	ctx context.Context, q DBTX, medicineID int64, hariIni time.Time, kunci bool,
) ([]Lapisan, error) {
	query := `
		SELECT br.id, b.tgl_kadaluarsa, br.tanggal_terima,
		       br.qty_sisa, br.harga_beli_per_dasar
		  FROM batch_receipts br
		  JOIN batches b ON b.id = br.batch_id
		 WHERE b.medicine_id = ?
		   AND ` + kondisiBolehDijual + `
		 ORDER BY b.tgl_kadaluarsa, br.tanggal_terima, br.id`
	if kunci {
		query += ` FOR UPDATE`
	}

	rows, err := q.QueryContext(ctx, query, medicineID, hariIni.Format(layoutTanggalSQL))
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil lapisan stok: %w", err)
	}
	defer rows.Close()

	hasil := make([]Lapisan, 0)
	for rows.Next() {
		var l Lapisan
		if err := rows.Scan(
			&l.ReceiptID, &l.TglKadaluarsa, &l.TanggalTerima,
			&l.QtySisa, &l.HargaBeliPerDasar,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca lapisan stok: %w", err)
		}
		hasil = append(hasil, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca lapisan stok: %w", err)
	}
	return hasil, nil
}

// KurangiStok mengurangi qty_sisa pada satu lapisan penerimaan.
// Pengurangan bersifat atomik: baris hanya berubah jika qty_sisa >= qty,
// jadi stok tidak pernah menjadi negatif walau ada bug di tempat lain.
func (StokKasir) KurangiStok(ctx context.Context, q DBTX, receiptID int64, qty float64) error {
	qty = bulatkan(qty, 3)
	if qty <= 0 {
		return validasi("jumlah pengurangan stok harus lebih dari 0")
	}

	res, err := q.ExecContext(ctx, `
		UPDATE batch_receipts
		   SET qty_sisa = qty_sisa - ?
		 WHERE id = ? AND qty_sisa >= ?`,
		qty, receiptID, qty,
	)
	if err != nil {
		return fmt.Errorf("gagal mengurangi stok: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa pengurangan stok: %w", err)
	}
	if n == 0 {
		return ErrStokTidakCukup
	}
	return nil
}

// TambahStok menambah qty_sisa pada satu lapisan penerimaan, untuk retur
// konsumen yang dinyatakan LAYAK_JUAL. Batas jumlahnya dijaga oleh validasi
// retur kumulatif di service (langkah 9), bukan di sini.
func (StokKasir) TambahStok(ctx context.Context, q DBTX, receiptID int64, qty float64) error {
	qty = bulatkan(qty, 3)
	if qty <= 0 {
		return validasi("jumlah penambahan stok harus lebih dari 0")
	}

	res, err := q.ExecContext(ctx, `
		UPDATE batch_receipts
		   SET qty_sisa = qty_sisa + ?
		 WHERE id = ?`,
		qty, receiptID,
	)
	if err != nil {
		return fmt.Errorf("gagal menambah stok: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("gagal memeriksa penambahan stok: %w", err)
	}
	if n == 0 {
		return ErrTidakDitemukan
	}
	return nil
}