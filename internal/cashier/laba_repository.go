package cashier

import (
	"context"
	"fmt"
	"time"
)

// sqlLabaPerTransaksi menghitung komponen laba per transaksi. 
// Setiap komponen memakai subquery sendiri supaya penjumlahan tidak terkali oleh join.
// Retur dihitung pada transaksi asalnya.
const sqlLabaPerTransaksi = `
	SELECT s.id, s.tanggal, u.nama, s.metode_bayar, s.total,
	       COALESCE((SELECT SUM(sib.qty_dasar * sib.harga_beli_snapshot)
	                   FROM sale_items si
	                   JOIN sale_item_batches sib ON sib.sale_item_id = si.id
	                  WHERE si.sale_id = s.id), 0) AS hpp,
	       COALESCE((SELECT SUM(rt.nilai_refund)
	                   FROM sale_items si
	                   JOIN sale_item_batches sib ON sib.sale_item_id = si.id
	                   JOIN returns rt ON rt.sale_item_batch_id = sib.id
	                  WHERE si.sale_id = s.id), 0) AS retur_refund,
	       COALESCE((SELECT SUM(rt.qty_dasar * sib.harga_beli_snapshot)
	                   FROM sale_items si
	                   JOIN sale_item_batches sib ON sib.sale_item_id = si.id
	                   JOIN returns rt ON rt.sale_item_batch_id = sib.id
	                  WHERE si.sale_id = s.id), 0) AS retur_hpp,
	       COALESCE((SELECT SUM(rt.qty_dasar * sib.harga_beli_snapshot)
	                   FROM sale_items si
	                   JOIN sale_item_batches sib ON sib.sale_item_id = si.id
	                   JOIN returns rt ON rt.sale_item_batch_id = sib.id
	                  WHERE si.sale_id = s.id
	                    AND rt.status_kelayakan = 'TIDAK_LAYAK_JUAL'), 0) AS kerugian_retur
	  FROM sales s
	  JOIN users u ON u.id = s.user_id
	 WHERE s.tanggal >= ? AND s.tanggal < ?`

// sqlRingkasanLaba menjumlahkan hasil query per transaksi di atas.
const sqlRingkasanLaba = `
	SELECT COUNT(*),
	       COALESCE(SUM(t.total), 0),
	       COALESCE(SUM(t.hpp), 0),
	       COALESCE(SUM(t.retur_refund), 0),
	       COALESCE(SUM(t.retur_hpp), 0),
	       COALESCE(SUM(t.kerugian_retur), 0)
	  FROM (` + sqlLabaPerTransaksi + `) t`

// LabaPerTransaksi mengambil komponen laba mentah per transaksi pada rentang [awal, akhir).
// Nilai turunan dihitung di service.
func (r *Repository) LabaPerTransaksi(
	ctx context.Context, awal, akhir time.Time, limit, offset int,
) ([]LabaTransaksi, error) {
	rows, err := r.db.QueryContext(ctx,
		sqlLabaPerTransaksi+` ORDER BY s.tanggal DESC, s.id DESC LIMIT ? OFFSET ?`,
		awal, akhir, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil laporan laba: %w", err)
	}
	defer rows.Close()

	hasil := make([]LabaTransaksi, 0)
	for rows.Next() {
		var t LabaTransaksi
		if err := rows.Scan(
			&t.SaleID, &t.Tanggal, &t.Kasir, &t.MetodeBayar,
			&t.Pendapatan, &t.HPP, &t.ReturRefund, &t.ReturHPP, &t.KerugianRetur,
		); err != nil {
			return nil, fmt.Errorf("gagal membaca laporan laba: %w", err)
		}
		hasil = append(hasil, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca laporan laba: %w", err)
	}
	return hasil, nil
}

// RingkasanLaba menjumlahkan komponen laba untuk seluruh transaksi pada
// rentang [awal, akhir), tidak terbatas pada satu halaman.
func (r *Repository) RingkasanLaba(
	ctx context.Context, awal, akhir time.Time,
) (*RingkasanLaba, error) {
	var rl RingkasanLaba
	err := r.db.QueryRowContext(ctx, sqlRingkasanLaba, awal, akhir).Scan(
		&rl.JumlahTransaksi,
		&rl.Pendapatan, &rl.HPP, &rl.ReturRefund, &rl.ReturHPP, &rl.KerugianRetur,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal menghitung ringkasan laba: %w", err)
	}
	return &rl, nil
}