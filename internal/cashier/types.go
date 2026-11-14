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