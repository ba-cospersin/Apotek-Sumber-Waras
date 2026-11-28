package warehouse

import (
	"context"
	"fmt"
	"log"
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

// ---------- Persediaan ----------

// StokTersedia mengembalikan jumlah stok obat (satuan dasar) yang BOLEH DIJUAL
// pada tanggal hariIni: batch AKTIF, kedaluwarsa lebih dari satu bulan setelah
// hariIni, dan masih ada sisa. Dipakai modul Kasir untuk menampilkan stok.
// Pemeriksaan yang sah tetap dilakukan Kasir di dalam transaksi penjualan.
// Jika hariIni kosong, dipakai tanggal hari ini.
func (s *Service) StokTersedia(
	ctx context.Context,
	medicineID int64,
	hariIni time.Time,
) (float64, error) {
	if medicineID <= 0 {
		return 0, validasi("medicine_id tidak valid")
	}

	tgl := s.hariIni()
	if !hariIni.IsZero() {
		tgl = time.Date(hariIni.Year(), hariIni.Month(), hariIni.Day(), 0, 0, 0, 0, time.Local)
	}

	return s.repo.SumStokDapatDijual(ctx, medicineID, tambahSatuBulan(tgl).Format(layoutTanggal))
}

// Ringkasan menghitung angka ringkas persediaan.
func (s *Service) Ringkasan(ctx context.Context) (*RingkasanStok, error) {
	hariIni := s.hariIni()
	rs, err := s.repo.RingkasanStok(
		ctx,
		hariIni.Format(layoutTanggal),
		tambahSatuBulan(hariIni).Format(layoutTanggal),
	)
	if err != nil {
		return nil, err
	}
	rs.StokBolehDijual = bulatkan(rs.StokBolehDijual, 3)
	return rs, nil
}

// DaftarBatch mengambil satu halaman daftar stok per batch.
func (s *Service) DaftarBatch(
	ctx context.Context,
	f FilterBatch,
) (*DaftarBatch, error) {
	hariIni := s.hariIni()

	q, page, perPage, err := siapkanFilterBatch(f, hariIni)
	if err != nil {
		return nil, err
	}

	rows, total, err := s.repo.ListBatch(ctx, q)
	if err != nil {
		return nil, err
	}

	for i := range rows {
		if err := isiStatusBatch(&rows[i], hariIni); err != nil {
			return nil, err
		}
	}

	return &DaftarBatch{
		Data: rows,
		Meta: Halaman{
			Page:         page,
			PerPage:      perPage,
			Total:        total,
			TotalHalaman: (total + perPage - 1) / perPage,
		},
	}, nil
}

// DetailBatch mengambil satu batch fisik beserta semua lapisan penerimaannya.
func (s *Service) DetailBatch(
	ctx context.Context,
	id int64,
) (*DetailBatch, error) {
	if id <= 0 {
		return nil, validasi("id batch tidak valid")
	}

	d, err := s.repo.GetBatch(ctx, id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("batch dengan id %d: %w", id, ErrTidakDitemukan)
	}

	d.TotalStok = bulatkan(d.TotalStok, 3)
	if err := isiStatusBatch(&d.BatchStok, s.hariIni()); err != nil {
		return nil, err
	}
	return d, nil
}

// BlokirBatch menandai batch fisik sebagai diblokir. Semua lapisan penerimaan
// batch itu tidak lagi boleh dijual, tetapi stoknya tetap tercatat untuk
// ditindaklanjuti (pemusnahan atau retur ke PBF).
func (s *Service) BlokirBatch(
	ctx context.Context,
	userID, batchID int64,
	alasan string,
) error {
	if userID <= 0 {
		return fmt.Errorf("user tidak dikenali")
	}
	if batchID <= 0 {
		return validasi("id batch tidak valid")
	}

	alasan = strings.TrimSpace(alasan)
	if alasan == "" {
		return validasi("alasan pemblokiran wajib diisi")
	}
	if len([]rune(alasan)) > 500 {
		return validasi("alasan pemblokiran maksimal 500 karakter")
	}

	err := s.repo.WithTx(ctx, func(tx *TxRepository) error {
		b, err := tx.FindBatchByID(ctx, batchID)
		if err != nil {
			return err
		}
		if b == nil {
			return fmt.Errorf("batch dengan id %d: %w", batchID, ErrTidakDitemukan)
		}
		if b.Status == statusBatchDiblokir {
			return ErrSudahDiblokir
		}
		return tx.SetBatchDiblokir(ctx, batchID, alasan, s.now())
	})
	if err != nil {
		return err
	}

	// Skema belum punya kolom "diblokir oleh", jadi pelakunya dicatat di log.
	log.Printf("warehouse: batch %d diblokir oleh user %d, alasan: %s", batchID, userID, alasan)
	return nil
}

// isiStatusBatch mengisi status_jual dan stok_boleh_dijual dari data batch.
func isiStatusBatch(b *BatchStok, hariIni time.Time) error {
	kadaluarsa, err := time.ParseInLocation(layoutTanggal, b.TglKadaluarsa, time.Local)
	if err != nil {
		return fmt.Errorf("tanggal kedaluwarsa batch %d tidak valid: %w", b.BatchID, err)
	}

	b.StatusJual = statusJual(b.StatusBatch, kadaluarsa, hariIni)
	b.StokBolehDijual = 0
	if b.StatusJual == statusJualBoleh {
		b.StokBolehDijual = b.TotalStok
	}
	return nil
}

// hitungHalaman memvalidasi page dan per_page, lalu memberi nilai bawaan.
func hitungHalaman(page, perPage int) (int, int, error) {
	if page < 0 || perPage < 0 {
		return 0, 0, validasi("page dan per_page tidak boleh negatif")
	}
	if page == 0 {
		page = 1
	}
	if perPage == 0 {
		perPage = perHalamanBawaan
	}
	if perPage > perHalamanMaks {
		perPage = perHalamanMaks
	}
	return page, perPage, nil
}

// siapkanFilterBatch memvalidasi filter dan mengubahnya menjadi query.
func siapkanFilterBatch(
	f FilterBatch,
	hariIni time.Time,
) (QueryBatch, int, int, error) {
	var q QueryBatch

	page, perPage, err := hitungHalaman(f.Page, f.PerPage)
	if err != nil {
		return q, 0, 0, err
	}

	kata := strings.TrimSpace(f.Kata)
	if len([]rune(kata)) > 100 {
		return q, 0, 0, validasi("q maksimal 100 karakter")
	}

	status := strings.ToUpper(strings.TrimSpace(f.Status))
	switch status {
	case "", statusJualBoleh, statusJualMendekati, statusJualKedaluwarsa, statusJualDiblokir, statusJualBermasalah:
	default:
		return q, 0, 0, validasi(
			"status harus salah satu dari %s, %s, %s, %s, %s",
			statusJualBoleh, statusJualMendekati, statusJualKedaluwarsa, statusJualDiblokir, statusJualBermasalah,
		)
	}

	q = QueryBatch{
		Kata:         kata,
		Status:       status,
		HariIni:      hariIni.Format(layoutTanggal),
		Batas:        tambahSatuBulan(hariIni).Format(layoutTanggal),
		HanyaAdaStok: !f.TermasukKosong,
		Limit:        perPage,
		Offset:       (page - 1) * perPage,
	}
	return q, page, perPage, nil
}

// ---------- Tindak lanjut stok: pemusnahan dan retur ke PBF ----------

// CatatTindakLanjut mencatat pemusnahan atau retur ke PBF untuk satu lapisan
// penerimaan, mengurangi stok lapisan itu, dan menghitung nilai kerugian
// (harga beli historis x jumlah - kompensasi).
//
// Semuanya berjalan dalam SATU transaksi. Lapisan dikunci (FOR UPDATE) supaya
// penjualan atau tindak lanjut lain yang berjalan bersamaan tidak membuat stok
// negatif.
func (s *Service) CatatTindakLanjut(
	ctx context.Context,
	userID int64,
	in TindakLanjutInput,
) (*TindakLanjut, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user tidak dikenali")
	}

	in, err := validasiTindakLanjut(in)
	if err != nil {
		return nil, err
	}

	waktu := s.now()
	var hasil *TindakLanjut

	err = s.repo.WithTx(ctx, func(tx *TxRepository) error {
		lap, err := tx.FindReceiptForUpdate(ctx, in.BatchReceiptID)
		if err != nil {
			return err
		}
		if lap == nil {
			return fmt.Errorf("lapisan penerimaan dengan id %d: %w", in.BatchReceiptID, ErrTidakDitemukan)
		}

		if in.QtyDasar > lap.QtySisa {
			return fmt.Errorf(
				"lapisan penerimaan %d: jumlah %v, sisa stok %v: %w",
				lap.ID, in.QtyDasar, lap.QtySisa, ErrStokTidakCukup,
			)
		}

		if err := tx.KurangiStokLapisan(ctx, lap.ID, in.QtyDasar); err != nil {
			return err
		}

		id, err := tx.InsertDisposal(ctx, DisposalBaru{
			ReceiptID:  lap.ID,
			UserID:     userID,
			Jenis:      in.Jenis,
			Sumber:     in.SumberTindakLanjut,
			Alasan:     in.Alasan,
			Qty:        in.QtyDasar,
			Kompensasi: in.NilaiKompensasi,
			Catatan:    in.Catatan,
			Waktu:      waktu,
		})
		if err != nil {
			return err
		}

		perolehan := bulatkan(lap.HargaBeliPerDasar*in.QtyDasar, 2)
		hasil = &TindakLanjut{
			DisposalID:         id,
			BatchReceiptID:     lap.ID,
			BatchID:            lap.BatchID,
			Jenis:              in.Jenis,
			SumberTindakLanjut: in.SumberTindakLanjut,
			Alasan:             in.Alasan,
			QtyDasar:           in.QtyDasar,
			NilaiPerolehan:     perolehan,
			NilaiKompensasi:    in.NilaiKompensasi,
			NilaiKerugian:      bulatkan(perolehan-in.NilaiKompensasi, 2),
			SisaSetelah:        bulatkan(lap.QtySisa-in.QtyDasar, 3),
			Tanggal:            waktu.Format("2006-01-02 15:04:05"),
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return hasil, nil
}

// DaftarTindakLanjut mengambil satu halaman riwayat tindak lanjut.
func (s *Service) DaftarTindakLanjut(
	ctx context.Context,
	f FilterTindakLanjut,
) (*DaftarTindakLanjut, error) {
	q, page, perPage, err := siapkanFilterTindakLanjut(f)
	if err != nil {
		return nil, err
	}

	rows, total, err := s.repo.ListTindakLanjut(ctx, q)
	if err != nil {
		return nil, err
	}

	return &DaftarTindakLanjut{
		Data: rows,
		Meta: Halaman{
			Page:         page,
			PerPage:      perPage,
			Total:        total,
			TotalHalaman: (total + perPage - 1) / perPage,
		},
	}, nil
}

// validasiTindakLanjut memeriksa dan merapikan input tanpa menyentuh database.
func validasiTindakLanjut(in TindakLanjutInput) (TindakLanjutInput, error) {
	if in.BatchReceiptID <= 0 {
		return in, validasi("batch_receipt_id wajib diisi")
	}

	in.Jenis = strings.ToUpper(strings.TrimSpace(in.Jenis))
	if in.Jenis != jenisPemusnahan && in.Jenis != jenisReturPBF {
		return in, validasi("jenis harus %s atau %s", jenisPemusnahan, jenisReturPBF)
	}

	in.SumberTindakLanjut = strings.ToUpper(strings.TrimSpace(in.SumberTindakLanjut))
	switch in.SumberTindakLanjut {
	case sumberInternalApotek, sumberBPOM, sumberProdusenPBF:
	default:
		return in, validasi(
			"sumber_tindak_lanjut harus %s, %s, atau %s",
			sumberInternalApotek, sumberBPOM, sumberProdusenPBF,
		)
	}

	in.Alasan = strings.TrimSpace(in.Alasan)
	if in.Alasan == "" {
		return in, validasi("alasan wajib diisi")
	}
	if len([]rune(in.Alasan)) > 100 {
		return in, validasi("alasan maksimal 100 karakter")
	}
	// Alasan yang umum disamakan hurufnya supaya laporan kerugian bisa dikelompokkan.
	switch strings.ToUpper(in.Alasan) {
	case "KEDALUWARSA", "PENARIKAN", "RUSAK":
		in.Alasan = strings.ToUpper(in.Alasan)
	}

	if math.IsNaN(in.QtyDasar) || math.IsInf(in.QtyDasar, 0) || in.QtyDasar <= 0 {
		return in, validasi("qty_dasar harus lebih dari 0")
	}
	in.QtyDasar = bulatkan(in.QtyDasar, 3)
	if in.QtyDasar <= 0 {
		return in, validasi("qty_dasar harus lebih dari 0")
	}

	if math.IsNaN(in.NilaiKompensasi) || math.IsInf(in.NilaiKompensasi, 0) || in.NilaiKompensasi < 0 {
		return in, validasi("nilai_kompensasi tidak boleh negatif")
	}
	if in.NilaiKompensasi > maksNilaiUang {
		return in, validasi("nilai_kompensasi terlalu besar")
	}
	in.NilaiKompensasi = bulatkan(in.NilaiKompensasi, 2)

	in.Catatan = strings.TrimSpace(in.Catatan)
	if len([]rune(in.Catatan)) > 1000 {
		return in, validasi("catatan maksimal 1000 karakter")
	}

	return in, nil
}

// siapkanFilterTindakLanjut memvalidasi filter riwayat.
func siapkanFilterTindakLanjut(
	f FilterTindakLanjut,
) (QueryTindakLanjut, int, int, error) {
	var q QueryTindakLanjut

	page, perPage, err := hitungHalaman(f.Page, f.PerPage)
	if err != nil {
		return q, 0, 0, err
	}

	jenis := strings.ToUpper(strings.TrimSpace(f.Jenis))
	if jenis != "" && jenis != jenisPemusnahan && jenis != jenisReturPBF {
		return q, 0, 0, validasi("jenis harus %s atau %s", jenisPemusnahan, jenisReturPBF)
	}
	if f.BatchID < 0 {
		return q, 0, 0, validasi("batch_id tidak valid")
	}

	dari := strings.TrimSpace(f.Dari)
	sampai := strings.TrimSpace(f.Sampai)

	var tDari, tSampai time.Time
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

	q = QueryTindakLanjut{
		Jenis:   jenis,
		BatchID: f.BatchID,
		Dari:    dari,
		Sampai:  sampai,
		Limit:   perPage,
		Offset:  (page - 1) * perPage,
	}
	return q, page, perPage, nil
}

// ---------- Stock opname ----------

// penyesuaianHitung adalah rencana perubahan stok satu lapisan.
// Satuan milli = seperseribu satuan dasar (bilangan bulat, bebas galat desimal).
type penyesuaianHitung struct {
	lapisan LapisanStok
	milli   int64 // negatif = dikurangi
}

// keMilli mengubah jumlah ke bilangan bulat seperseribu satuan.
func keMilli(x float64) int64 { return int64(math.Round(x * 1000)) }

// dariMilli mengubah bilangan bulat seperseribu satuan kembali ke jumlah.
func dariMilli(m int64) float64 { return float64(m) / 1000 }

// alokasiSelisih membagi selisih hasil opname ke lapisan penerimaan sebuah
// batch, sesuai aturan rancangan (2.3 no. 14):
//
//   - Kekurangan (selisih < 0): mengurangi lapisan yang LEBIH DAHULU diterima.
//     Jika melebihi sisa lapisan itu, dilanjutkan ke lapisan berikutnya.
//     Lapisan yang sisanya 0 dilewati.
//   - Kelebihan (selisih > 0): seluruhnya ditambahkan ke lapisan yang PALING
//     AKHIR diterima.
//
// lapisan HARUS sudah berurutan dari yang paling dahulu diterima. Total
// penyesuaian yang dikembalikan selalu sama dengan selisih.
func alokasiSelisih(lapisan []LapisanStok, selisihMilli int64) ([]penyesuaianHitung, error) {
	if selisihMilli == 0 {
		return nil, nil
	}
	if len(lapisan) == 0 {
		return nil, fmt.Errorf("batch tidak punya lapisan penerimaan")
	}

	// Kelebihan: ke lapisan paling akhir.
	if selisihMilli > 0 {
		return []penyesuaianHitung{{lapisan: lapisan[len(lapisan)-1], milli: selisihMilli}}, nil
	}

	// Kekurangan: dari lapisan paling awal.
	sisaKurang := -selisihMilli
	var hasil []penyesuaianHitung
	for _, l := range lapisan {
		if sisaKurang == 0 {
			break
		}
		tersedia := keMilli(l.QtySisa)
		if tersedia <= 0 {
			continue
		}
		ambil := tersedia
		if sisaKurang < ambil {
			ambil = sisaKurang
		}
		hasil = append(hasil, penyesuaianHitung{lapisan: l, milli: -ambil})
		sisaKurang -= ambil
	}
	if sisaKurang > 0 {
		return nil, fmt.Errorf("kekurangan %v melebihi total stok lapisan", dariMilli(sisaKurang))
	}
	return hasil, nil
}

// CatatStockOpname mencatat hasil hitung fisik satu batch fisik.
//
// Sistem menjumlahkan sisa stok semua lapisan batch itu (stok menurut sistem),
// menghitung selisih (fisik - sistem), mengalokasikannya ke lapisan sesuai
// aturan, dan menyimpan hasil opname beserta detail tiap lapisan yang berubah.
// Selisih TIDAK menimpa angka stok begitu saja: setiap perubahan tercatat.
//
// Semuanya berjalan dalam SATU transaksi. Semua lapisan batch dikunci
// (FOR UPDATE), jadi penjualan atau tindak lanjut yang berjalan bersamaan
// menunggu, dan hitungan stok menurut sistem tidak berubah di tengah jalan.
func (s *Service) CatatStockOpname(
	ctx context.Context,
	userID int64,
	in StockOpnameInput,
) (*StockOpname, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("user tidak dikenali")
	}

	in, waktu, err := s.validasiStockOpname(in)
	if err != nil {
		return nil, err
	}

	var opnameID int64

	err = s.repo.WithTx(ctx, func(tx *TxRepository) error {
		ada, err := tx.BatchAda(ctx, in.BatchID)
		if err != nil {
			return err
		}
		if !ada {
			return fmt.Errorf("batch dengan id %d: %w", in.BatchID, ErrTidakDitemukan)
		}

		lapisan, err := tx.LockLapisanBatch(ctx, in.BatchID)
		if err != nil {
			return err
		}
		if len(lapisan) == 0 {
			return validasi("batch ini belum punya lapisan penerimaan, tidak bisa di-opname")
		}

		var sistemMilli int64
		for _, l := range lapisan {
			sistemMilli += keMilli(l.QtySisa)
		}
		fisikMilli := keMilli(in.QtyFisik)

		rencana, err := alokasiSelisih(lapisan, fisikMilli-sistemMilli)
		if err != nil {
			return err
		}

		id, err := tx.InsertStockOpname(ctx, OpnameBaru{
			BatchID:   in.BatchID,
			UserID:    userID,
			QtySistem: dariMilli(sistemMilli),
			QtyFisik:  dariMilli(fisikMilli),
			Catatan:   in.Catatan,
			Waktu:     waktu,
		})
		if err != nil {
			return err
		}

		for _, p := range rencana {
			sebelum := keMilli(p.lapisan.QtySisa)
			sesudah := sebelum + p.milli

			if p.milli < 0 {
				err = tx.KurangiStokLapisan(ctx, p.lapisan.ID, dariMilli(-p.milli))
			} else {
				err = tx.TambahStokLapisan(ctx, p.lapisan.ID, dariMilli(p.milli))
			}
			if err != nil {
				return err
			}

			if err := tx.InsertOpnameDetail(
				ctx, id, p.lapisan.ID,
				dariMilli(p.milli), dariMilli(sebelum), dariMilli(sesudah),
			); err != nil {
				return err
			}
		}

		opnameID = id
		return nil
	})
	if err != nil {
		return nil, err
	}

	return s.DetailStockOpname(ctx, opnameID)
}

// DetailStockOpname mengambil satu hasil opname beserta penyesuaiannya.
func (s *Service) DetailStockOpname(
	ctx context.Context,
	id int64,
) (*StockOpname, error) {
	if id <= 0 {
		return nil, validasi("id stock opname tidak valid")
	}

	o, err := s.repo.GetStockOpname(ctx, id)
	if err != nil {
		return nil, err
	}
	if o == nil {
		return nil, fmt.Errorf("stock opname dengan id %d: %w", id, ErrTidakDitemukan)
	}

	isiTurunanOpname(o)
	return o, nil
}

// DaftarStockOpname mengambil satu halaman riwayat opname.
func (s *Service) DaftarStockOpname(
	ctx context.Context,
	f FilterStockOpname,
) (*DaftarStockOpname, error) {
	q, page, perPage, err := siapkanFilterStockOpname(f)
	if err != nil {
		return nil, err
	}

	rows, total, err := s.repo.ListStockOpname(ctx, q)
	if err != nil {
		return nil, err
	}

	for i := range rows {
		rows[i].Selisih = bulatkan(rows[i].QtyFisik-rows[i].QtySistem, 3)
		rows[i].StatusSelisih = statusSelisih(rows[i].QtySistem, rows[i].QtyFisik)
	}

	return &DaftarStockOpname{
		Data: rows,
		Meta: Halaman{
			Page:         page,
			PerPage:      perPage,
			Total:        total,
			TotalHalaman: (total + perPage - 1) / perPage,
		},
	}, nil
}

// statusSelisih menentukan SESUAI, KURANG, atau LEBIH dari dua jumlah.
func statusSelisih(sistem, fisik float64) string {
	d := keMilli(fisik) - keMilli(sistem)
	switch {
	case d < 0:
		return selisihKurang
	case d > 0:
		return selisihLebih
	default:
		return selisihSesuai
	}
}

// isiTurunanOpname mengisi selisih, status, dan nilai selisih.
func isiTurunanOpname(o *StockOpname) {
	o.Selisih = bulatkan(o.QtyFisik-o.QtySistem, 3)
	o.StatusSelisih = statusSelisih(o.QtySistem, o.QtyFisik)

	var nilai float64
	for _, p := range o.Penyesuaian {
		nilai += p.QtyPenyesuaian * p.HargaBeliPerDasar
	}
	o.NilaiSelisih = bulatkan(nilai, 2)
}

// validasiStockOpname memeriksa input tanpa menyentuh database dan menentukan
// waktu pencatatan opname.
func (s *Service) validasiStockOpname(
	in StockOpnameInput,
) (StockOpnameInput, time.Time, error) {
	if in.BatchID <= 0 {
		return in, time.Time{}, validasi("batch_id wajib diisi")
	}

	if math.IsNaN(in.QtyFisik) || math.IsInf(in.QtyFisik, 0) || in.QtyFisik < 0 {
		return in, time.Time{}, validasi("qty_fisik tidak boleh negatif")
	}
	if in.QtyFisik > maksQty {
		return in, time.Time{}, validasi("qty_fisik terlalu besar")
	}
	in.QtyFisik = bulatkan(in.QtyFisik, 3)

	in.Catatan = strings.TrimSpace(in.Catatan)
	if len([]rune(in.Catatan)) > 1000 {
		return in, time.Time{}, validasi("catatan maksimal 1000 karakter")
	}

	// Tanggal opname. Kosong = sekarang. Jika diisi, jamnya memakai waktu saat
	// ini supaya urutan beberapa opname pada tanggal yang sama tetap terjaga.
	waktu := s.now()
	if t := strings.TrimSpace(in.TglOpname); t != "" {
		tgl, err := time.ParseInLocation(layoutTanggal, t, time.Local)
		if err != nil {
			return in, time.Time{}, validasi("tgl_opname harus berformat YYYY-MM-DD")
		}
		if tgl.After(s.hariIni()) {
			return in, time.Time{}, validasi("tgl_opname tidak boleh di masa depan")
		}
		n := s.now()
		waktu = time.Date(tgl.Year(), tgl.Month(), tgl.Day(), n.Hour(), n.Minute(), n.Second(), 0, time.Local)
	}

	return in, waktu, nil
}

// siapkanFilterStockOpname memvalidasi filter riwayat opname.
func siapkanFilterStockOpname(
	f FilterStockOpname,
) (QueryStockOpname, int, int, error) {
	var q QueryStockOpname

	page, perPage, err := hitungHalaman(f.Page, f.PerPage)
	if err != nil {
		return q, 0, 0, err
	}
	if f.BatchID < 0 {
		return q, 0, 0, validasi("batch_id tidak valid")
	}

	dari := strings.TrimSpace(f.Dari)
	sampai := strings.TrimSpace(f.Sampai)

	var tDari, tSampai time.Time
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

	q = QueryStockOpname{
		BatchID:      f.BatchID,
		Dari:         dari,
		Sampai:       sampai,
		HanyaSelisih: f.HanyaSelisih,
		Limit:        perPage,
		Offset:       (page - 1) * perPage,
	}
	return q, page, perPage, nil
}

// ---------- Sinkronisasi dengan halaman ----------

// OpsiPenerimaan mengambil pilihan untuk mengisi form penerimaan.
func (s *Service) OpsiPenerimaan(ctx context.Context) (*OpsiPenerimaan, error) {
	return s.repo.ListOpsiPenerimaan(ctx)
}

// PratinjauStockOpname menghitung alokasi opname seandainya hasil hitung fisik
// disimpan sekarang, TANPA mengubah atau mengunci data apa pun.
//
// Perhitungannya memakai alokasiSelisih yang sama dengan CatatStockOpname,
// sehingga yang ditampilkan di layar sama dengan yang nanti disimpan (selama
// stok tidak berubah di antaranya).
func (s *Service) PratinjauStockOpname(
	ctx context.Context,
	batchID int64,
	qtyFisik float64,
) (*PratinjauOpname, error) {
	in, _, err := s.validasiStockOpname(StockOpnameInput{BatchID: batchID, QtyFisik: qtyFisik})
	if err != nil {
		return nil, err
	}

	d, err := s.repo.GetBatch(ctx, in.BatchID)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, fmt.Errorf("batch dengan id %d: %w", in.BatchID, ErrTidakDitemukan)
	}
	if len(d.Lapisan) == 0 {
		return nil, validasi("batch ini belum punya lapisan penerimaan, tidak bisa di-opname")
	}

	lapisan := make([]LapisanStok, len(d.Lapisan))
	var sistemMilli int64
	for i, l := range d.Lapisan {
		lapisan[i] = LapisanStok{ID: l.ReceiptID, BatchID: d.BatchID, QtySisa: l.QtySisa, HargaBeliPerDasar: l.HargaBeliPerDasar}
		sistemMilli += keMilli(l.QtySisa)
	}
	fisikMilli := keMilli(in.QtyFisik)

	rencana, err := alokasiSelisih(lapisan, fisikMilli-sistemMilli)
	if err != nil {
		return nil, err
	}
	perubahan := map[int64]int64{}
	for _, p := range rencana {
		perubahan[p.lapisan.ID] = p.milli
	}

	hasil := &PratinjauOpname{
		BatchID:     d.BatchID,
		MedicineID:  d.MedicineID,
		KodeObat:    d.KodeObat,
		NamaObat:    d.NamaObat,
		SatuanDasar: d.SatuanDasar,
		NoBatch:     d.NoBatch,
		QtySistem:   dariMilli(sistemMilli),
		QtyFisik:    dariMilli(fisikMilli),
		Selisih:     dariMilli(fisikMilli - sistemMilli),
		Lapisan:     make([]PenyesuaianLapisan, 0, len(d.Lapisan)),
	}
	hasil.StatusSelisih = statusSelisih(hasil.QtySistem, hasil.QtyFisik)

	var nilai float64
	for _, l := range d.Lapisan {
		sebelum := keMilli(l.QtySisa)
		ubah := perubahan[l.ReceiptID]
		nilai += dariMilli(ubah) * l.HargaBeliPerDasar
		hasil.Lapisan = append(hasil.Lapisan, PenyesuaianLapisan{
			ReceiptID:         l.ReceiptID,
			NoFaktur:          l.NoFaktur,
			TanggalTerima:     l.TanggalTerima,
			HargaBeliPerDasar: l.HargaBeliPerDasar,
			QtyPenyesuaian:    dariMilli(ubah),
			QtySisaSebelum:    dariMilli(sebelum),
			QtySisaSesudah:    dariMilli(sebelum + ubah),
		})
	}
	hasil.NilaiSelisih = bulatkan(nilai, 2)

	return hasil, nil
}
