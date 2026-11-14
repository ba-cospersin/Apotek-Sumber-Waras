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