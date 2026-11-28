package warehouse

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	neturl "net/url"
	"strings"
	"testing"
	"time"
)

func formReq(t *testing.T, v neturl.Values) *http.Request {
	t.Helper()
	req := httptest.NewRequest("POST", "/penerimaan", strings.NewReader(v.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	return req
}

func TestModeResponDari(t *testing.T) {
	kasus := []struct {
		accept string
		harap  modeRespon
	}{
		{"", responAPI},
		{"*/*", responAPI},
		{"application/json", responJSON},
		{"application/json, text/plain, */*", responJSON},
		{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", responHalaman},
		{"text/html, application/json", responHalaman}, // browser menang
	}
	for _, k := range kasus {
		req := httptest.NewRequest("GET", "/", nil)
		if k.accept != "" {
			req.Header.Set("Accept", k.accept)
		}
		if got := modeResponDari(req); got != k.harap {
			t.Fatalf("Accept %q: dapat %d, seharusnya %d", k.accept, got, k.harap)
		}
	}
}

func TestHalamanKembaliDari(t *testing.T) {
	kasus := map[string]string{
		"":                       "/bawaan",
		"/cek-stok":              "/cek-stok",
		"/kedaluwarsa":           "/kedaluwarsa",
		"/stock-opname":          "/stock-opname",
		"/penerimaan":            "/penerimaan",
		"  /cek-stok  ":          "/cek-stok",
		"https://jahat.example":  "/bawaan",
		"//jahat.example":        "/bawaan",
		"/\\jahat.example":       "/bawaan",
		"/penerimaan?x=1":        "/bawaan",
		"/penerimaan/":           "/bawaan",
		"/PENERIMAAN":            "/bawaan",
		"/dashboard":             "/bawaan", // bukan halaman gudang
		"javascript:alert(1)":    "/bawaan",
		"/penerimaan\r\nX: y":    "/bawaan",
		"/cek-stok%0d%0aX:%20y":  "/bawaan",
		"/data-master":           "/bawaan",
		"/../penerimaan":         "/bawaan",
		"/cek-stok#fragmen":      "/bawaan",
		"/stock-opname/../login": "/bawaan",
	}
	for masuk, harap := range kasus {
		req := formReq(t, neturl.Values{"kembali": {masuk}})
		if got := halamanKembaliDari(req, "/bawaan"); got != harap {
			t.Fatalf("kembali %q: dapat %q, seharusnya %q", masuk, got, harap)
		}
	}
}

func TestAlamatDenganPesan(t *testing.T) {
	got := alamatDenganPesan("/cek-stok", "ok", "Batch berhasil diblokir.")
	if got != "/cek-stok?ok=Batch+berhasil+diblokir." {
		t.Fatalf("alamat salah: %s", got)
	}

	// Karakter khusus harus di-escape, supaya tidak bisa menyisipkan parameter lain.
	got = alamatDenganPesan("/penerimaan", "galat", "a&ok=1#x?y")
	u, err := neturl.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("galat") != "a&ok=1#x?y" || u.Query().Get("ok") != "" || u.Fragment != "" {
		t.Fatalf("pesan tidak di-escape dengan benar: %s", got)
	}

	// Pesan panjang dipotong.
	panjang := strings.Repeat("é", 500)
	u, _ = neturl.Parse(alamatDenganPesan("/penerimaan", "galat", panjang))
	if n := len([]rune(u.Query().Get("galat"))); n != panjangPesanMaks+1 {
		t.Fatalf("pesan seharusnya dipotong jadi %d karakter, dapat %d", panjangPesanMaks+1, n)
	}
}

func TestKlasifikasiGalat(t *testing.T) {
	kasus := []struct {
		err    error
		status int
	}{
		{validasi("x"), 400},
		{fmt.Errorf("lapisan: %w", ErrStokTidakCukup), 409},
		{ErrFakturDuplikat, 409},
		{fmt.Errorf("item 1: %w", ErrKedaluwarsaBeda), 409},
		{ErrSudahDiblokir, 409},
		{fmt.Errorf("batch 9: %w", ErrTidakDitemukan), 404},
		{errors.New("kesalahan database rahasia"), 500},
	}
	for _, k := range kasus {
		status, pesan := klasifikasiGalat(k.err)
		if status != k.status {
			t.Fatalf("%v: status %d, seharusnya %d", k.err, status, k.status)
		}
		if status == 500 && (strings.Contains(pesan, "rahasia") || pesan == "") {
			t.Fatalf("galat 500 tidak boleh membocorkan detail: %q", pesan)
		}
	}
}

func TestPesanUntukPengguna(t *testing.T) {
	if got := pesanUntukPengguna(validasi("alasan wajib diisi").Error()); got != "alasan wajib diisi" {
		t.Fatalf("awalan seharusnya dibuang: %q", got)
	}
	if got := pesanUntukPengguna("Terjadi kesalahan pada server"); got != "Terjadi kesalahan pada server" {
		t.Fatalf("pesan lain tidak boleh berubah: %q", got)
	}
}

func formPenerimaanValid() neturl.Values {
	return neturl.Values{
		"supplier_id":    {"1"},
		"no_faktur":      {"INV-1"},
		"tgl_faktur":     {"2026-11-05"},
		"tgl_terima":     {"2026-11-06"},
		"medicine_id":    {"1", "2"},
		"unit_id":        {"2", ""},
		"no_batch":       {"B1", "B2"},
		"tgl_kadaluarsa": {"2028-01-01", "2028-02-01"},
		"qty":            {"3", "10.5"},
		"harga_beli":     {"5000", "120.25"},
		"kembali":        {"/penerimaan"},
		"total_invoice":  {"999999"}, // field berlebih harus diabaikan
	}
}

func TestPenerimaanDariForm_Valid(t *testing.T) {
	in, err := penerimaanDariForm(formReq(t, formPenerimaanValid()))
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if in.SupplierID != 1 || in.NoFaktur != "INV-1" || in.TglFaktur != "2026-11-05" || in.TglTerima != "2026-11-06" || len(in.Items) != 2 {
		t.Fatalf("header salah: %+v", in)
	}
	a, b := in.Items[0], in.Items[1]
	if a.MedicineID != 1 || a.UnitID != 2 || a.NoBatch != "B1" || a.TglKadaluarsa != "2028-01-01" || a.Qty != 3 || a.HargaBeli != 5000 {
		t.Fatalf("baris 1 salah: %+v", a)
	}
	if b.MedicineID != 2 || b.UnitID != 0 || b.Qty != 10.5 || b.HargaBeli != 120.25 {
		t.Fatalf("baris 2 salah (unit_id kosong harus 0 = satuan dasar): %+v", b)
	}
}

func TestPenerimaanDariForm_Tolak(t *testing.T) {
	kasus := []struct {
		nama string
		ubah func(v neturl.Values)
		sub  string // potongan pesan yang diharapkan
	}{
		{"tanpa satu pun baris", func(v neturl.Values) {
			for _, k := range []string{"medicine_id", "unit_id", "no_batch", "tgl_kadaluarsa", "qty", "harga_beli"} {
				v.Del(k)
			}
		}, "minimal berisi satu obat"},
		{"kolom qty hilang (input disabled)", func(v neturl.Values) { v.Del("qty") }, "qty=0"},
		{"kolom unit_id kurang satu baris", func(v neturl.Values) { v["unit_id"] = []string{"2"} }, "unit_id=1"},
		{"kolom no_batch lebih banyak", func(v neturl.Values) { v["no_batch"] = []string{"a", "b", "c"} }, "no_batch=3"},
		{"supplier bukan angka", func(v neturl.Values) { v.Set("supplier_id", "PBF Satu") }, "supplier_id harus berupa angka bulat"},
		{"medicine_id desimal", func(v neturl.Values) { v["medicine_id"] = []string{"1.5", "2"} }, "item 1: medicine_id"},
		{"unit_id huruf", func(v neturl.Values) { v["unit_id"] = []string{"2", "box"} }, "item 2: unit_id"},
		{"qty koma", func(v neturl.Values) { v["qty"] = []string{"3", "10,5"} }, "item 2: qty"},
		{"qty NaN", func(v neturl.Values) { v["qty"] = []string{"NaN", "1"} }, "item 1: qty"},
		{"harga tak hingga", func(v neturl.Values) { v["harga_beli"] = []string{"1", "Inf"} }, "item 2: harga_beli"},
	}
	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			v := formPenerimaanValid()
			k.ubah(v)
			_, err := penerimaanDariForm(formReq(t, v))
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
			if !strings.Contains(err.Error(), k.sub) {
				t.Fatalf("pesan %q seharusnya memuat %q", err.Error(), k.sub)
			}
		})
	}
}

func TestPenerimaanDariForm_SupplierKosongDiserahkanKeService(t *testing.T) {
	// supplier_id kosong terbaca 0; service-lah yang menolaknya dengan pesan yang jelas.
	v := formPenerimaanValid()
	v.Del("supplier_id")
	in, err := penerimaanDariForm(formReq(t, v))
	if err != nil || in.SupplierID != 0 {
		t.Fatalf("supplier_id kosong seharusnya terbaca 0 tanpa error: %+v %v", in, err)
	}
	if _, err := serviceUji().validasiPenerimaan(in); !errors.Is(err, ErrValidasi) {
		t.Fatalf("service seharusnya menolak supplier_id 0, dapat %v", err)
	}
}

func TestSiapkanFilterBatch_Bermasalah(t *testing.T) {
	hariIni := time.Date(2026, 11, 7, 0, 0, 0, 0, time.Local)
	q, _, _, err := siapkanFilterBatch(FilterBatch{Status: "bermasalah"}, hariIni)
	if err != nil || q.Status != statusJualBermasalah || q.Batas != "2026-12-07" {
		t.Fatalf("filter bermasalah salah: %+v %v", q, err)
	}
}
