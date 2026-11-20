package cashier

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	batasHasilCari   = 20
	maxItemTransaksi = 100
)

// Stok adalah kebutuhan kasir terhadap persediaan. Implementasinya ada di
// stok_kasir.go (StokKasir). Semua fungsi menerima koneksi (q) supaya bisa
// berjalan di dalam transaksi penjualan.
type Stok interface {
	StokTersedia(ctx context.Context, q DBTX, medicineID int64, hariIni time.Time) (float64, error)
	// LapisanTersedia hanya mengembalikan lapisan yang boleh dijual
	// (tidak diblokir, belum mendekati kedaluwarsa). Dikunci jika kunci = true.
	LapisanTersedia(ctx context.Context, q DBTX, medicineID int64, hariIni time.Time, kunci bool) ([]Lapisan, error)
	KurangiStok(ctx context.Context, q DBTX, receiptID int64, qty float64) error
}

// Service berisi aturan bisnis modul kasir.
// Tidak bergantung pada net/http dan tidak menjalankan SQL langsung.
type Service struct {
	repo *Repository
	stok Stok
	now  func() time.Time // diganti saat pengujian
}

func NewService(repo *Repository, stok Stok) *Service {
	return &Service{repo: repo, stok: stok, now: time.Now}
}

// hariIni mengembalikan tanggal hari ini tanpa jam.
func (s *Service) hariIni() time.Time {
	n := s.now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
}

// CariObat mencari obat untuk kasir, lengkap dengan satuan jual dan
// stok yang boleh dijual.
func (s *Service) CariObat(ctx context.Context, kata string) ([]Obat, error) {
	kata = strings.TrimSpace(kata)
	if len([]rune(kata)) < 2 {
		return nil, validasi("kata kunci minimal 2 karakter")
	}

	daftar, err := s.repo.CariObat(ctx, kata, batasHasilCari)
	if err != nil {
		return nil, err
	}

	hari := s.hariIni()
	for i := range daftar {
		stok, err := s.stok.StokTersedia(ctx, s.repo.db, daftar[i].ID, hari)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil stok obat %d: %w", daftar[i].ID, err)
		}
		daftar[i].StokTersedia = stok
	}
	return daftar, nil
}

// itemSiap adalah satu baris keranjang yang sudah dihitung.
type itemSiap struct {
	unit  UnitObat
	qty   float64 // dalam satuan pilihan
	hasil HasilItem
}

// Jual menyimpan satu transaksi penjualan.
//
// Semuanya berjalan dalam SATU transaksi database: jika stok tidak cukup
// atau ada langkah yang gagal, tidak ada yang tersimpan dan stok tidak berubah.
// Harga selalu dihitung dari database, bukan dari data browser.
func (s *Service) Jual(
	ctx context.Context,
	userID int64,
	in TransaksiInput,
) (*HasilTransaksi, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user tidak dikenali")
	}

	metode := strings.ToUpper(strings.TrimSpace(in.MetodeBayar))
	if metode != MetodeTunai && metode != MetodeNontunai {
		return nil, validasi("metode pembayaran harus %s atau %s", MetodeTunai, MetodeNontunai)
	}
	if len(in.Items) == 0 {
		return nil, validasi("keranjang masih kosong")
	}
	if len(in.Items) > maxItemTransaksi {
		return nil, validasi("maksimal %d item per transaksi", maxItemTransaksi)
	}
	for i, it := range in.Items {
		if it.UnitID <= 0 {
			return nil, validasi("item %d: unit_id tidak valid", i+1)
		}
		if bulatkan(it.Qty, 3) <= 0 {
			return nil, validasi("item %d: jumlah harus lebih dari 0", i+1)
		}
	}

	var hasil *HasilTransaksi

	err := s.repo.WithTx(ctx, func(tx *TxRepository) error {
		q := tx.DB()
		hari := s.hariIni()

		// 1. Hitung harga setiap item dari data master.
		siap := make([]itemSiap, 0, len(in.Items))
		var total float64
		for i, it := range in.Items {
			u, err := tx.FindUnit(ctx, it.UnitID)
			if err != nil {
				return err
			}
			if u == nil {
				return validasi("item %d: satuan %d tidak ditemukan atau tidak aktif", i+1, it.UnitID)
			}
			h, err := HitungItem(u.SatuanJual, it.Qty)
			if err != nil {
				return fmt.Errorf("item %d (%s): %w", i+1, u.NamaObat, err)
			}
			total += h.Subtotal
			siap = append(siap, itemSiap{unit: *u, qty: bulatkan(it.Qty, 3), hasil: h})
		}
		total = bulatkan(total, 2)

		// 2. Periksa pembayaran sebelum menyentuh stok.
		jumlahBayar := bulatkan(in.JumlahBayar, 2)
		kembalian, err := HitungPembayaran(metode, total, jumlahBayar)
		if err != nil {
			return err
		}

		// 3. Proses obat dengan urutan id yang sama pada setiap transaksi,
		//    supaya dua kasir yang berbelanja obat sama tidak saling mengunci
		//    dalam urutan berbeda (deadlock).
		sort.SliceStable(siap, func(a, b int) bool {
			return siap[a].unit.MedicineID < siap[b].unit.MedicineID
		})

		saleID, err := tx.InsertSale(ctx, userID, s.now(), total, metode, jumlahBayar)
		if err != nil {
			return err
		}

		for _, it := range siap {
			itemID, err := tx.InsertSaleItem(
				ctx, saleID, it.unit.ID, it.qty,
				it.hasil.QtyDasar, it.hasil.HargaJualPerDasar,
			)
			if err != nil {
				return err
			}

			// Dibaca ulang dan dikunci untuk setiap item, sehingga item kedua
			// untuk obat yang sama melihat sisa stok setelah item pertama.
			lapisan, err := s.stok.LapisanTersedia(ctx, q, it.unit.MedicineID, hari, true)
			if err != nil {
				return err
			}
			alokasi, err := Alokasikan(lapisan, it.hasil.QtyDasar)
			if errors.Is(err, ErrStokTidakCukup) {
				return fmt.Errorf("%w untuk %s", ErrStokTidakCukup, it.unit.NamaObat)
			}
			if err != nil {
				return err
			}

			for _, a := range alokasi {
				if err := s.stok.KurangiStok(ctx, q, a.ReceiptID, a.QtyDasar); err != nil {
					return err
				}
				if _, err := tx.InsertSaleItemBatch(
					ctx, itemID, a.ReceiptID, a.QtyDasar, a.HargaBeliPerDasar,
				); err != nil {
					return err
				}
			}
		}

		hasil = &HasilTransaksi{
			SaleID:      saleID,
			Total:       total,
			JumlahBayar: jumlahBayar,
			Kembalian:   kembalian,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return hasil, nil
}