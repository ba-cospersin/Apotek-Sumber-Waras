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

// fakturTerima adalah seluruh input penerimaan yang sudah divalidasi.
type fakturTerima struct {
	noFaktur  string
	tglFaktur time.Time
	tglTerima time.Time
	items     []itemTerima
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

	fk, err := s.validasiPenerimaan(in)
	if err != nil {
		return nil, err
	}
	noFaktur, tglFaktur, tglTerima, items := fk.noFaktur, fk.tglFaktur, fk.tglTerima, fk.items
	hariIni := s.hariIni()

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

			// Batch yang diblokir tetap boleh menerima lapisan baru: stoknya
			// tercatat (tidak bisa dijual) lalu ditindaklanjuti lewat RETUR_PBF.
			statusBatch := statusBatchAktif
			if batch != nil {
				statusBatch = batch.Status
			}

			var batchID int64
			switch {
			case batch == nil:
				batchID, err = tx.InsertBatch(ctx, it.medicineID, it.noBatch, it.tglKadaluarsa)
				if err != nil {
					return err
				}

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
			receiptID, err := tx.InsertReceipt(ctx, batchID, purchaseID, qtyDasar, hargaPerDasar, tglTerima)
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
				StatusJual:        statusJual(statusBatch, it.tglKadaluarsa, hariIni),
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
			TglTerima:  tglTerima.Format(layoutTanggal),
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
) (*fakturTerima, error) {
	hariIni := s.hariIni()

	if in.SupplierID <= 0 {
		return nil, validasi("supplier_id wajib diisi")
	}

	noFaktur := strings.TrimSpace(in.NoFaktur)
	if noFaktur == "" {
		return nil, validasi("no_faktur wajib diisi")
	}
	if len(noFaktur) > 100 {
		return nil, validasi("no_faktur maksimal 100 karakter")
	}

	tglFaktur, err := time.ParseInLocation(layoutTanggal, strings.TrimSpace(in.TglFaktur), time.Local)
	if err != nil {
		return nil, validasi("tgl_faktur harus berformat YYYY-MM-DD")
	}
	if tglFaktur.After(hariIni) {
		return nil, validasi("tgl_faktur tidak boleh di masa depan")
	}

	// Tanggal penerimaan fisik. Kosong = sekarang. Jika diisi, jamnya memakai
	// waktu saat ini supaya urutan beberapa penerimaan pada tanggal yang sama
	// tetap sesuai urutan pencatatan (dipakai sebagai urutan kedua FEFO).
	tglTerima := s.now()
	if t := strings.TrimSpace(in.TglTerima); t != "" {
		tgl, err := time.ParseInLocation(layoutTanggal, t, time.Local)
		if err != nil {
			return nil, validasi("tgl_terima harus berformat YYYY-MM-DD")
		}
		if tgl.After(hariIni) {
			return nil, validasi("tgl_terima tidak boleh di masa depan")
		}
		n := s.now()
		tglTerima = time.Date(tgl.Year(), tgl.Month(), tgl.Day(), n.Hour(), n.Minute(), n.Second(), 0, time.Local)
	}

	if len(in.Items) == 0 {
		return nil, validasi("items minimal berisi satu obat")
	}

	items := make([]itemTerima, 0, len(in.Items))
	for i, it := range in.Items {
		nomor := i + 1

		if it.MedicineID <= 0 {
			return nil, validasi("item %d: medicine_id wajib diisi", nomor)
		}
		if it.UnitID < 0 {
			return nil, validasi("item %d: unit_id tidak valid", nomor)
		}

		noBatch := strings.TrimSpace(it.NoBatch)
		if noBatch == "" {
			return nil, validasi("item %d: no_batch wajib diisi", nomor)
		}
		if len(noBatch) > 100 {
			return nil, validasi("item %d: no_batch maksimal 100 karakter", nomor)
		}

		kadaluarsa, err := time.ParseInLocation(layoutTanggal, strings.TrimSpace(it.TglKadaluarsa), time.Local)
		if err != nil {
			return nil, validasi("item %d: tgl_kadaluarsa harus berformat YYYY-MM-DD", nomor)
		}
		if it.Qty <= 0 {
			return nil, validasi("item %d: qty harus lebih dari 0", nomor)
		}
		if it.HargaBeli < 0 {
			return nil, validasi("item %d: harga_beli tidak boleh negatif", nomor)
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

	return &fakturTerima{
		noFaktur:  noFaktur,
		tglFaktur: tglFaktur,
		tglTerima: tglTerima,
		items:     items,
	}, nil
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

// ---------- Riwayat penerimaan ----------

const (
	perHalamanBawaan = 20
	perHalamanMaks   = 100
)

// DaftarPenerimaan mengambil satu halaman daftar faktur.
func (s *Service) DaftarPenerimaan(
	ctx context.Context,
	f FilterPenerimaan,
) (*DaftarPenerimaan, error) {
	q, page, perPage, err := siapkanFilter(f)
	if err != nil {
		return nil, err
	}

	rows, total, err := s.repo.ListPenerimaan(ctx, q)
	if err != nil {
		return nil, err
	}

	return &DaftarPenerimaan{
		Data: rows,
		Meta: Halaman{
			Page:         page,
			PerPage:      perPage,
			Total:        total,
			TotalHalaman: (total + perPage - 1) / perPage,
		},
	}, nil
}

// DetailPenerimaan mengambil satu faktur beserta itemnya.
func (s *Service) DetailPenerimaan(
	ctx context.Context,
	id int64,
) (*DetailPenerimaan, error) {
	if id <= 0 {
		return nil, validasi("id faktur tidak valid")
	}

	d, err := s.repo.GetPenerimaan(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("faktur dengan id %d: %w", id, ErrTidakDitemukan)
	}
	return d, nil
}

// siapkanFilter memvalidasi filter dan mengubah halaman menjadi LIMIT/OFFSET.
func siapkanFilter(f FilterPenerimaan) (QueryPenerimaan, int, int, error) {
	var q QueryPenerimaan

	if f.SupplierID < 0 {
		return q, 0, 0, validasi("supplier_id tidak valid")
	}
	if f.Page < 0 || f.PerPage < 0 {
		return q, 0, 0, validasi("page dan per_page tidak boleh negatif")
	}

	page := f.Page
	if page == 0 {
		page = 1
	}
	perPage := f.PerPage
	if perPage == 0 {
		perPage = perHalamanBawaan
	}
	if perPage > perHalamanMaks {
		perPage = perHalamanMaks
	}

	dari := strings.TrimSpace(f.Dari)
	sampai := strings.TrimSpace(f.Sampai)

	var tDari, tSampai time.Time
	var err error

	if dari != "" {
		tDari, err = time.ParseInLocation(layoutTanggal, dari, time.Local)
		if err != nil {
			return q, 0, 0, validasi("dari harus berformat YYYY-MM-DD")
		}
	}
	if sampai != "" {
		tSampai, err = time.ParseInLocation(layoutTanggal, sampai, time.Local)
		if err != nil {
			return q, 0, 0, validasi("sampai harus berformat YYYY-MM-DD")
		}
	}
	if dari != "" && sampai != "" && tDari.After(tSampai) {
		return q, 0, 0, validasi("dari tidak boleh lebih besar dari sampai")
	}

	q = QueryPenerimaan{
		SupplierID: f.SupplierID,
		Dari:       dari,
		Sampai:     sampai,
		Limit:      perPage,
		Offset:     (page - 1) * perPage,
	}
	return q, page, perPage, nil
}

// ---------- Status jual ----------

// Status jual stok yang baru diterima.
const (
	statusJualBoleh       = "BOLEH_DIJUAL"
	statusJualMendekati   = "MENDEKATI_KEDALUWARSA"
	statusJualKedaluwarsa = "KEDALUWARSA"
	statusJualDiblokir    = "DIBLOKIR"
)

// statusJual menentukan apakah stok boleh dijual pada hari ini, menurut
// aturan rancangan: batch diblokir tidak boleh dijual, dan obat hanya boleh
// dijual jika kedaluwarsanya LEBIH dari satu bulan setelah hari ini.
func statusJual(statusBatch string, kadaluarsa, hariIni time.Time) string {
	switch {
	case statusBatch == statusBatchDiblokir:
		return statusJualDiblokir
	case !kadaluarsa.After(hariIni):
		return statusJualKedaluwarsa
	case !kadaluarsa.After(tambahSatuBulan(hariIni)):
		return statusJualMendekati
	default:
		return statusJualBoleh
	}
}

// tambahSatuBulan menambah satu bulan kalender. Jika tanggal tujuan tidak ada
// (misalnya 31 Januari + 1 bulan), dipakai hari terakhir bulan tujuan.
func tambahSatuBulan(t time.Time) time.Time {
	y, m, d := t.Date()
	m++
	akhir := time.Date(y, m+1, 0, 0, 0, 0, 0, t.Location()).Day()
	if d > akhir {
		d = akhir
	}
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
