package warehouse

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
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

// ---------- Stock opname ----------

func lap(id int64, sisa float64) LapisanStok { return LapisanStok{ID: id, QtySisa: sisa} }

func TestAlokasiSelisih(t *testing.T) {
	type alok struct {
		id    int64
		milli int64
	}
	kasus := []struct {
		nama     string
		lapisan  []LapisanStok // urut dari yang paling dahulu diterima
		selisih  int64         // milli
		harapan  []alok
		harapErr bool
	}{
		{"tanpa selisih", []LapisanStok{lap(1, 100)}, 0, nil, false},
		{"kurang, cukup di lapisan tertua", []LapisanStok{lap(1, 100), lap(2, 50)}, -30000, []alok{{1, -30000}}, false},
		{"kurang pas sebesar lapisan tertua", []LapisanStok{lap(1, 100), lap(2, 50)}, -100000, []alok{{1, -100000}}, false},
		{"kurang melewati lapisan tertua", []LapisanStok{lap(1, 100), lap(2, 50)}, -120000, []alok{{1, -100000}, {2, -20000}}, false},
		{"kurang sebesar seluruh stok", []LapisanStok{lap(1, 100), lap(2, 50)}, -150000, []alok{{1, -100000}, {2, -50000}}, false},
		{"lapisan kosong dilewati", []LapisanStok{lap(1, 10), lap(2, 0), lap(3, 20)}, -25000, []alok{{1, -10000}, {3, -15000}}, false},
		{"lapisan tertua kosong dilewati", []LapisanStok{lap(1, 0), lap(2, 40)}, -5000, []alok{{2, -5000}}, false},
		{"lebih masuk ke lapisan terakhir", []LapisanStok{lap(1, 100), lap(2, 50)}, 60000, []alok{{2, 60000}}, false},
		{"lebih, lapisan terakhir sisanya 0", []LapisanStok{lap(1, 100), lap(2, 0)}, 5000, []alok{{2, 5000}}, false},
		{"lebih, satu lapisan saja", []LapisanStok{lap(1, 7)}, 1, []alok{{1, 1}}, false},
		{"desimal tanpa galat", []LapisanStok{lap(1, 0.1), lap(2, 0.2)}, -200, []alok{{1, -100}, {2, -100}}, false},
		{"desimal 0.1 + 0.2 tepat 0.3", []LapisanStok{lap(1, 0.1), lap(2, 0.2)}, -300, []alok{{1, -100}, {2, -200}}, false},
		{"kurang melebihi seluruh stok", []LapisanStok{lap(1, 10)}, -10001, nil, true},
		{"tanpa lapisan", nil, -1, nil, true},
		{"tanpa lapisan dan tanpa selisih", nil, 0, nil, false},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			got, err := alokasiSelisih(k.lapisan, k.selisih)
			if k.harapErr {
				if err == nil {
					t.Fatalf("seharusnya error, dapat %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("tidak seharusnya error: %v", err)
			}
			if len(got) != len(k.harapan) {
				t.Fatalf("jumlah penyesuaian %d, seharusnya %d: %+v", len(got), len(k.harapan), got)
			}
			var total int64
			for i, g := range got {
				if g.lapisan.ID != k.harapan[i].id || g.milli != k.harapan[i].milli {
					t.Fatalf("penyesuaian %d: dapat (id %d, %d), seharusnya (id %d, %d)",
						i, g.lapisan.ID, g.milli, k.harapan[i].id, k.harapan[i].milli)
				}
				total += g.milli
			}
			// Aturan dokumen: total penyesuaian SAMA dengan selisih.
			if total != k.selisih {
				t.Fatalf("total penyesuaian %d, seharusnya sama dengan selisih %d", total, k.selisih)
			}
		})
	}
}

func TestAlokasiSelisih_TidakPernahMembuatStokNegatif(t *testing.T) {
	lapisan := []LapisanStok{lap(1, 3.333), lap(2, 0), lap(3, 12.5), lap(4, 0.001)}
	var sistem int64
	for _, l := range lapisan {
		sistem += keMilli(l.QtySisa)
	}

	// Semua kemungkinan hasil hitung fisik dari 0 sampai sistem.
	for fisik := int64(0); fisik <= sistem; fisik += 7 {
		got, err := alokasiSelisih(lapisan, fisik-sistem)
		if err != nil {
			t.Fatalf("fisik %d: %v", fisik, err)
		}
		sisa := map[int64]int64{}
		for _, l := range lapisan {
			sisa[l.ID] = keMilli(l.QtySisa)
		}
		var total int64
		for _, g := range got {
			sisa[g.lapisan.ID] += g.milli
			total += g.milli
			if sisa[g.lapisan.ID] < 0 {
				t.Fatalf("fisik %d: lapisan %d menjadi negatif", fisik, g.lapisan.ID)
			}
		}
		if total != fisik-sistem {
			t.Fatalf("fisik %d: total %d bukan selisih %d", fisik, total, fisik-sistem)
		}
	}
}

func TestKeMilli(t *testing.T) {
	if keMilli(0.1+0.2) != 300 || keMilli(1189) != 1189000 || keMilli(0.0004) != 0 || keMilli(2.0005) != 2001 {
		t.Fatalf("keMilli salah: %d %d %d %d", keMilli(0.1+0.2), keMilli(1189), keMilli(0.0004), keMilli(2.0005))
	}
	if dariMilli(1234) != 1.234 {
		t.Fatalf("dariMilli salah: %v", dariMilli(1234))
	}
}

func TestStatusSelisih(t *testing.T) {
	if statusSelisih(100, 100) != selisihSesuai || statusSelisih(100, 99.999) != selisihKurang ||
		statusSelisih(100, 100.001) != selisihLebih || statusSelisih(0.1+0.2, 0.3) != selisihSesuai {
		t.Fatalf("statusSelisih salah")
	}
}

func TestIsiTurunanOpname(t *testing.T) {
	o := &StockOpname{
		QtySistem: 150, QtyFisik: 40,
		Penyesuaian: []PenyesuaianLapisan{
			{QtyPenyesuaian: -100, HargaBeliPerDasar: 500},
			{QtyPenyesuaian: -10, HargaBeliPerDasar: 520},
		},
	}
	isiTurunanOpname(o)
	if o.Selisih != -110 || o.StatusSelisih != selisihKurang || o.NilaiSelisih != -55200 {
		t.Fatalf("turunan salah: %+v", o)
	}
}

func TestValidasiStockOpname_Valid(t *testing.T) {
	in, waktu, err := serviceUji().validasiStockOpname(StockOpnameInput{BatchID: 1, QtyFisik: 0, Catatan: "  habis  "})
	if err != nil {
		t.Fatalf("qty_fisik 0 harus valid (stok bisa habis): %v", err)
	}
	if in.QtyFisik != 0 || in.Catatan != "habis" {
		t.Fatalf("input tidak dirapikan: %+v", in)
	}
	if waktu.Format(layoutTanggal) != "2026-11-07" {
		t.Fatalf("waktu bawaan salah: %v", waktu)
	}

	in, waktu, err = serviceUji().validasiStockOpname(StockOpnameInput{BatchID: 1, QtyFisik: 12.34567, TglOpname: "2026-11-05"})
	if err != nil || in.QtyFisik != 12.346 {
		t.Fatalf("pembulatan salah: %+v %v", in, err)
	}
	if waktu.Format(layoutTanggal) != "2026-11-05" || waktu.Hour() != 10 {
		t.Fatalf("tgl_opname salah: %v", waktu)
	}
}

func TestValidasiStockOpname_Tolak(t *testing.T) {
	kasus := []struct {
		nama string
		in   StockOpnameInput
	}{
		{"batch id nol", StockOpnameInput{BatchID: 0, QtyFisik: 1}},
		{"qty negatif", StockOpnameInput{BatchID: 1, QtyFisik: -0.001}},
		{"qty NaN", StockOpnameInput{BatchID: 1, QtyFisik: math.NaN()}},
		{"qty tak hingga", StockOpnameInput{BatchID: 1, QtyFisik: math.Inf(1)}},
		{"qty terlalu besar", StockOpnameInput{BatchID: 1, QtyFisik: 1e13}},
		{"catatan terlalu panjang", StockOpnameInput{BatchID: 1, QtyFisik: 1, Catatan: strings.Repeat("x", 1001)}},
		{"tanggal salah format", StockOpnameInput{BatchID: 1, QtyFisik: 1, TglOpname: "kemarin"}},
		{"tanggal masa depan", StockOpnameInput{BatchID: 1, QtyFisik: 1, TglOpname: "2026-11-08"}},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			if _, _, err := serviceUji().validasiStockOpname(k.in); !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}

func TestCatatStockOpname_ValidasiSebelumDatabase(t *testing.T) {
	// Validasi berjalan sebelum menyentuh database, jadi repo boleh nil.
	_, err := serviceUji().CatatStockOpname(context.Background(), 1, StockOpnameInput{BatchID: 1, QtyFisik: -1})
	if !errors.Is(err, ErrValidasi) {
		t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
	}
	if _, err := serviceUji().DetailStockOpname(context.Background(), 0); !errors.Is(err, ErrValidasi) {
		t.Fatalf("id nol seharusnya ErrValidasi, dapat: %v", err)
	}
}

func TestSiapkanFilterStockOpname(t *testing.T) {
	q, page, perPage, err := siapkanFilterStockOpname(FilterStockOpname{
		BatchID: 4, Dari: "2026-11-01", Sampai: "2026-11-30", HanyaSelisih: true, Page: 2, PerPage: 10,
	})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if q.BatchID != 4 || !q.HanyaSelisih || page != 2 || perPage != 10 || q.Offset != 10 {
		t.Fatalf("hasil salah: %+v", q)
	}

	kasus := []FilterStockOpname{
		{BatchID: -1}, {Dari: "kemarin"}, {Sampai: "07/11/2026"},
		{Dari: "2026-12-01", Sampai: "2026-11-01"}, {Page: -1}, {PerPage: -1},
	}
	for i, f := range kasus {
		if _, _, _, err := siapkanFilterStockOpname(f); !errors.Is(err, ErrValidasi) {
			t.Fatalf("kasus %d seharusnya ErrValidasi, dapat: %v", i, err)
		}
	}
}

func TestAngkaDesimalWajibForm(t *testing.T) {
	buat := func(nilai string, ada bool) *http.Request {
		form := neturl.Values{}
		if ada {
			form.Set("x", nilai)
		}
		req := httptest.NewRequest("POST", "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		return req
	}

	if n, err := angkaDesimalWajibForm(buat("0", true), "x"); err != nil || n != 0 {
		t.Fatalf("0 seharusnya sah: %v %v", n, err)
	}
	if n, err := angkaDesimalWajibForm(buat("12.5", true), "x"); err != nil || n != 12.5 {
		t.Fatalf("12.5 salah: %v %v", n, err)
	}
	// Kosong atau tidak dikirim TIDAK boleh dianggap 0.
	for _, r := range []*http.Request{buat("", true), buat("   ", true), buat("", false)} {
		if _, err := angkaDesimalWajibForm(r, "x"); !errors.Is(err, ErrValidasi) {
			t.Fatalf("kosong seharusnya ErrValidasi, dapat %v", err)
		}
	}
	if _, err := angkaDesimalWajibForm(buat("NaN", true), "x"); !errors.Is(err, ErrValidasi) {
		t.Fatalf("NaN seharusnya ErrValidasi, dapat %v", err)
	}
}

func TestAdalahDeadlock(t *testing.T) {
	deadlock := &mysql.MySQLError{Number: 1213, Message: "Deadlock found when trying to get lock"}
	if !adalahDeadlock(deadlock) || !adalahDeadlock(fmt.Errorf("gagal: %w", deadlock)) {
		t.Fatalf("deadlock tidak terdeteksi")
	}
	if adalahDeadlock(nil) || adalahDeadlock(errors.New("biasa")) ||
		adalahDeadlock(&mysql.MySQLError{Number: 1062, Message: "duplicate"}) {
		t.Fatalf("error lain dianggap deadlock")
	}
}

func TestUlangiJikaDeadlock(t *testing.T) {
	deadlock := fmt.Errorf("gagal: %w", &mysql.MySQLError{Number: 1213, Message: "Deadlock"})
	ctx := context.Background()

	// Dua kali deadlock lalu berhasil.
	n := 0
	err := ulangiJikaDeadlock(ctx, func() error {
		n++
		if n < 3 {
			return deadlock
		}
		return nil
	})
	if err != nil || n != 3 {
		t.Fatalf("seharusnya berhasil pada percobaan ke-3: err=%v n=%d", err, n)
	}

	// Error biasa tidak diulang.
	n = 0
	biasa := errors.New("bukan deadlock")
	if err := ulangiJikaDeadlock(ctx, func() error { n++; return biasa }); !errors.Is(err, biasa) || n != 1 {
		t.Fatalf("error biasa seharusnya 1 kali: err=%v n=%d", err, n)
	}

	// Deadlock terus-menerus berhenti setelah batas percobaan.
	n = 0
	if err := ulangiJikaDeadlock(ctx, func() error { n++; return deadlock }); !adalahDeadlock(err) || n != maksPercobaanTx {
		t.Fatalf("seharusnya berhenti setelah %d percobaan: err=%v n=%d", maksPercobaanTx, err, n)
	}

	// Konteks dibatalkan: berhenti tanpa menunggu.
	ctxBatal, batal := context.WithCancel(ctx)
	batal()
	n = 0
	if err := ulangiJikaDeadlock(ctxBatal, func() error { n++; return deadlock }); !adalahDeadlock(err) || n != 1 {
		t.Fatalf("konteks batal seharusnya 1 percobaan: err=%v n=%d", err, n)
	}
}
