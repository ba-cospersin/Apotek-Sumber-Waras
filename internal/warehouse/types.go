package warehouse

import (
	"errors"
	"fmt"
	"time"
)

// Format tanggal yang dipakai untuk input/output (YYYY-MM-DD).
const layoutTanggal = "2006-01-02"

// Nilai status_batch pada tabel batches.
const (
	statusBatchAktif    = "AKTIF"
	statusBatchDiblokir = "DIBLOKIR"
)

// Error yang dikenali handler untuk menentukan kode HTTP.
var (
	// ErrValidasi: input tidak valid -> HTTP 400.
	ErrValidasi = errors.New("data tidak valid")

	// Error konflik dengan data yang sudah ada -> HTTP 409.
	ErrFakturDuplikat  = errors.New("nomor faktur sudah pernah dicatat untuk PBF ini")
	ErrBatchDiblokir   = errors.New("batch sedang diblokir dan tidak bisa menerima stok")
	ErrKedaluwarsaBeda = errors.New("tanggal kedaluwarsa berbeda dengan batch yang sudah tercatat")
)

// validasi membuat error validasi dengan pesan yang bisa
// langsung ditampilkan ke pengguna.
func validasi(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidasi, fmt.Sprintf(format, args...))
}

// ---------- Input penerimaan barang ----------

// ItemPenerimaanInput adalah satu baris obat pada faktur PBF.
//
// Qty dan HargaBeli dinyatakan dalam satuan yang dipilih (UnitID).
// Jika UnitID = 0, dianggap satuan dasar obat.
type ItemPenerimaanInput struct {
	MedicineID    int64   `json:"medicine_id"`
	UnitID        int64   `json:"unit_id"`
	NoBatch       string  `json:"no_batch"`
	TglKadaluarsa string  `json:"tgl_kadaluarsa"` // YYYY-MM-DD
	Qty           float64 `json:"qty"`
	HargaBeli     float64 `json:"harga_beli"` // per satuan yang dipilih
}

// PenerimaanInput adalah satu faktur PBF beserta item-itemnya.
type PenerimaanInput struct {
	SupplierID int64                 `json:"supplier_id"`
	NoFaktur   string                `json:"no_faktur"`
	TglFaktur  string                `json:"tgl_faktur"` // YYYY-MM-DD
	Items      []ItemPenerimaanInput `json:"items"`
}

// ---------- Hasil penerimaan barang ----------

// ItemPenerimaan adalah lapisan penerimaan yang berhasil dibuat.
// Semua jumlah dan harga sudah dalam satuan dasar.
type ItemPenerimaan struct {
	ReceiptID         int64   `json:"receipt_id"`
	BatchID           int64   `json:"batch_id"`
	MedicineID        int64   `json:"medicine_id"`
	NoBatch           string  `json:"no_batch"`
	TglKadaluarsa     string  `json:"tgl_kadaluarsa"`
	QtyDasar          float64 `json:"qty_dasar"`
	HargaBeliPerDasar float64 `json:"harga_beli_per_dasar"`
}

// Penerimaan adalah hasil pencatatan satu faktur.
type Penerimaan struct {
	PurchaseID int64            `json:"purchase_id"`
	SupplierID int64            `json:"supplier_id"`
	NoFaktur   string           `json:"no_faktur"`
	TglFaktur  string           `json:"tgl_faktur"`
	Total      float64          `json:"total"`
	Items      []ItemPenerimaan `json:"items"`
}

// ---------- Data hasil query (dipakai antara repository dan service) ----------

type Medicine struct {
	ID       int64
	IsActive bool
}

type Unit struct {
	ID                int64
	IsiPerSatuanDasar float64
	IsActive          bool
}

type Batch struct {
	ID            int64
	TglKadaluarsa time.Time
	Status        string
}
