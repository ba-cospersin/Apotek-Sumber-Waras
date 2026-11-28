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
	TglTerima  string                `json:"tgl_terima"` // YYYY-MM-DD, opsional (kosong = hari ini)
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

	// StatusJual memberi tahu apakah stok yang baru diterima boleh dijual:
	// BOLEH_DIJUAL, MENDEKATI_KEDALUWARSA, KEDALUWARSA, atau DIBLOKIR.
	// Selain BOLEH_DIJUAL, stok tetap tercatat dan perlu ditindaklanjuti.
	StatusJual string `json:"status_jual"`
}

// Penerimaan adalah hasil pencatatan satu faktur.
type Penerimaan struct {
	PurchaseID int64            `json:"purchase_id"`
	SupplierID int64            `json:"supplier_id"`
	NoFaktur   string           `json:"no_faktur"`
	TglFaktur  string           `json:"tgl_faktur"`
	TglTerima  string           `json:"tgl_terima"`
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

// ---------- Persediaan dan pemblokiran batch ----------

// ErrSudahDiblokir: batch sudah berstatus diblokir -> HTTP 409.
var ErrSudahDiblokir = errors.New("batch sudah berstatus diblokir")

// FilterBatch adalah filter dari pengguna untuk daftar stok per batch.
type FilterBatch struct {
	Kata           string // cari di nama obat, kode obat, atau nomor batch
	Status         string // BOLEH_DIJUAL, MENDEKATI_KEDALUWARSA, KEDALUWARSA, DIBLOKIR
	TermasukKosong bool   // true = batch yang stoknya habis ikut ditampilkan
	Page           int
	PerPage        int
}

// QueryBatch adalah filter yang sudah divalidasi, siap untuk SQL.
type QueryBatch struct {
	Kata         string
	Status       string
	HariIni      string // YYYY-MM-DD
	Batas        string // YYYY-MM-DD: hari ini + 1 bulan
	HanyaAdaStok bool
	Limit        int
	Offset       int
}

// BatchStok adalah satu batch fisik beserta total stoknya.
type BatchStok struct {
	BatchID         int64   `json:"batch_id"`
	MedicineID      int64   `json:"medicine_id"`
	KodeObat        string  `json:"kode_obat"`
	NamaObat        string  `json:"nama_obat"`
	SatuanDasar     string  `json:"satuan_dasar"`
	NoBatch         string  `json:"no_batch"`
	TglKadaluarsa   string  `json:"tgl_kadaluarsa"`
	StatusBatch     string  `json:"status_batch"` // AKTIF atau DIBLOKIR (database)
	StatusJual      string  `json:"status_jual"`  // hasil perhitungan, lihat statusJual
	TotalStok       float64 `json:"total_stok"`
	StokBolehDijual float64 `json:"stok_boleh_dijual"`
	JumlahLapisan   int     `json:"jumlah_lapisan"`
}

// DaftarBatch adalah satu halaman daftar stok per batch.
type DaftarBatch struct {
	Data []BatchStok `json:"data"`
	Meta Halaman     `json:"meta"`
}

// LapisanBatch adalah satu lapisan penerimaan pada sebuah batch.
type LapisanBatch struct {
	ReceiptID         int64   `json:"receipt_id"`
	PurchaseID        int64   `json:"purchase_id"`
	NoFaktur          string  `json:"no_faktur"`
	TglFaktur         string  `json:"tgl_faktur"`
	NamaPBF           string  `json:"nama_pbf"`
	TanggalTerima     string  `json:"tanggal_terima"`
	QtyDiterima       float64 `json:"qty_diterima"`
	QtySisa           float64 `json:"qty_sisa"`
	HargaBeliPerDasar float64 `json:"harga_beli_per_dasar"`
}

// DetailBatch adalah satu batch fisik beserta seluruh lapisan penerimaannya.
type DetailBatch struct {
	BatchStok
	AlasanBlokir  string         `json:"alasan_blokir"`
	TanggalBlokir string         `json:"tanggal_blokir"`
	Lapisan       []LapisanBatch `json:"lapisan"`
}

// RingkasanStok adalah angka ringkas untuk bagian atas halaman stok.
// Yang dihitung hanya batch yang masih punya stok.
type RingkasanStok struct {
	TotalBatch                int     `json:"total_batch"`
	BatchBolehDijual          int     `json:"batch_boleh_dijual"`
	BatchMendekatiKedaluwarsa int     `json:"batch_mendekati_kedaluwarsa"`
	BatchKedaluwarsa          int     `json:"batch_kedaluwarsa"`
	BatchDiblokir             int     `json:"batch_diblokir"`
	StokBolehDijual           float64 `json:"stok_boleh_dijual"`
}

// ---------- Tindak lanjut stok: pemusnahan dan retur ke PBF ----------

// ErrStokTidakCukup: jumlah tindak lanjut melebihi sisa stok lapisan -> HTTP 409.
var ErrStokTidakCukup = errors.New("jumlah melebihi sisa stok lapisan penerimaan")

// Nilai kolom jenis pada tabel batch_disposals.
const (
	jenisPemusnahan = "PEMUSNAHAN"
	jenisReturPBF   = "RETUR_PBF"
)

// Nilai kolom sumber_tindak_lanjut pada tabel batch_disposals.
const (
	sumberInternalApotek = "INTERNAL_APOTEK"
	sumberBPOM           = "BPOM"
	sumberProdusenPBF    = "PRODUSEN_PBF"
)

// Batas nilai uang (DECIMAL(15,2)).
const maksNilaiUang = 9999999999999.99

// TindakLanjutInput adalah satu pemusnahan atau retur ke PBF untuk satu lapisan.
type TindakLanjutInput struct {
	BatchReceiptID     int64   `json:"batch_receipt_id"`
	Jenis              string  `json:"jenis"`                // PEMUSNAHAN atau RETUR_PBF
	SumberTindakLanjut string  `json:"sumber_tindak_lanjut"` // INTERNAL_APOTEK, BPOM, PRODUSEN_PBF
	Alasan             string  `json:"alasan"`               // KEDALUWARSA, PENARIKAN, RUSAK, atau alasan lain
	QtyDasar           float64 `json:"qty_dasar"`            // dalam satuan dasar
	NilaiKompensasi    float64 `json:"nilai_kompensasi"`     // dari PBF, boleh 0
	Catatan            string  `json:"catatan"`
}

// TindakLanjut adalah hasil pencatatan satu tindak lanjut.
type TindakLanjut struct {
	DisposalID         int64   `json:"disposal_id"`
	BatchReceiptID     int64   `json:"batch_receipt_id"`
	BatchID            int64   `json:"batch_id"`
	Jenis              string  `json:"jenis"`
	SumberTindakLanjut string  `json:"sumber_tindak_lanjut"`
	Alasan             string  `json:"alasan"`
	QtyDasar           float64 `json:"qty_dasar"`
	NilaiPerolehan     float64 `json:"nilai_perolehan"` // harga beli historis x jumlah
	NilaiKompensasi    float64 `json:"nilai_kompensasi"`
	NilaiKerugian      float64 `json:"nilai_kerugian"` // perolehan - kompensasi
	SisaSetelah        float64 `json:"sisa_setelah"`   // sisa lapisan setelah dikurangi
	Tanggal            string  `json:"tanggal"`
}

// LapisanStok adalah data lapisan penerimaan yang dikunci saat tindak lanjut.
type LapisanStok struct {
	ID                int64
	BatchID           int64
	QtySisa           float64
	HargaBeliPerDasar float64
}

// DisposalBaru adalah baris yang akan disimpan ke batch_disposals.
type DisposalBaru struct {
	ReceiptID  int64
	UserID     int64
	Jenis      string
	Sumber     string
	Alasan     string
	Qty        float64
	Kompensasi float64
	Catatan    string
	Waktu      time.Time
}

// FilterTindakLanjut adalah filter dari pengguna untuk riwayat tindak lanjut.
type FilterTindakLanjut struct {
	Jenis   string
	BatchID int64
	Dari    string // YYYY-MM-DD
	Sampai  string // YYYY-MM-DD
	Page    int
	PerPage int
}

// QueryTindakLanjut adalah filter yang sudah divalidasi, siap untuk SQL.
type QueryTindakLanjut struct {
	Jenis   string
	BatchID int64
	Dari    string
	Sampai  string
	Limit   int
	Offset  int
}

// RiwayatTindakLanjut adalah satu baris pada riwayat tindak lanjut.
type RiwayatTindakLanjut struct {
	DisposalID         int64   `json:"disposal_id"`
	BatchReceiptID     int64   `json:"batch_receipt_id"`
	BatchID            int64   `json:"batch_id"`
	MedicineID         int64   `json:"medicine_id"`
	KodeObat           string  `json:"kode_obat"`
	NamaObat           string  `json:"nama_obat"`
	SatuanDasar        string  `json:"satuan_dasar"`
	NoBatch            string  `json:"no_batch"`
	Jenis              string  `json:"jenis"`
	SumberTindakLanjut string  `json:"sumber_tindak_lanjut"`
	Alasan             string  `json:"alasan"`
	QtyDasar           float64 `json:"qty_dasar"`
	NilaiPerolehan     float64 `json:"nilai_perolehan"`
	NilaiKompensasi    float64 `json:"nilai_kompensasi"`
	NilaiKerugian      float64 `json:"nilai_kerugian"`
	Catatan            string  `json:"catatan"`
	DicatatOleh        string  `json:"dicatat_oleh"`
	Tanggal            string  `json:"tanggal"`
}

// DaftarTindakLanjut adalah satu halaman riwayat tindak lanjut.
type DaftarTindakLanjut struct {
	Data []RiwayatTindakLanjut `json:"data"`
	Meta Halaman               `json:"meta"`
}

// ---------- Stock opname ----------

// Batas jumlah (DECIMAL(15,3)).
const maksQty = 999999999999.999

// Nilai status_selisih pada hasil stock opname.
const (
	selisihSesuai = "SESUAI"
	selisihKurang = "KURANG"
	selisihLebih  = "LEBIH"
)

// StockOpnameInput adalah hasil hitung fisik satu batch fisik.
type StockOpnameInput struct {
	BatchID   int64   `json:"batch_id"`
	QtyFisik  float64 `json:"qty_fisik"`  // hasil hitung fisik, satuan dasar
	Catatan   string  `json:"catatan"`    // opsional
	TglOpname string  `json:"tgl_opname"` // YYYY-MM-DD, opsional (kosong = hari ini)
}

// PenyesuaianLapisan adalah perubahan stok pada satu lapisan penerimaan
// akibat selisih stock opname.
type PenyesuaianLapisan struct {
	ReceiptID         int64   `json:"receipt_id"`
	NoFaktur          string  `json:"no_faktur"`
	TanggalTerima     string  `json:"tanggal_terima"`
	HargaBeliPerDasar float64 `json:"harga_beli_per_dasar"`
	QtyPenyesuaian    float64 `json:"qty_penyesuaian"` // negatif = dikurangi
	QtySisaSebelum    float64 `json:"qty_sisa_sebelum"`
	QtySisaSesudah    float64 `json:"qty_sisa_sesudah"`
}

// StockOpname adalah satu hasil opname beserta penyesuaiannya.
type StockOpname struct {
	OpnameID      int64                `json:"opname_id"`
	BatchID       int64                `json:"batch_id"`
	MedicineID    int64                `json:"medicine_id"`
	KodeObat      string               `json:"kode_obat"`
	NamaObat      string               `json:"nama_obat"`
	SatuanDasar   string               `json:"satuan_dasar"`
	NoBatch       string               `json:"no_batch"`
	TglKadaluarsa string               `json:"tgl_kadaluarsa"`
	QtySistem     float64              `json:"qty_sistem"`
	QtyFisik      float64              `json:"qty_fisik"`
	Selisih       float64              `json:"selisih"`        // fisik - sistem
	StatusSelisih string               `json:"status_selisih"` // SESUAI, KURANG, LEBIH
	NilaiSelisih  float64              `json:"nilai_selisih"`  // total (penyesuaian x harga beli lapisan)
	Catatan       string               `json:"catatan"`
	DicatatOleh   string               `json:"dicatat_oleh"`
	Tanggal       string               `json:"tanggal"`
	Penyesuaian   []PenyesuaianLapisan `json:"penyesuaian"`
}

// OpnameBaru adalah baris yang akan disimpan ke stock_opnames.
type OpnameBaru struct {
	BatchID   int64
	UserID    int64
	QtySistem float64
	QtyFisik  float64
	Catatan   string
	Waktu     time.Time
}

// FilterStockOpname adalah filter dari pengguna untuk riwayat opname.
type FilterStockOpname struct {
	BatchID      int64
	Dari         string // YYYY-MM-DD
	Sampai       string // YYYY-MM-DD
	HanyaSelisih bool   // true = hanya opname yang ada selisihnya
	Page         int
	PerPage      int
}

// QueryStockOpname adalah filter yang sudah divalidasi, siap untuk SQL.
type QueryStockOpname struct {
	BatchID      int64
	Dari         string
	Sampai       string
	HanyaSelisih bool
	Limit        int
	Offset       int
}

// RiwayatStockOpname adalah satu baris pada riwayat opname.
type RiwayatStockOpname struct {
	OpnameID      int64   `json:"opname_id"`
	BatchID       int64   `json:"batch_id"`
	MedicineID    int64   `json:"medicine_id"`
	KodeObat      string  `json:"kode_obat"`
	NamaObat      string  `json:"nama_obat"`
	SatuanDasar   string  `json:"satuan_dasar"`
	NoBatch       string  `json:"no_batch"`
	QtySistem     float64 `json:"qty_sistem"`
	QtyFisik      float64 `json:"qty_fisik"`
	Selisih       float64 `json:"selisih"`
	StatusSelisih string  `json:"status_selisih"`
	JumlahLapisan int     `json:"jumlah_lapisan_disesuaikan"`
	Catatan       string  `json:"catatan"`
	DicatatOleh   string  `json:"dicatat_oleh"`
	Tanggal       string  `json:"tanggal"`
}

// DaftarStockOpname adalah satu halaman riwayat opname.
type DaftarStockOpname struct {
	Data []RiwayatStockOpname `json:"data"`
	Meta Halaman              `json:"meta"`
}

// ---------- Sinkronisasi dengan halaman ----------

// statusJualBermasalah adalah filter gabungan untuk halaman kedaluwarsa:
// batch yang TIDAK boleh dijual (diblokir, kedaluwarsa, atau mendekati
// kedaluwarsa) dan masih punya stok. Bukan status per batch.
const statusJualBermasalah = "BERMASALAH"

// SatuanOpsi adalah satu satuan jual sebuah obat.
type SatuanOpsi struct {
	ID                int64   `json:"id"`
	NamaSatuan        string  `json:"nama_satuan"`
	IsiPerSatuanDasar float64 `json:"isi_per_satuan_dasar"`
}

// ObatOpsi adalah satu obat aktif beserta satuan-satuannya.
type ObatOpsi struct {
	ID          int64        `json:"id"`
	Kode        string       `json:"kode"`
	Nama        string       `json:"nama"`
	SatuanDasar string       `json:"satuan_dasar"`
	Satuan      []SatuanOpsi `json:"satuan"`
}

// PBFOpsi adalah satu PBF.
type PBFOpsi struct {
	ID      int64  `json:"id"`
	NamaPBF string `json:"nama_pbf"`
}

// OpsiPenerimaan adalah pilihan untuk mengisi form penerimaan.
type OpsiPenerimaan struct {
	PBF  []PBFOpsi  `json:"pbf"`
	Obat []ObatOpsi `json:"obat"`
}

// PratinjauOpname adalah hasil alokasi opname seandainya disimpan sekarang.
// Lapisan berisi SEMUA lapisan batch (yang tidak berubah punya penyesuaian 0).
type PratinjauOpname struct {
	BatchID       int64                `json:"batch_id"`
	MedicineID    int64                `json:"medicine_id"`
	KodeObat      string               `json:"kode_obat"`
	NamaObat      string               `json:"nama_obat"`
	SatuanDasar   string               `json:"satuan_dasar"`
	NoBatch       string               `json:"no_batch"`
	QtySistem     float64              `json:"qty_sistem"`
	QtyFisik      float64              `json:"qty_fisik"`
	Selisih       float64              `json:"selisih"`
	StatusSelisih string               `json:"status_selisih"`
	NilaiSelisih  float64              `json:"nilai_selisih"`
	Lapisan       []PenyesuaianLapisan `json:"lapisan"`
}
