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

	// ErrStatusSudahFinal: status kelayakan retur sudah ditetapkan -> HTTP 409.
	ErrStatusSudahFinal = errors.New("status kelayakan sudah ditetapkan")
)

// validasi membuat error validasi dengan pesan yang bisa langsung ditampilkan ke pengguna.
func validasi(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrValidasi, fmt.Sprintf(format, args...))
}

// Lapisan adalah satu lapisan penerimaan yang SUDAH lolos syarat jual (tidak diblokir, belum mendekati kedaluwarsa).
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

// StrukItem adalah satu baris pada struk.
type StrukItem struct {
	NamaObat    string  `json:"nama_obat"`
	NamaSatuan  string  `json:"nama_satuan"`
	Qty         float64 `json:"qty"`          // dalam satuan yang dibeli
	HargaSatuan float64 `json:"harga_satuan"` // per satuan yang dibeli
	Subtotal    float64 `json:"subtotal"`

	// Diisi repository, dipakai service untuk menghitung Subtotal dan HargaSatuan. 
	// Tidak ikut dikirim sebagai JSON.
	qtyDasar   float64
	hargaDasar float64
}

// Struk adalah data untuk menampilkan dan mencetak struk penjualan.
type Struk struct {
	SaleID      int64       `json:"sale_id"`
	Tanggal     time.Time   `json:"tanggal"`
	Kasir       string      `json:"kasir"`
	MetodeBayar string      `json:"metode_bayar"`
	Items       []StrukItem `json:"items"`
	Total       float64     `json:"total"`
	JumlahBayar float64     `json:"jumlah_bayar"`
	Kembalian   float64     `json:"kembalian"`
}

// Jenis pengembalian dari konsumen (disimpan di returns.jenis_pengembalian).
const (
	JenisSalahMembeli        = "SALAH_MEMBELI"        // salah membeli atau perubahan resep
	JenisObatRusak           = "OBAT_RUSAK"           // obat rusak atau dugaan masalah mutu
	JenisKesalahanPenyerahan = "KESALAHAN_PENYERAHAN" // salah menyerahkan obat
	JenisPenarikanResmi      = "PENARIKAN_RESMI"      // pengembalian akibat penarikan resmi
)

// Status kelayakan obat yang dikembalikan (returns.status_kelayakan).
const (
	StatusMenunggu       = "MENUNGGU_PEMERIKSAAN"
	StatusLayakJual      = "LAYAK_JUAL"
	StatusTidakLayakJual = "TIDAK_LAYAK_JUAL"
)

// ReturInput adalah permintaan pencatatan satu retur konsumen.
type ReturInput struct {
	SaleItemBatchID   int64   `json:"sale_item_batch_id"`
	QtyDasar          float64 `json:"qty_dasar"` // dalam satuan dasar
	JenisPengembalian string  `json:"jenis_pengembalian"`
	Alasan            string  `json:"alasan"`
	KondisiObat       string  `json:"kondisi_obat"`
	StatusKelayakan   string  `json:"status_kelayakan"` // opsional, bawaan MENUNGGU_PEMERIKSAAN
}

// HasilRetur adalah hasil retur yang berhasil disimpan.
type HasilRetur struct {
	ReturID         int64   `json:"retur_id"`
	SaleItemBatchID int64   `json:"sale_item_batch_id"`
	QtyDasar        float64 `json:"qty_dasar"`
	NilaiRefund     float64 `json:"nilai_refund"`
	StatusKelayakan string  `json:"status_kelayakan"`
	StokDitambah    bool    `json:"stok_ditambah"`
}

// KelayakanInput adalah permintaan penetapan kelayakan obat retur.
type KelayakanInput struct {
	StatusKelayakan string `json:"status_kelayakan"` // LAYAK_JUAL atau TIDAK_LAYAK_JUAL
}

// HasilKelayakan adalah hasil penetapan kelayakan.
type HasilKelayakan struct {
	ReturID         int64  `json:"retur_id"`
	StatusKelayakan string `json:"status_kelayakan"`
	StokDitambah    bool   `json:"stok_ditambah"`
}

// ReturBatch adalah satu alokasi batch pada transaksi asal beserta sisa yang masih bisa diretur.
type ReturBatch struct {
	SaleItemBatchID int64     `json:"sale_item_batch_id"`
	NoBatch         string    `json:"no_batch"`
	TglKadaluarsa   time.Time `json:"tgl_kadaluarsa"`
	QtyDasar        float64   `json:"qty_dasar"`     // jumlah yang dialokasikan saat dijual
	SudahDiretur    float64   `json:"sudah_diretur"` // total retur sebelumnya
	BisaDiretur     float64   `json:"bisa_diretur"`
}

// ReturItem adalah satu obat pada transaksi asal.
type ReturItem struct {
	NamaObat          string       `json:"nama_obat"`
	NamaSatuan        string       `json:"nama_satuan"` // satuan yang dibeli
	QtyDibeli         float64      `json:"qty_dibeli"`  // dalam satuan yang dibeli
	SatuanDasar       string       `json:"satuan_dasar"`
	HargaJualPerDasar float64      `json:"harga_jual_per_dasar"`
	Batches           []ReturBatch `json:"batches"`
}

// PenjualanRetur adalah transaksi asal yang ditampilkan di halaman retur.
type PenjualanRetur struct {
	SaleID  int64       `json:"sale_id"`
	Tanggal time.Time   `json:"tanggal"`
	Kasir   string      `json:"kasir"`
	Total   float64     `json:"total"`
	Items   []ReturItem `json:"items"`
}