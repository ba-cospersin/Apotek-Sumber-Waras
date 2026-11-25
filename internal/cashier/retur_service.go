package cashier

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	maxPanjangAlasan  = 500
	maxPanjangKondisi = 100 // sesuai returns.kondisi_obat VARCHAR(100)
)

func jenisValid(j string) bool {
	switch j {
	case JenisSalahMembeli, JenisObatRusak, JenisKesalahanPenyerahan, JenisPenarikanResmi:
		return true
	}
	return false
}

// sisaRetur menghitung jumlah yang masih bisa diretur pada satu alokasi.
// Dihitung dalam seperseribu supaya desimal tidak menghasilkan selisih.
func sisaRetur(dialokasikan, sudahDiretur float64) float64 {
	sisa := keMilli(dialokasikan) - keMilli(sudahDiretur)
	if sisa < 0 {
		sisa = 0
	}
	return dariMilli(sisa)
}

// hitungRefund menghitung nilai pengembalian dana dari harga jual snapshot.
func hitungRefund(qtyDasar, hargaJualPerDasar float64) float64 {
	return bulatkan(qtyDasar*hargaJualPerDasar, 2)
}

// PenjualanUntukRetur mengambil transaksi asal beserta sisa yang bisa diretur.
func (s *Service) PenjualanUntukRetur(ctx context.Context, saleID int64) (*PenjualanRetur, error) {
	if saleID <= 0 {
		return nil, validasi("id penjualan tidak valid")
	}
	p, err := s.repo.AmbilPenjualanRetur(ctx, saleID)
	if err != nil {
		return nil, err
	}
	p.NoTransaksi = noTransaksi(p.Tanggal, p.SaleID)
	for i := range p.Items {
		for j := range p.Items[i].Batches {
			b := &p.Items[i].Batches[j]
			b.BisaDiretur = sisaRetur(b.QtyDasar, b.SudahDiretur)
		}
	}
	return p, nil
}

// CatatRetur menyimpan retur konsumen.
//
// Semuanya dalam SATU transaksi: alokasi dikunci, jumlah kumulatif diperiksa,
// retur disimpan, dan jika langsung LAYAK_JUAL stok ditambah ke lapisan
// penerimaan asal. Jika ada langkah yang gagal, tidak ada yang tersimpan.
func (s *Service) CatatRetur(ctx context.Context, userID int64, in ReturInput) (*HasilRetur, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user tidak dikenali")
	}
	if in.SaleItemBatchID <= 0 {
		return nil, validasi("sale_item_batch_id tidak valid")
	}
	qty := bulatkan(in.QtyDasar, 3)
	if qty <= 0 {
		return nil, validasi("jumlah retur harus lebih dari 0")
	}

	jenis := strings.ToUpper(strings.TrimSpace(in.JenisPengembalian))
	if !jenisValid(jenis) {
		return nil, validasi("jenis pengembalian tidak dikenal")
	}

	alasan := strings.TrimSpace(in.Alasan)
	if alasan == "" {
		return nil, validasi("alasan wajib diisi")
	}
	if utf8.RuneCountInString(alasan) > maxPanjangAlasan {
		return nil, validasi("alasan maksimal %d karakter", maxPanjangAlasan)
	}

	kondisi := strings.TrimSpace(in.KondisiObat)
	if utf8.RuneCountInString(kondisi) > maxPanjangKondisi {
		return nil, validasi("kondisi obat maksimal %d karakter", maxPanjangKondisi)
	}

	// Retur konsumen selalu dimulai dari MENUNGGU_PEMERIKSAAN.
	// Penetapan LAYAK_JUAL/TIDAK_LAYAK_JUAL hanya boleh dilakukan Owner
	// melalui endpoint kelayakan, bukan saat Kasir mencatat retur.
	status := StatusMenunggu

	var hasil *HasilRetur

	err := s.repo.WithTx(ctx, func(tx *TxRepository) error {
		a, err := tx.KunciAlokasi(ctx, in.SaleItemBatchID)
		if err != nil {
			return err
		}
		if a == nil {
			return fmt.Errorf("%w (alokasi penjualan %d)", ErrTidakDitemukan, in.SaleItemBatchID)
		}

		sudah, err := tx.TotalRetur(ctx, a.ID)
		if err != nil {
			return err
		}
		sisa := sisaRetur(a.QtyDasar, sudah)
		if keMilli(qty) > keMilli(sisa) {
			return validasi("jumlah retur melebihi yang bisa diretur (sisa %v)", sisa)
		}

		refund := hitungRefund(qty, a.HargaJualPerDasar)

		returID, err := tx.InsertRetur(ctx, ReturBaru{
			SaleItemBatchID: a.ID,
			UserID:          userID,
			Jenis:           jenis,
			QtyDasar:        qty,
			Alasan:          alasan,
			Kondisi:         kondisi,
			Status:          status,
			NilaiRefund:     refund,
			Tanggal:         s.now(),
		})
		if err != nil {
			return err
		}

		// Stok belum ditambah. Penambahan hanya terjadi setelah Owner
		// menetapkan retur sebagai LAYAK_JUAL lewat TetapkanKelayakan.
		ditambah := false

		hasil = &HasilRetur{
			ReturID:         returID,
			SaleItemBatchID: a.ID,
			QtyDasar:        qty,
			NilaiRefund:     refund,
			StatusKelayakan: status,
			StokDitambah:    ditambah,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hasil, nil
}

// TetapkanKelayakan menetapkan hasil pemeriksaan obat retur yang masih
// MENUNGGU_PEMERIKSAAN. Jika LAYAK_JUAL, stok ditambah ke lapisan asal.
// Status hanya bisa ditetapkan satu kali, sehingga stok tidak bisa
// ditambah dua kali.
func (s *Service) TetapkanKelayakan(
	ctx context.Context, returID int64, status string,
) (*HasilKelayakan, error) {
	if returID <= 0 {
		return nil, validasi("id retur tidak valid")
	}
	status = strings.ToUpper(strings.TrimSpace(status))
	if status != StatusLayakJual && status != StatusTidakLayakJual {
		return nil, validasi("status kelayakan harus %s atau %s", StatusLayakJual, StatusTidakLayakJual)
	}

	var hasil *HasilKelayakan

	err := s.repo.WithTx(ctx, func(tx *TxRepository) error {
		r, err := tx.KunciRetur(ctx, returID)
		if err != nil {
			return err
		}
		if r == nil {
			return fmt.Errorf("%w (retur %d)", ErrTidakDitemukan, returID)
		}
		if r.Status != StatusMenunggu {
			return fmt.Errorf("%w (status sekarang %s)", ErrStatusSudahFinal, r.Status)
		}

		if err := tx.UbahKelayakan(ctx, r.ID, status); err != nil {
			return err
		}

		ditambah := false
		if status == StatusLayakJual {
			if err := s.stok.TambahStok(ctx, tx.DB(), r.ReceiptID, r.QtyDasar); err != nil {
				return err
			}
			ditambah = true
		}

		hasil = &HasilKelayakan{
			ReturID:         r.ID,
			StatusKelayakan: status,
			StokDitambah:    ditambah,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hasil, nil
}

const (
	riwayatBawaan = 10
	riwayatMaks   = 50
)

// idDariNomor menerima nomor transaksi berformat TRX-YYYYMMDD-NNNN atau angka biasa.
func idDariNomor(nomor string) (int64, bool) {
	s := strings.ToUpper(strings.TrimSpace(nomor))
	if strings.HasPrefix(s, "TRX-") {
		s = s[strings.LastIndex(s, "-")+1:]
	}
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

// noRetur membentuk nomor retur yang ditampilkan, misalnya RET-20261106-0012.
// Diturunkan dari tanggal dan id, tidak disimpan di database.
func noRetur(tanggal time.Time, id int64) string {
	return fmt.Sprintf("RET-%s-%04d", tanggal.Format("20060102"), id)
}

// CariPenjualanRetur mencari transaksi asal dari nomor berformat TRX-... atau angka.
// Jika berformat TRX, bagian tanggalnya harus cocok dengan tanggal transaksi
// (mencegah salah ketik diterima).
func (s *Service) CariPenjualanRetur(ctx context.Context, nomor string) (*PenjualanRetur, error) {
	id, ok := idDariNomor(nomor)
	if !ok {
		return nil, validasi("nomor transaksi tidak valid")
	}
	p, err := s.PenjualanUntukRetur(ctx, id)
	if err != nil {
		return nil, err
	}

	if bagian := strings.Split(strings.ToUpper(strings.TrimSpace(nomor)), "-"); len(bagian) > 1 {
		if len(bagian) != 3 || bagian[1] != p.Tanggal.Format("20060102") {
			return nil, ErrTidakDitemukan
		}
	}
	return p, nil
}

// RiwayatRetur mengambil retur terbaru untuk ditampilkan di halaman retur.
func (s *Service) RiwayatRetur(ctx context.Context, limit int) ([]ReturRiwayat, error) {
	if limit == 0 {
		limit = riwayatBawaan
	}
	if limit < 0 || limit > riwayatMaks {
		return nil, validasi("limit harus antara 1 dan %d", riwayatMaks)
	}

	daftar, err := s.repo.RiwayatRetur(ctx, limit)
	if err != nil {
		return nil, err
	}
	for i := range daftar {
		daftar[i].NoRetur = noRetur(daftar[i].Tanggal, daftar[i].ReturID)
	}
	return daftar, nil
}
