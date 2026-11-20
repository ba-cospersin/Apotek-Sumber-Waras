package cashier

import (
	"errors"
	"fmt"
	"time"
)

var (
	// ErrValidasi: input tidak valid -> HTTP 400.
	ErrValidasi = errors.New("data tidak valid")

	// ErrStokTidakCukup: stok yang boleh dijual tidak mencukupi -> HTTP 409.
	ErrStokTidakCukup = errors.New("stok tidak mencukupi")

	// ErrTidakDitemukan: data yang diminta tidak ada -> HTTP 404.
	ErrTidakDitemukan = errors.New("data tidak ditemukan")
)

// validasi membuat error validasi dengan pesan yang bisa
// langsung ditampilkan ke pengguna.
func validasi(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidasi, fmt.Sprintf(format, args...))
}

// Lapisan adalah satu lapisan penerimaan yang SUDAH lolos syarat jual
// (tidak diblokir, belum mendekati kedaluwarsa). Penyaringan dilakukan di SQL.
type Lapisan struct {
	ReceiptID         int64
	TglKadaluarsa     time.Time
	TanggalTerima     time.Time
	QtySisa           float64 // satuan dasar
	HargaBeliPerDasar float64
}

// Alokasi adalah satu baris yang nanti disimpan ke sale_item_batches.
type Alokasi struct {
	ReceiptID         int64
	QtyDasar          float64
	HargaBeliPerDasar float64
}

// SatuanJual adalah satu satuan jual obat beserta konversi dan harganya.
type SatuanJual struct {
	ID                int64   `json:"id"`
	Nama              string  `json:"nama"`
	IsiPerSatuanDasar float64 `json:"isi_per_satuan_dasar"`
	HargaJual         float64 `json:"harga_jual"` // per satuan jual ini
}

// Obat adalah hasil pencarian untuk kasir.
type Obat struct {
	ID           int64        `json:"id"`
	Kode         string       `json:"kode"`
	Nama         string       `json:"nama"`
	SatuanDasar  string       `json:"satuan_dasar"`
	StokTersedia float64      `json:"stok_tersedia"` // satuan dasar, hanya yang boleh dijual
	Satuan       []SatuanJual `json:"satuan"`
}

// UnitObat adalah satuan jual beserta obat pemiliknya.
type UnitObat struct {
	SatuanJual
	MedicineID int64
	NamaObat   string
}

// ItemInput adalah satu baris keranjang dari kasir.
// Harga TIDAK dikirim dari browser; server menghitungnya dari database.
type ItemInput struct {
	UnitID int64   `json:"unit_id"`
	Qty    float64 `json:"qty"` // dalam satuan yang dipilih (UnitID)
}

// TransaksiInput adalah permintaan penyimpanan satu penjualan.
type TransaksiInput struct {
	MetodeBayar string      `json:"metode_bayar"` // TUNAI atau NONTUNAI
	JumlahBayar float64     `json:"jumlah_bayar"`
	Items       []ItemInput `json:"items"`
}

// HasilTransaksi adalah hasil penjualan yang berhasil disimpan.
type HasilTransaksi struct {
	SaleID      int64   `json:"sale_id"`
	Total       float64 `json:"total"`
	JumlahBayar float64 `json:"jumlah_bayar"`
	Kembalian   float64 `json:"kembalian"`
}