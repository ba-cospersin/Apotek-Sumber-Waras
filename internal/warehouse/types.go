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

// ---------- Riwayat penerimaan ----------

// ErrTidakDitemukan: data yang diminta tidak ada -> HTTP 404.
var ErrTidakDitemukan = errors.New("data tidak ditemukan")

// FilterPenerimaan adalah filter dari pengguna untuk daftar faktur.
// Nilai kosong/0 berarti filter tidak dipakai.
type FilterPenerimaan struct {
	SupplierID int64
	Dari       string // YYYY-MM-DD, berdasarkan tgl_faktur
	Sampai     string // YYYY-MM-DD, berdasarkan tgl_faktur
	Page       int
	PerPage    int
}

// QueryPenerimaan adalah filter yang sudah divalidasi, siap untuk SQL.
type QueryPenerimaan struct {
	SupplierID int64
	Dari       string
	Sampai     string
	Limit      int
	Offset     int
}

// RingkasanPenerimaan adalah satu baris pada daftar faktur.
type RingkasanPenerimaan struct {
	PurchaseID  int64   `json:"purchase_id"`
	SupplierID  int64   `json:"supplier_id"`
	NamaPBF     string  `json:"nama_pbf"`
	NoFaktur    string  `json:"no_faktur"`
	TglFaktur   string  `json:"tgl_faktur"`
	Total       float64 `json:"total"`
	JumlahItem  int     `json:"jumlah_item"`
	DicatatOleh string  `json:"dicatat_oleh"`
}

// Halaman adalah informasi paginasi.
type Halaman struct {
	Page         int `json:"page"`
	PerPage      int `json:"per_page"`
	Total        int `json:"total"`
	TotalHalaman int `json:"total_halaman"`
}

// DaftarPenerimaan adalah satu halaman daftar faktur.
type DaftarPenerimaan struct {
	Data []RingkasanPenerimaan `json:"data"`
	Meta Halaman               `json:"meta"`
}

// ItemDetailPenerimaan adalah satu lapisan penerimaan pada sebuah faktur.
// Semua jumlah dan harga dalam satuan dasar.
type ItemDetailPenerimaan struct {
	ReceiptID         int64   `json:"receipt_id"`
	BatchID           int64   `json:"batch_id"`
	MedicineID        int64   `json:"medicine_id"`
	KodeObat          string  `json:"kode_obat"`
	NamaObat          string  `json:"nama_obat"`
	SatuanDasar       string  `json:"satuan_dasar"`
	NoBatch           string  `json:"no_batch"`
	TglKadaluarsa     string  `json:"tgl_kadaluarsa"`
	StatusBatch       string  `json:"status_batch"`
	QtyDiterima       float64 `json:"qty_diterima"`
	QtySisa           float64 `json:"qty_sisa"`
	HargaBeliPerDasar float64 `json:"harga_beli_per_dasar"`
	TanggalTerima     string  `json:"tanggal_terima"`
}

// DetailPenerimaan adalah satu faktur beserta seluruh itemnya.
type DetailPenerimaan struct {
	PurchaseID  int64                  `json:"purchase_id"`
	SupplierID  int64                  `json:"supplier_id"`
	NamaPBF     string                 `json:"nama_pbf"`
	NoFaktur    string                 `json:"no_faktur"`
	TglFaktur   string                 `json:"tgl_faktur"`
	Total       float64                `json:"total"`
	DicatatOleh string                 `json:"dicatat_oleh"`
	Items       []ItemDetailPenerimaan `json:"items"`
}
