package warehouse

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"
	"time"
)

// serviceUji membuat Service dengan "hari ini" yang tetap: 7 November 2026.
// repo tidak dipakai karena validasiPenerimaan tidak menyentuh database.
func serviceUji() *Service {
	return &Service{
		now: func() time.Time {
			return time.Date(2026, 11, 7, 10, 0, 0, 0, time.Local)
		},
	}
}

// inputValid mengembalikan input yang lolos validasi.
func inputValid() PenerimaanInput {
	return PenerimaanInput{
		SupplierID: 1,
		NoFaktur:   "FK-001",
		TglFaktur:  "2026-11-07",
		Items: []ItemPenerimaanInput{
			{
				MedicineID:    1,
				NoBatch:       "AMX-C-1126",
				TglKadaluarsa: "2028-01-31",
				Qty:           10,
				HargaBeli:     5000,
			},
		},
	}
}

func TestValidasiPenerimaan_Valid(t *testing.T) {
	fk, err := serviceUji().validasiPenerimaan(inputValid())
	if err != nil {
		t.Fatalf("seharusnya valid, tapi error: %v", err)
	}
	if fk.noFaktur != "FK-001" || len(fk.items) != 1 {
		t.Fatalf("hasil tidak sesuai: %q, %d item", fk.noFaktur, len(fk.items))
	}
	// tgl_terima kosong -> memakai waktu sekarang (7 Nov 2026).
	if fk.tglTerima.Format(layoutTanggal) != "2026-11-07" {
		t.Fatalf("tgl_terima bawaan salah: %v", fk.tglTerima)
	}
}

func TestValidasiPenerimaan_Tolak(t *testing.T) {
	kasus := []struct {
		nama string
		ubah func(in *PenerimaanInput)
	}{
		{"supplier kosong", func(in *PenerimaanInput) { in.SupplierID = 0 }},
		{"no faktur kosong", func(in *PenerimaanInput) { in.NoFaktur = "   " }},
		{"tgl faktur salah format", func(in *PenerimaanInput) { in.TglFaktur = "07-11-2026" }},
		{"tgl faktur masa depan", func(in *PenerimaanInput) { in.TglFaktur = "2026-11-08" }},
		{"items kosong", func(in *PenerimaanInput) { in.Items = nil }},
		{"medicine_id kosong", func(in *PenerimaanInput) { in.Items[0].MedicineID = 0 }},
		{"unit_id negatif", func(in *PenerimaanInput) { in.Items[0].UnitID = -1 }},
		{"no batch kosong", func(in *PenerimaanInput) { in.Items[0].NoBatch = "" }},
		{"kedaluwarsa salah format", func(in *PenerimaanInput) { in.Items[0].TglKadaluarsa = "besok" }},
		{"tgl terima salah format", func(in *PenerimaanInput) { in.TglTerima = "kemarin" }},
		{"tgl terima masa depan", func(in *PenerimaanInput) { in.TglTerima = "2026-11-08" }},
		{"qty nol", func(in *PenerimaanInput) { in.Items[0].Qty = 0 }},
		{"qty negatif", func(in *PenerimaanInput) { in.Items[0].Qty = -5 }},
		{"harga negatif", func(in *PenerimaanInput) { in.Items[0].HargaBeli = -1 }},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			in := inputValid()
			k.ubah(&in)

			_, err := serviceUji().validasiPenerimaan(in)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}

func TestBulatkan(t *testing.T) {
	if got := bulatkan(1666.666666, 2); got != 1666.67 {
		t.Fatalf("bulatkan 2 desimal: dapat %v", got)
	}
	if got := bulatkan(2.00049, 3); got != 2.0 {
		t.Fatalf("bulatkan 3 desimal: dapat %v", got)
	}
}

func TestSiapkanFilter_Bawaan(t *testing.T) {
	q, page, perPage, err := siapkanFilter(FilterPenerimaan{})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if page != 1 || perPage != 20 || q.Limit != 20 || q.Offset != 0 {
		t.Fatalf("bawaan salah: page=%d perPage=%d limit=%d offset=%d",
			page, perPage, q.Limit, q.Offset)
	}
}

func TestSiapkanFilter_Halaman(t *testing.T) {
	q, page, perPage, err := siapkanFilter(FilterPenerimaan{Page: 3, PerPage: 10})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if page != 3 || perPage != 10 || q.Limit != 10 || q.Offset != 20 {
		t.Fatalf("paginasi salah: page=%d perPage=%d limit=%d offset=%d",
			page, perPage, q.Limit, q.Offset)
	}
}

func TestSiapkanFilter_PerPageDibatasi(t *testing.T) {
	_, _, perPage, err := siapkanFilter(FilterPenerimaan{PerPage: 5000})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if perPage != 100 {
		t.Fatalf("per_page harus dibatasi 100, dapat %d", perPage)
	}
}

func TestSiapkanFilter_Tolak(t *testing.T) {
	kasus := []struct {
		nama   string
		filter FilterPenerimaan
	}{
		{"supplier negatif", FilterPenerimaan{SupplierID: -1}},
		{"page negatif", FilterPenerimaan{Page: -1}},
		{"per_page negatif", FilterPenerimaan{PerPage: -5}},
		{"dari salah format", FilterPenerimaan{Dari: "07/11/2026"}},
		{"sampai salah format", FilterPenerimaan{Sampai: "kemarin"}},
		{"dari lebih besar dari sampai", FilterPenerimaan{Dari: "2026-11-10", Sampai: "2026-11-01"}},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			_, _, _, err := siapkanFilter(k.filter)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}

// Obat yang sudah kedaluwarsa HARUS lolos validasi: dicatat dulu,
// lalu ditindaklanjuti lewat RETUR_PBF.
func TestValidasiPenerimaan_KedaluwarsaDiterima(t *testing.T) {
	in := inputValid()
	in.Items[0].TglKadaluarsa = "2025-01-01"

	if _, err := serviceUji().validasiPenerimaan(in); err != nil {
		t.Fatalf("obat kedaluwarsa seharusnya diterima, dapat: %v", err)
	}
}

func TestValidasiPenerimaan_TglTerima(t *testing.T) {
	in := inputValid()
	in.TglTerima = "2026-11-05"

	fk, err := serviceUji().validasiPenerimaan(in)
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if fk.tglTerima.Format(layoutTanggal) != "2026-11-05" {
		t.Fatalf("tgl_terima salah: %v", fk.tglTerima)
	}
	// Jam memakai waktu saat ini (10:00 pada serviceUji).
	if fk.tglTerima.Hour() != 10 {
		t.Fatalf("jam seharusnya mengikuti waktu sekarang, dapat %d", fk.tglTerima.Hour())
	}
}

func TestStatusJual(t *testing.T) {
	hariIni := time.Date(2026, 11, 7, 0, 0, 0, 0, time.Local)
	tgl := func(s string) time.Time {
		x, _ := time.ParseInLocation(layoutTanggal, s, time.Local)
		return x
	}

	kasus := []struct {
		nama, status, kadaluarsa, harapan string
	}{
		{"jauh", statusBatchAktif, "2028-01-01", statusJualBoleh},
		{"tepat 1 bulan + 1 hari", statusBatchAktif, "2026-12-08", statusJualBoleh},
		{"tepat 1 bulan (tidak boleh)", statusBatchAktif, "2026-12-07", statusJualMendekati},
		{"kurang dari 1 bulan", statusBatchAktif, "2026-11-27", statusJualMendekati},
		{"hari ini", statusBatchAktif, "2026-11-07", statusJualKedaluwarsa},
		{"sudah lewat", statusBatchAktif, "2026-01-01", statusJualKedaluwarsa},
		{"diblokir menang atas ED jauh", statusBatchDiblokir, "2028-01-01", statusJualDiblokir},
		{"diblokir menang atas kedaluwarsa", statusBatchDiblokir, "2026-01-01", statusJualDiblokir},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			if got := statusJual(k.status, tgl(k.kadaluarsa), hariIni); got != k.harapan {
				t.Fatalf("dapat %s, seharusnya %s", got, k.harapan)
			}
		})
	}
}

func TestTambahSatuBulan(t *testing.T) {
	tgl := func(s string) time.Time {
		x, _ := time.ParseInLocation(layoutTanggal, s, time.Local)
		return x
	}
	kasus := map[string]string{
		"2026-11-07": "2026-12-07",
		"2026-12-15": "2027-01-15", // pindah tahun
		"2026-01-31": "2026-02-28", // tanggal tujuan tidak ada
		"2028-01-31": "2028-02-29", // tahun kabisat
		"2026-03-31": "2026-04-30",
	}
	for masuk, harapan := range kasus {
		if got := tambahSatuBulan(tgl(masuk)).Format(layoutTanggal); got != harapan {
			t.Fatalf("%s + 1 bulan: dapat %s, seharusnya %s", masuk, got, harapan)
		}
	}
}

// ---------- Persediaan ----------

func TestSiapkanFilterBatch_Bawaan(t *testing.T) {
	hariIni := time.Date(2026, 11, 7, 0, 0, 0, 0, time.Local)

	q, page, perPage, err := siapkanFilterBatch(FilterBatch{}, hariIni)
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if page != 1 || perPage != 20 || q.Limit != 20 || q.Offset != 0 {
		t.Fatalf("paginasi bawaan salah: %d %d %d %d", page, perPage, q.Limit, q.Offset)
	}
	if !q.HanyaAdaStok {
		t.Fatalf("bawaannya hanya batch yang masih ada stok")
	}
	if q.HariIni != "2026-11-07" || q.Batas != "2026-12-07" {
		t.Fatalf("tanggal salah: hari ini %s, batas %s", q.HariIni, q.Batas)
	}
}

func TestSiapkanFilterBatch_Opsi(t *testing.T) {
	hariIni := time.Date(2026, 11, 7, 0, 0, 0, 0, time.Local)

	q, _, _, err := siapkanFilterBatch(FilterBatch{
		Kata: "  para  ", Status: "diblokir", TermasukKosong: true, Page: 3, PerPage: 500,
	}, hariIni)
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if q.Kata != "para" || q.Status != statusJualDiblokir || q.HanyaAdaStok {
		t.Fatalf("opsi tidak terbaca: %+v", q)
	}
	if q.Limit != 100 || q.Offset != 200 {
		t.Fatalf("per_page harus dibatasi 100: limit %d offset %d", q.Limit, q.Offset)
	}
}

func TestSiapkanFilterBatch_Tolak(t *testing.T) {
	hariIni := time.Date(2026, 11, 7, 0, 0, 0, 0, time.Local)

	kasus := []struct {
		nama   string
		filter FilterBatch
	}{
		{"status ngawur", FilterBatch{Status: "SEMUA"}},
		{"page negatif", FilterBatch{Page: -1}},
		{"per_page negatif", FilterBatch{PerPage: -1}},
		{"kata terlalu panjang", FilterBatch{Kata: strings.Repeat("a", 101)}},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			_, _, _, err := siapkanFilterBatch(k.filter, hariIni)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}

func TestEscapeLike(t *testing.T) {
	kasus := map[string]string{
		"para": "para",
		"50%":  "50!%",
		"a_b":  "a!_b",
		"a!b":  "a!!b",
		"%_!":  "!%!_!!",
	}
	for masuk, harapan := range kasus {
		if got := escapeLike(masuk); got != harapan {
			t.Fatalf("escapeLike(%q) = %q, seharusnya %q", masuk, got, harapan)
		}
	}
}

func TestBlokirBatch_Validasi(t *testing.T) {
	// Validasi berjalan sebelum menyentuh database, jadi repo boleh nil.
	svc := serviceUji()
	ctx := context.Background()

	kasus := []struct {
		nama    string
		batchID int64
		alasan  string
	}{
		{"id nol", 0, "penarikan"},
		{"alasan kosong", 1, ""},
		{"alasan spasi saja", 1, "   "},
		{"alasan terlalu panjang", 1, strings.Repeat("x", 501)},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			err := svc.BlokirBatch(ctx, 1, k.batchID, k.alasan)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}

func TestStokTersedia_Validasi(t *testing.T) {
	if _, err := serviceUji().StokTersedia(context.Background(), 0, time.Time{}); !errors.Is(err, ErrValidasi) {
		t.Fatalf("medicine_id 0 seharusnya ErrValidasi, dapat: %v", err)
	}
}

func TestIsiStatusBatch(t *testing.T) {
	hariIni := time.Date(2026, 11, 7, 0, 0, 0, 0, time.Local)

	b := BatchStok{BatchID: 1, TglKadaluarsa: "2028-01-01", StatusBatch: statusBatchAktif, TotalStok: 120}
	if err := isiStatusBatch(&b, hariIni); err != nil {
		t.Fatal(err)
	}
	if b.StatusJual != statusJualBoleh || b.StokBolehDijual != 120 {
		t.Fatalf("batch aktif jauh: %+v", b)
	}

	b = BatchStok{BatchID: 2, TglKadaluarsa: "2028-01-01", StatusBatch: statusBatchDiblokir, TotalStok: 120}
	_ = isiStatusBatch(&b, hariIni)
	if b.StatusJual != statusJualDiblokir || b.StokBolehDijual != 0 {
		t.Fatalf("batch diblokir tidak boleh dijual: %+v", b)
	}

	b = BatchStok{BatchID: 3, TglKadaluarsa: "bukan-tanggal", StatusBatch: statusBatchAktif}
	if err := isiStatusBatch(&b, hariIni); err == nil {
		t.Fatalf("tanggal rusak seharusnya error")
	}
}

// ---------- Tindak lanjut stok ----------

func tindakLanjutValid() TindakLanjutInput {
	return TindakLanjutInput{
		BatchReceiptID:     1,
		Jenis:              "PEMUSNAHAN",
		SumberTindakLanjut: "INTERNAL_APOTEK",
		Alasan:             "KEDALUWARSA",
		QtyDasar:           10,
		NilaiKompensasi:    0,
	}
}

func TestValidasiTindakLanjut_Valid(t *testing.T) {
	in, err := validasiTindakLanjut(tindakLanjutValid())
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if in.Jenis != jenisPemusnahan || in.QtyDasar != 10 {
		t.Fatalf("hasil tidak sesuai: %+v", in)
	}
}

func TestValidasiTindakLanjut_Dirapikan(t *testing.T) {
	in := tindakLanjutValid()
	in.Jenis = " retur_pbf "
	in.SumberTindakLanjut = "produsen_pbf"
	in.Alasan = "  rusak "
	in.QtyDasar = 2.0004
	in.NilaiKompensasi = 1234.567
	in.Catatan = "  dikirim via ekspedisi  "

	got, err := validasiTindakLanjut(in)
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if got.Jenis != jenisReturPBF || got.SumberTindakLanjut != sumberProdusenPBF {
		t.Fatalf("jenis/sumber tidak dinormalisasi: %+v", got)
	}
	if got.Alasan != "RUSAK" {
		t.Fatalf("alasan umum seharusnya disamakan hurufnya, dapat %q", got.Alasan)
	}
	if got.QtyDasar != 2 || got.NilaiKompensasi != 1234.57 {
		t.Fatalf("pembulatan salah: qty %v, kompensasi %v", got.QtyDasar, got.NilaiKompensasi)
	}
	if got.Catatan != "dikirim via ekspedisi" {
		t.Fatalf("catatan tidak dirapikan: %q", got.Catatan)
	}

	// Alasan di luar daftar umum dibiarkan apa adanya.
	in.Alasan = "Kontaminasi"
	got, _ = validasiTindakLanjut(in)
	if got.Alasan != "Kontaminasi" {
		t.Fatalf("alasan lain seharusnya tetap, dapat %q", got.Alasan)
	}
}

func TestValidasiTindakLanjut_Tolak(t *testing.T) {
	kasus := []struct {
		nama string
		ubah func(in *TindakLanjutInput)
	}{
		{"receipt id nol", func(in *TindakLanjutInput) { in.BatchReceiptID = 0 }},
		{"jenis kosong", func(in *TindakLanjutInput) { in.Jenis = "" }},
		{"jenis ngawur", func(in *TindakLanjutInput) { in.Jenis = "BUANG" }},
		{"sumber kosong", func(in *TindakLanjutInput) { in.SumberTindakLanjut = "" }},
		{"sumber ngawur", func(in *TindakLanjutInput) { in.SumberTindakLanjut = "TETANGGA" }},
		{"alasan kosong", func(in *TindakLanjutInput) { in.Alasan = "  " }},
		{"alasan terlalu panjang", func(in *TindakLanjutInput) { in.Alasan = strings.Repeat("a", 101) }},
		{"qty nol", func(in *TindakLanjutInput) { in.QtyDasar = 0 }},
		{"qty negatif", func(in *TindakLanjutInput) { in.QtyDasar = -1 }},
		{"qty terlalu kecil (dibulatkan jadi 0)", func(in *TindakLanjutInput) { in.QtyDasar = 0.0001 }},
		{"qty NaN", func(in *TindakLanjutInput) { in.QtyDasar = math.NaN() }},
		{"qty tak hingga", func(in *TindakLanjutInput) { in.QtyDasar = math.Inf(1) }},
		{"kompensasi negatif", func(in *TindakLanjutInput) { in.NilaiKompensasi = -1 }},
		{"kompensasi NaN", func(in *TindakLanjutInput) { in.NilaiKompensasi = math.NaN() }},
		{"kompensasi tak hingga", func(in *TindakLanjutInput) { in.NilaiKompensasi = math.Inf(1) }},
		{"kompensasi terlalu besar", func(in *TindakLanjutInput) { in.NilaiKompensasi = 1e14 }},
		{"catatan terlalu panjang", func(in *TindakLanjutInput) { in.Catatan = strings.Repeat("x", 1001) }},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			in := tindakLanjutValid()
			k.ubah(&in)
			if _, err := validasiTindakLanjut(in); !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}

func TestCatatTindakLanjut_ValidasiSebelumDatabase(t *testing.T) {
	// Validasi berjalan sebelum menyentuh database, jadi repo boleh nil.
	in := tindakLanjutValid()
	in.QtyDasar = 0
	if _, err := serviceUji().CatatTindakLanjut(context.Background(), 1, in); !errors.Is(err, ErrValidasi) {
		t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
	}
}

func TestSiapkanFilterTindakLanjut(t *testing.T) {
	q, page, perPage, err := siapkanFilterTindakLanjut(FilterTindakLanjut{
		Jenis: "retur_pbf", BatchID: 3, Dari: "2026-11-01", Sampai: "2026-11-30", Page: 2, PerPage: 10,
	})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if q.Jenis != jenisReturPBF || q.BatchID != 3 || page != 2 || perPage != 10 || q.Offset != 10 {
		t.Fatalf("hasil salah: %+v page %d perPage %d", q, page, perPage)
	}

	kasus := []FilterTindakLanjut{
		{Jenis: "BUANG"},
		{BatchID: -1},
		{Dari: "kemarin"},
		{Sampai: "07/11/2026"},
		{Dari: "2026-12-01", Sampai: "2026-11-01"},
		{Page: -1},
	}
	for i, f := range kasus {
		if _, _, _, err := siapkanFilterTindakLanjut(f); !errors.Is(err, ErrValidasi) {
			t.Fatalf("kasus %d seharusnya ErrValidasi, dapat: %v", i, err)
		}
	}
}

func TestAngkaForm(t *testing.T) {
	buat := func(nilai string) *http.Request {
		form := neturl.Values{"x": {nilai}}
		req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}

	if n, err := angkaDesimalForm(buat("12.5"), "x"); err != nil || n != 12.5 {
		t.Fatalf("12.5 salah: %v %v", n, err)
	}
	if n, err := angkaDesimalForm(buat(""), "x"); err != nil || n != 0 {
		t.Fatalf("kosong seharusnya 0: %v %v", n, err)
	}
	for _, buruk := range []string{"abc", "1,5", "NaN", "Inf", "-Inf", "+Inf"} {
		if _, err := angkaDesimalForm(buat(buruk), "x"); !errors.Is(err, ErrValidasi) {
			t.Fatalf("%q seharusnya ErrValidasi, dapat %v", buruk, err)
		}
	}
	if n, err := angkaBulatForm(buat(" 42 "), "x"); err != nil || n != 42 {
		t.Fatalf("42 salah: %v %v", n, err)
	}
	if _, err := angkaBulatForm(buat("4.2"), "x"); !errors.Is(err, ErrValidasi) {
		t.Fatalf("4.2 seharusnya ErrValidasi, dapat %v", err)
	}
}
