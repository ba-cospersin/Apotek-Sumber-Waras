package warehouse

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ---------- Mode respons ----------

// modeRespon menentukan bentuk balasan untuk endpoint yang menerima form.
//
//   - responHalaman: browser mengirim form biasa (Accept: text/html).
//     Hasilnya redirect ke halaman dengan pesan di query string.
//   - responJSON: kode JavaScript (fetch) meminta JSON (Accept: application/json).
//     Hasilnya JSON, dan halaman tetap di tempat sehingga isian form tidak hilang.
//   - responAPI: tanpa preferensi (curl, tes). Perilaku bawaan: redirect ke
//     detail JSON jika sukses, dan galat berupa JSON.
type modeRespon int

const (
	responAPI modeRespon = iota
	responJSON
	responHalaman
)

func modeResponDari(r *http.Request) modeRespon {
	accept := r.Header.Get("Accept")
	switch {
	case strings.Contains(accept, "text/html"):
		return responHalaman
	case strings.Contains(accept, "application/json"):
		return responJSON
	default:
		return responAPI
	}
}

// halamanKembali adalah daftar halaman yang boleh menjadi tujuan redirect.
// Daftar ini sengaja tertutup supaya parameter "kembali" tidak bisa dipakai
// untuk mengarahkan pengguna ke alamat lain (open redirect).
var halamanKembali = map[string]bool{
	"/penerimaan":   true,
	"/cek-stok":     true,
	"/kedaluwarsa":  true,
	"/stock-opname": true,
}

// halamanKembaliDari membaca field form "kembali"; jika tidak ada atau tidak
// ada di daftar, dipakai bawaan.
func halamanKembaliDari(r *http.Request, bawaan string) string {
	k := strings.TrimSpace(r.PostFormValue("kembali"))
	if halamanKembali[k] {
		return k
	}
	return bawaan
}

const panjangPesanMaks = 300

// alamatDenganPesan membentuk alamat halaman dengan pesan di query string.
// kunci adalah "ok" untuk berhasil atau "galat" untuk gagal.
func alamatDenganPesan(halaman, kunci, pesan string) string {
	if r := []rune(pesan); len(r) > panjangPesanMaks {
		pesan = string(r[:panjangPesanMaks]) + "…"
	}
	return halaman + "?" + kunci + "=" + url.QueryEscape(pesan)
}

// klasifikasiGalat menentukan kode HTTP dan pesan untuk pengguna.
// Galat tak terduga dicatat di log dan pesannya dibuat umum.
func klasifikasiGalat(err error) (int, string) {
	switch {
	case errors.Is(err, ErrValidasi):
		return http.StatusBadRequest, err.Error()

	case errors.Is(err, ErrFakturDuplikat),
		errors.Is(err, ErrKedaluwarsaBeda),
		errors.Is(err, ErrSudahDiblokir),
		errors.Is(err, ErrStokTidakCukup):
		return http.StatusConflict, err.Error()

	case errors.Is(err, ErrTidakDitemukan):
		return http.StatusNotFound, err.Error()

	default:
		log.Printf("warehouse: kesalahan tak terduga: %v", err)
		return http.StatusInternalServerError, "Terjadi kesalahan pada server"
	}
}

// pesanUntukPengguna merapikan pesan galat untuk ditampilkan di halaman.
func pesanUntukPengguna(pesan string) string {
	return strings.TrimPrefix(pesan, ErrValidasi.Error()+": ")
}

// galatForm membalas kegagalan sesuai mode respons.
func (h *Handler) galatForm(w http.ResponseWriter, r *http.Request, err error, kembali string) {
	status, pesan := klasifikasiGalat(err)
	if modeResponDari(r) == responHalaman {
		http.Redirect(w, r, alamatDenganPesan(kembali, "galat", pesanUntukPengguna(pesan)), http.StatusSeeOther)
		return
	}
	writeError(w, status, pesan)
}

// suksesForm membalas keberhasilan sesuai mode respons.
//
//	responHalaman: redirect ke kembali?ok=pesan
//	responJSON   : 201 dengan {"data": data}
//	responAPI    : redirect ke alamatAPI (detail JSON)
func (h *Handler) suksesForm(
	w http.ResponseWriter,
	r *http.Request,
	kembali, pesan, alamatAPI string,
	data any,
) {
	switch modeResponDari(r) {
	case responHalaman:
		http.Redirect(w, r, alamatDenganPesan(kembali, "ok", pesan), http.StatusSeeOther)
	case responJSON:
		writeJSON(w, http.StatusCreated, map[string]any{"data": data})
	default:
		http.Redirect(w, r, alamatAPI, http.StatusSeeOther)
	}
}

// ---------- Form penerimaan ----------

// kolomBarisPenerimaan adalah nama field yang diulang untuk tiap baris obat,
// dalam urutan tetap (supaya pesan galat selalu sama).
var kolomBarisPenerimaan = []string{"medicine_id", "unit_id", "no_batch", "tgl_kadaluarsa", "qty", "harga_beli"}

// penerimaanDariForm membaca form penerimaan.
//
// Field untuk seluruh faktur: supplier_id, no_faktur, tgl_faktur, tgl_terima.
// Field untuk tiap baris obat (nama yang sama DIULANG untuk tiap baris):
// medicine_id, unit_id, no_batch, tgl_kadaluarsa, qty, harga_beli.
// Field lain (misalnya "kembali") diabaikan.
func penerimaanDariForm(r *http.Request) (PenerimaanInput, error) {
	var in PenerimaanInput

	supplierID, err := bulatDariTeks(r.PostFormValue("supplier_id"), "supplier_id")
	if err != nil {
		return in, err
	}
	in.SupplierID = supplierID
	in.NoFaktur = r.PostFormValue("no_faktur")
	in.TglFaktur = r.PostFormValue("tgl_faktur")
	in.TglTerima = r.PostFormValue("tgl_terima")

	kolom := map[string][]string{}
	panjang := 0
	for _, nama := range kolomBarisPenerimaan {
		nilai := r.PostForm[nama]
		kolom[nama] = nilai
		if len(nilai) > panjang {
			panjang = len(nilai)
		}
	}
	if panjang == 0 {
		return in, validasi("items minimal berisi satu obat")
	}

	// Setiap kolom harus punya isian untuk SETIAP baris. Input yang diberi
	// atribut disabled tidak ikut terkirim, sehingga kolomnya kurang.
	seragam := true
	rincian := make([]string, 0, len(kolomBarisPenerimaan))
	for _, nama := range kolomBarisPenerimaan {
		rincian = append(rincian, fmt.Sprintf("%s=%d", nama, len(kolom[nama])))
		if len(kolom[nama]) != panjang {
			seragam = false
		}
	}
	if !seragam {
		return in, validasi(
			"jumlah isian tiap kolom baris tidak sama (%s); pastikan setiap baris memiliki semua kolom dan tidak ada input yang dinonaktifkan",
			strings.Join(rincian, ", "),
		)
	}

	in.Items = make([]ItemPenerimaanInput, 0, panjang)
	for i := 0; i < panjang; i++ {
		nomor := i + 1

		medicineID, err := bulatDariTeks(kolom["medicine_id"][i], fmt.Sprintf("item %d: medicine_id", nomor))
		if err != nil {
			return in, err
		}
		unitID, err := bulatDariTeks(kolom["unit_id"][i], fmt.Sprintf("item %d: unit_id", nomor))
		if err != nil {
			return in, err
		}
		qty, err := desimalDariTeks(kolom["qty"][i], fmt.Sprintf("item %d: qty", nomor))
		if err != nil {
			return in, err
		}
		harga, err := desimalDariTeks(kolom["harga_beli"][i], fmt.Sprintf("item %d: harga_beli", nomor))
		if err != nil {
			return in, err
		}

		in.Items = append(in.Items, ItemPenerimaanInput{
			MedicineID:    medicineID,
			UnitID:        unitID,
			NoBatch:       kolom["no_batch"][i],
			TglKadaluarsa: kolom["tgl_kadaluarsa"][i],
			Qty:           qty,
			HargaBeli:     harga,
		})
	}

	return in, nil
}

// bulatDariTeks mengubah teks menjadi bilangan bulat. Kosong dianggap 0.
func bulatDariTeks(teks, nama string) (int64, error) {
	teks = strings.TrimSpace(teks)
	if teks == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(teks, 10, 64)
	if err != nil {
		return 0, validasi("%s harus berupa angka bulat", nama)
	}
	return n, nil
}

// desimalDariTeks mengubah teks menjadi angka desimal (titik sebagai pemisah).
// Kosong dianggap 0 (nanti ditolak service jika 0 tidak sah). NaN dan tak
// hingga ditolak.
func desimalDariTeks(teks, nama string) (float64, error) {
	teks = strings.TrimSpace(teks)
	if teks == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(teks, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, validasi("%s harus berupa angka (gunakan titik sebagai desimal)", nama)
	}
	return n, nil
}
