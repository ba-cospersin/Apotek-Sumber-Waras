package warehouse

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// Service berisi aturan bisnis modul gudang.
// Tidak bergantung pada net/http dan tidak menjalankan SQL langsung.
type Service struct {
	repo *Repository
	now  func() time.Time // diganti saat pengujian
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo, now: time.Now}
}

// itemTerima adalah item penerimaan yang sudah divalidasi dan dirapikan.
type itemTerima struct {
	medicineID    int64
	unitID        int64
	noBatch       string
	tglKadaluarsa time.Time
	qty           float64
	hargaBeli     float64
}

// CatatPenerimaan mencatat satu faktur PBF beserta semua itemnya.
//
// Semuanya berjalan dalam SATU transaksi: jika satu item gagal,
// faktur dan item lain yang sudah diproses ikut dibatalkan.
func (s *Service) CatatPenerimaan(
	ctx context.Context,
	userID int64,
	in PenerimaanInput,
) (*Penerimaan, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user tidak dikenali")
	}

	noFaktur, tglFaktur, items, err := s.validasiPenerimaan(in)
	if err != nil {
		return nil, err
	}

	var hasil *Penerimaan

	err = s.repo.WithTx(ctx, func(tx *TxRepository) error {
		ada, err := tx.SupplierExists(ctx, in.SupplierID)
		if err != nil {
			return err
		}
		if !ada {
			return validasi("supplier_id %d tidak ditemukan", in.SupplierID)
		}

		purchaseID, err := tx.InsertPurchase(
			ctx, in.SupplierID, userID, noFaktur, tglFaktur,
		)
		if err != nil {
			return err
		}

		var total float64
		hasilItems := make([]ItemPenerimaan, 0, len(items))

		for i, it := range items {
			nomor := i + 1

			// 1. Pastikan obat ada dan aktif.
			med, err := tx.FindMedicine(ctx, it.medicineID)
			if err != nil {
				return err
			}
			if med == nil {
				return validasi("item %d: obat dengan id %d tidak ditemukan", nomor, it.medicineID)
			}
			if !med.IsActive {
				return validasi("item %d: obat dengan id %d tidak aktif", nomor, it.medicineID)
			}

			// 2. Tentukan konversi ke satuan dasar.
			isi := 1.0
			if it.unitID != 0 {
				unit, err := tx.FindUnit(ctx, it.medicineID, it.unitID)
				if err != nil {
					return err
				}
				if unit == nil {
					return validasi("item %d: satuan id %d bukan milik obat id %d", nomor, it.unitID, it.medicineID)
				}
				if !unit.IsActive {
					return validasi("item %d: satuan id %d tidak aktif", nomor, it.unitID)
				}
				if unit.IsiPerSatuanDasar <= 0 {
					return fmt.Errorf("konversi satuan id %d tidak valid di database", it.unitID)
				}
				isi = unit.IsiPerSatuanDasar
			}

			qtyDasar := bulatkan(it.qty*isi, 3)
			if qtyDasar <= 0 {
				return validasi("item %d: jumlah dalam satuan dasar harus lebih dari 0", nomor)
			}
			hargaPerDasar := bulatkan(it.hargaBeli/isi, 2)

			// 3. Cari batch fisik; buat baru kalau belum ada.
			batch, err := tx.FindBatchForUpdate(ctx, it.medicineID, it.noBatch)
			if err != nil {
				return err
			}

			var batchID int64
			switch {
			case batch == nil:
				batchID, err = tx.InsertBatch(ctx, it.medicineID, it.noBatch, it.tglKadaluarsa)
				if err != nil {
					return err
				}

			case batch.Status == statusBatchDiblokir:
				return fmt.Errorf("item %d (batch %s): %w", nomor, it.noBatch, ErrBatchDiblokir)

			case batch.TglKadaluarsa.Format(layoutTanggal) != it.tglKadaluarsa.Format(layoutTanggal):
				return fmt.Errorf(
					"item %d (batch %s): tercatat %s, input %s: %w",
					nomor, it.noBatch,
					batch.TglKadaluarsa.Format(layoutTanggal),
					it.tglKadaluarsa.Format(layoutTanggal),
					ErrKedaluwarsaBeda,
				)

			default:
				batchID = batch.ID
			}

			// 4. Tambah lapisan penerimaan baru (inilah yang menambah stok).
			receiptID, err := tx.InsertReceipt(ctx, batchID, purchaseID, qtyDasar, hargaPerDasar)
			if err != nil {
				return err
			}

			total += it.qty * it.hargaBeli
			hasilItems = append(hasilItems, ItemPenerimaan{
				ReceiptID:         receiptID,
				BatchID:           batchID,
				MedicineID:        it.medicineID,
				NoBatch:           it.noBatch,
				TglKadaluarsa:     it.tglKadaluarsa.Format(layoutTanggal),
				QtyDasar:          qtyDasar,
				HargaBeliPerDasar: hargaPerDasar,
			})
		}

		total = bulatkan(total, 2)
		if err := tx.UpdatePurchaseTotal(ctx, purchaseID, total); err != nil {
			return err
		}

		hasil = &Penerimaan{
			PurchaseID: purchaseID,
			SupplierID: in.SupplierID,
			NoFaktur:   noFaktur,
			TglFaktur:  tglFaktur.Format(layoutTanggal),
			Total:      total,
			Items:      hasilItems,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return hasil, nil
}

// validasiPenerimaan memeriksa input tanpa menyentuh database.
func (s *Service) validasiPenerimaan(
	in PenerimaanInput,
) (string, time.Time, []itemTerima, error) {
	hariIni := s.hariIni()

	if in.SupplierID <= 0 {
		return "", time.Time{}, nil, validasi("supplier_id wajib diisi")
	}

	noFaktur := strings.TrimSpace(in.NoFaktur)
	if noFaktur == "" {
		return "", time.Time{}, nil, validasi("no_faktur wajib diisi")
	}
	if len(noFaktur) > 100 {
		return "", time.Time{}, nil, validasi("no_faktur maksimal 100 karakter")
	}

	tglFaktur, err := time.ParseInLocation(layoutTanggal, strings.TrimSpace(in.TglFaktur), time.Local)
	if err != nil {
		return "", time.Time{}, nil, validasi("tgl_faktur harus berformat YYYY-MM-DD")
	}
	if tglFaktur.After(hariIni) {
		return "", time.Time{}, nil, validasi("tgl_faktur tidak boleh di masa depan")
	}

	if len(in.Items) == 0 {
		return "", time.Time{}, nil, validasi("items minimal berisi satu obat")
	}

	items := make([]itemTerima, 0, len(in.Items))
	for i, it := range in.Items {
		nomor := i + 1

		if it.MedicineID <= 0 {
			return "", time.Time{}, nil, validasi("item %d: medicine_id wajib diisi", nomor)
		}
		if it.UnitID < 0 {
			return "", time.Time{}, nil, validasi("item %d: unit_id tidak valid", nomor)
		}

		noBatch := strings.TrimSpace(it.NoBatch)
		if noBatch == "" {
			return "", time.Time{}, nil, validasi("item %d: no_batch wajib diisi", nomor)
		}
		if len(noBatch) > 100 {
			return "", time.Time{}, nil, validasi("item %d: no_batch maksimal 100 karakter", nomor)
		}

		kadaluarsa, err := time.ParseInLocation(layoutTanggal, strings.TrimSpace(it.TglKadaluarsa), time.Local)
		if err != nil {
			return "", time.Time{}, nil, validasi("item %d: tgl_kadaluarsa harus berformat YYYY-MM-DD", nomor)
		}
		if !kadaluarsa.After(hariIni) {
			return "", time.Time{}, nil, validasi("item %d: obat sudah kedaluwarsa, tidak bisa diterima", nomor)
		}

		if it.Qty <= 0 {
			return "", time.Time{}, nil, validasi("item %d: qty harus lebih dari 0", nomor)
		}
		if it.HargaBeli < 0 {
			return "", time.Time{}, nil, validasi("item %d: harga_beli tidak boleh negatif", nomor)
		}

		items = append(items, itemTerima{
			medicineID:    it.MedicineID,
			unitID:        it.UnitID,
			noBatch:       noBatch,
			tglKadaluarsa: kadaluarsa,
			qty:           it.Qty,
			hargaBeli:     it.HargaBeli,
		})
	}

	return noFaktur, tglFaktur, items, nil
}

// hariIni mengembalikan tanggal hari ini (jam 00:00, zona waktu lokal).
func (s *Service) hariIni() time.Time {
	n := s.now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.Local)
}

// bulatkan membulatkan angka ke sejumlah desimal tertentu.
func bulatkan(x float64, desimal int) float64 {
	p := math.Pow(10, float64(desimal))
	return math.Round(x*p) / p
}
