package cashier_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/joho/godotenv"

	. "github.com/ba-cospersin/Apotek-Sumber-Waras/internal/cashier"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/config"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/database"
)

const layoutTanggalSQL = "2006-01-02"

// keMilli dipakai untuk membandingkan kuantitas desimal tanpa selisih float.
func keMilli(x float64) int64 { return int64(math.Round(x * 1000)) }

// Tes integrasi memakai database sungguhan. Hanya berjalan jika ASW_INTEGRASI=1
// dan DB_NAME berakhiran _test, supaya data asli tidak tersentuh.

func bukaDBUji(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("ASW_INTEGRASI") != "1" {
		t.Skip("tes integrasi dilewati (set ASW_INTEGRASI=1 dan DB_NAME berakhiran _test)")
	}

	// .env dibaca dari akar proyek. Variabel yang sudah diset di terminal
	// (misalnya DB_NAME) tidak ditimpa oleh .env.
	_ = godotenv.Load("../../.env")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("gagal membaca konfigurasi: %v", err)
	}
	if !strings.HasSuffix(cfg.DBName, "_test") {
		t.Fatalf("DB_NAME %q harus berakhiran _test supaya data asli aman", cfg.DBName)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		t.Fatalf("gagal terhubung ke database uji: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func sisip(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	res, err := db.Exec(query, args...)
	if err != nil {
		t.Fatalf("gagal menyiapkan data uji: %v\nquery: %s", err, query)
	}
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// tglKedaluwarsa menghasilkan tanggal (YYYY-MM-DD) relatif terhadap hari ini.
func tglKedaluwarsa(bulan, hari int) string {
	return time.Now().AddDate(0, bulan, hari).Format(layoutTanggalSQL)
}

// fixture adalah satu obat uji beserta PBF dan faktur miliknya sendiri.
type fixture struct {
	db         *sql.DB
	svc        *Service
	userID     int64
	supplierID int64
	purchaseID int64
	medicineID int64
	unitTablet int64 // isi 1, harga Rp1.000
	unitStrip  int64 // isi 10, harga Rp9.000
	sufiks     string
}

func buatFixture(t *testing.T, db *sql.DB) *fixture {
	t.Helper()
	f := &fixture{
		db:     db,
		svc:    NewService(NewRepository(db), StokKasir{}),
		sufiks: fmt.Sprintf("%d", time.Now().UnixNano()),
	}

	err := db.QueryRow(
		`SELECT id FROM users WHERE role = 'KASIR' AND is_active = 1 ORDER BY id LIMIT 1`,
	).Scan(&f.userID)
	if err != nil {
		t.Fatalf("akun KASIR tidak ditemukan (jalankan migrate dan seed di database uji): %v", err)
	}

	f.supplierID = sisip(t, db, `INSERT INTO suppliers (nama_pbf) VALUES (?)`, "PBF Uji "+f.sufiks)
	f.purchaseID = sisip(t, db,
		`INSERT INTO purchases (supplier_id, user_id, no_faktur, tgl_faktur, total)
		 VALUES (?, ?, ?, CURDATE(), 0)`,
		f.supplierID, f.userID, "FK-"+f.sufiks)
	f.medicineID = sisip(t, db,
		`INSERT INTO medicines (kode, nama, satuan_dasar) VALUES (?, ?, 'tablet')`,
		"UJI"+f.sufiks, "Obat Uji "+f.sufiks)
	f.unitTablet = sisip(t, db,
		`INSERT INTO medicine_units (medicine_id, nama_satuan, isi_per_satuan_dasar, harga_jual)
		 VALUES (?, 'tablet', 1, 1000)`, f.medicineID)
	f.unitStrip = sisip(t, db,
		`INSERT INTO medicine_units (medicine_id, nama_satuan, isi_per_satuan_dasar, harga_jual)
		 VALUES (?, 'strip', 10, 9000)`, f.medicineID)

	t.Cleanup(func() { f.bersihkan(t) })
	return f
}

// tambahLapisan membuat satu batch beserta satu lapisan penerimaan.
func (f *fixture) tambahLapisan(
	t *testing.T, noBatch, kedaluwarsa, status string,
	qty, hargaBeli float64, terima time.Time,
) int64 {
	t.Helper()
	batchID := sisip(t, f.db,
		`INSERT INTO batches (medicine_id, no_batch, tgl_kadaluarsa, status_batch)
		 VALUES (?, ?, ?, ?)`,
		f.medicineID, noBatch+"-"+f.sufiks, kedaluwarsa, status)
	return sisip(t, f.db,
		`INSERT INTO batch_receipts
		    (batch_id, purchase_id, qty_diterima, qty_sisa, harga_beli_per_dasar, tanggal_terima)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		batchID, f.purchaseID, qty, qty, hargaBeli, terima)
}

func (f *fixture) jual(unitID int64, qty, bayar float64) (*HasilTransaksi, error) {
	return f.svc.Jual(context.Background(), f.userID, TransaksiInput{
		MetodeBayar: MetodeTunai,
		JumlahBayar: bayar,
		Items:       []ItemInput{{UnitID: unitID, Qty: qty}},
	})
}

func (f *fixture) sisa(t *testing.T, receiptID int64) float64 {
	t.Helper()
	var v float64
	if err := f.db.QueryRow(
		`SELECT qty_sisa FROM batch_receipts WHERE id = ?`, receiptID,
	).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func (f *fixture) jumlahPenjualan(t *testing.T) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(`SELECT COUNT(*) FROM sales`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func (f *fixture) jumlahRetur(t *testing.T, saleItemBatchID int64) int {
	t.Helper()
	var n int
	if err := f.db.QueryRow(
		`SELECT COUNT(*) FROM returns WHERE sale_item_batch_id = ?`, saleItemBatchID,
	).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

type alokasiDB struct {
	ID        int64
	ReceiptID int64
	Qty       float64
	HargaBeli float64
}

func (f *fixture) alokasi(t *testing.T, saleID int64) []alokasiDB {
	t.Helper()
	rows, err := f.db.Query(
		`SELECT sib.id, sib.batch_receipt_id, sib.qty_dasar, sib.harga_beli_snapshot
		   FROM sale_item_batches sib
		   JOIN sale_items si ON si.id = sib.sale_item_id
		  WHERE si.sale_id = ?
		  ORDER BY sib.id`, saleID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var hasil []alokasiDB
	for rows.Next() {
		var a alokasiDB
		if err := rows.Scan(&a.ID, &a.ReceiptID, &a.Qty, &a.HargaBeli); err != nil {
			t.Fatal(err)
		}
		hasil = append(hasil, a)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return hasil
}

// periksaKonsistensi memastikan setiap perubahan stok punya catatan:
// qty_sisa = qty_diterima - penjualan + retur layak jual, dan tidak negatif.
func (f *fixture) periksaKonsistensi(t *testing.T) {
	t.Helper()
	rows, err := f.db.Query(
		`SELECT br.id, br.qty_sisa,
		        br.qty_diterima
		        - COALESCE((SELECT SUM(sib.qty_dasar)
		                      FROM sale_item_batches sib
		                     WHERE sib.batch_receipt_id = br.id), 0)
		        + COALESCE((SELECT SUM(rt.qty_dasar)
		                      FROM returns rt
		                      JOIN sale_item_batches sib ON sib.id = rt.sale_item_batch_id
		                     WHERE sib.batch_receipt_id = br.id
		                       AND rt.status_kelayakan = 'LAYAK_JUAL'), 0)
		   FROM batch_receipts br
		   JOIN batches b ON b.id = br.batch_id
		  WHERE b.medicine_id = ?`, f.medicineID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		var sisa, seharusnya float64
		if err := rows.Scan(&id, &sisa, &seharusnya); err != nil {
			t.Fatal(err)
		}
		if keMilli(sisa) < 0 {
			t.Errorf("lapisan %d: stok negatif (%v)", id, sisa)
		}
		if keMilli(sisa) != keMilli(seharusnya) {
			t.Errorf("lapisan %d: qty_sisa %v, seharusnya %v (penerimaan - penjualan + retur layak jual)",
				id, sisa, seharusnya)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

// bersihkan menghapus semua data uji milik fixture ini, sesuai urutan relasi.
func (f *fixture) bersihkan(t *testing.T) {
	var saleIDs []int64
	rows, err := f.db.Query(
		`SELECT DISTINCT si.sale_id
		   FROM sale_items si
		   JOIN medicine_units u ON u.id = si.unit_id
		  WHERE u.medicine_id = ?`, f.medicineID)
	if err == nil {
		for rows.Next() {
			var id int64
			if rows.Scan(&id) == nil {
				saleIDs = append(saleIDs, id)
			}
		}
		rows.Close()
	}

	hapus := func(query string, args ...any) {
		if _, err := f.db.Exec(query, args...); err != nil {
			t.Logf("bersihkan: %v (query: %s)", err, query)
		}
	}

	hapus(`DELETE rt FROM returns rt
	         JOIN sale_item_batches sib ON sib.id = rt.sale_item_batch_id
	         JOIN batch_receipts br ON br.id = sib.batch_receipt_id
	         JOIN batches b ON b.id = br.batch_id
	        WHERE b.medicine_id = ?`, f.medicineID)
	hapus(`DELETE sib FROM sale_item_batches sib
	         JOIN batch_receipts br ON br.id = sib.batch_receipt_id
	         JOIN batches b ON b.id = br.batch_id
	        WHERE b.medicine_id = ?`, f.medicineID)
	hapus(`DELETE si FROM sale_items si
	         JOIN medicine_units u ON u.id = si.unit_id
	        WHERE u.medicine_id = ?`, f.medicineID)
	for _, id := range saleIDs {
		hapus(`DELETE FROM sales WHERE id = ?`, id)
	}
	hapus(`DELETE br FROM batch_receipts br
	         JOIN batches b ON b.id = br.batch_id
	        WHERE b.medicine_id = ?`, f.medicineID)
	hapus(`DELETE FROM batches WHERE medicine_id = ?`, f.medicineID)
	hapus(`DELETE FROM purchases WHERE id = ?`, f.purchaseID)
	hapus(`DELETE FROM medicine_units WHERE medicine_id = ?`, f.medicineID)
	hapus(`DELETE FROM medicines WHERE id = ?`, f.medicineID)
	hapus(`DELETE FROM suppliers WHERE id = ?`, f.supplierID)
}

// ---------------------------------------------------------------------------

func TestIntegrasi_PenjualanMultiBatch(t *testing.T) {
	db := bukaDBUji(t)
	f := buatFixture(t, db)
	ctx := context.Background()
	terima := time.Now().AddDate(0, 0, -10)

	rA := f.tambahLapisan(t, "A", tglKedaluwarsa(6, 0), "AKTIF", 30, 500, terima)
	rB := f.tambahLapisan(t, "B", tglKedaluwarsa(12, 0), "AKTIF", 50, 520, terima)
	rC := f.tambahLapisan(t, "C", tglKedaluwarsa(0, 10), "AKTIF", 100, 400, terima)    // mendekati kedaluwarsa
	rD := f.tambahLapisan(t, "D", tglKedaluwarsa(12, 0), "DIBLOKIR", 100, 400, terima) // diblokir

	// Hanya A dan B yang boleh dijual.
	stok, err := StokKasir{}.StokTersedia(ctx, f.db, f.medicineID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if stok != 80 {
		t.Fatalf("stok tersedia %v, seharusnya 80 (batch C dan D tidak boleh dihitung)", stok)
	}

	// 1 strip (10 tablet) diambil dari A, kedaluwarsa terdekat.
	h, err := f.jual(f.unitStrip, 1, 100000)
	if err != nil {
		t.Fatal(err)
	}
	if h.Total != 9000 || h.Kembalian != 91000 {
		t.Fatalf("hasil tidak sesuai: %+v", h)
	}
	if got := f.sisa(t, rA); got != 20 {
		t.Fatalf("sisa A %v, seharusnya 20", got)
	}

	// 4 strip (40 tablet): 20 dari A (habis) dan 20 dari B.
	h, err = f.jual(f.unitStrip, 4, 100000)
	if err != nil {
		t.Fatal(err)
	}
	al := f.alokasi(t, h.SaleID)
	if len(al) != 2 ||
		al[0].ReceiptID != rA || al[0].Qty != 20 || al[0].HargaBeli != 500 ||
		al[1].ReceiptID != rB || al[1].Qty != 20 || al[1].HargaBeli != 520 {
		t.Fatalf("alokasi tidak sesuai: %+v", al)
	}
	if f.sisa(t, rA) != 0 || f.sisa(t, rB) != 30 {
		t.Fatalf("sisa A=%v B=%v, seharusnya 0 dan 30", f.sisa(t, rA), f.sisa(t, rB))
	}

	// Stok tidak cukup: tidak boleh ada yang berubah sama sekali.
	sebelum := f.jumlahPenjualan(t)
	_, err = f.jual(f.unitStrip, 4, 100000) // butuh 40, tersedia 30
	if !errors.Is(err, ErrStokTidakCukup) {
		t.Fatalf("seharusnya ErrStokTidakCukup, dapat %v", err)
	}
	if sesudah := f.jumlahPenjualan(t); sesudah != sebelum {
		t.Fatalf("penjualan gagal meninggalkan data: %d -> %d", sebelum, sesudah)
	}
	if f.sisa(t, rB) != 30 || f.sisa(t, rC) != 100 || f.sisa(t, rD) != 100 {
		t.Fatal("stok berubah setelah penjualan gagal, atau batch terlarang tersentuh")
	}

	f.periksaKonsistensi(t)
}

func TestIntegrasi_KeranjangObatSama(t *testing.T) {
	db := bukaDBUji(t)
	f := buatFixture(t, db)
	ctx := context.Background()
	rA := f.tambahLapisan(t, "A", tglKedaluwarsa(6, 0), "AKTIF", 25, 500, time.Now().AddDate(0, 0, -5))

	// 2 strip (20) + 6 tablet = 26 > 25. Item kedua harus melihat stok setelah item pertama.
	sebelum := f.jumlahPenjualan(t)
	_, err := f.svc.Jual(ctx, f.userID, TransaksiInput{
		MetodeBayar: MetodeTunai, JumlahBayar: 100000,
		Items: []ItemInput{{UnitID: f.unitStrip, Qty: 2}, {UnitID: f.unitTablet, Qty: 6}},
	})
	if !errors.Is(err, ErrStokTidakCukup) {
		t.Fatalf("seharusnya ErrStokTidakCukup, dapat %v", err)
	}
	if f.sisa(t, rA) != 25 || f.jumlahPenjualan(t) != sebelum {
		t.Fatal("transaksi gagal meninggalkan perubahan")
	}

	// 2 strip + 5 tablet = 25 pas.
	h, err := f.svc.Jual(ctx, f.userID, TransaksiInput{
		MetodeBayar: MetodeTunai, JumlahBayar: 30000,
		Items: []ItemInput{{UnitID: f.unitStrip, Qty: 2}, {UnitID: f.unitTablet, Qty: 5}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if h.Total != 23000 || h.Kembalian != 7000 {
		t.Fatalf("hasil tidak sesuai: %+v", h)
	}
	if got := f.sisa(t, rA); got != 0 {
		t.Fatalf("sisa %v, seharusnya 0", got)
	}

	f.periksaKonsistensi(t)
}

func TestIntegrasi_DuaKasirBersamaan(t *testing.T) {
	db := bukaDBUji(t)
	f := buatFixture(t, db)
	rA := f.tambahLapisan(t, "A", tglKedaluwarsa(6, 0), "AKTIF", 100, 500, time.Now().AddDate(0, 0, -5))

	// 8 pembeli serentak, masing-masing 15 tablet = 120 > 100.
	// Yang benar: tepat 6 berhasil (90), 2 ditolak, sisa 10.
	const pembeli = 8
	mulai := make(chan struct{})
	hasil := make(chan error, pembeli)
	var wg sync.WaitGroup
	for i := 0; i < pembeli; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-mulai
			_, err := f.jual(f.unitTablet, 15, 100000)
			hasil <- err
		}()
	}
	close(mulai)
	wg.Wait()
	close(hasil)

	berhasil, ditolak := 0, 0
	for err := range hasil {
		switch {
		case err == nil:
			berhasil++
		case errors.Is(err, ErrStokTidakCukup):
			ditolak++
		default:
			t.Errorf("hasil tak terduga: %v", err)
		}
	}
	if berhasil != 6 || ditolak != 2 {
		t.Fatalf("berhasil %d, ditolak %d; seharusnya 6 dan 2", berhasil, ditolak)
	}
	if got := f.sisa(t, rA); got != 10 {
		t.Fatalf("sisa %v, seharusnya 10", got)
	}

	f.periksaKonsistensi(t)
}

func TestIntegrasi_ReturKumulatifDanKelayakan(t *testing.T) {
	db := bukaDBUji(t)
	f := buatFixture(t, db)
	ctx := context.Background()
	rA := f.tambahLapisan(t, "A", tglKedaluwarsa(6, 0), "AKTIF", 100, 500, time.Now().AddDate(0, 0, -5))

	h, err := f.jual(f.unitTablet, 10, 100000) // stok jadi 90
	if err != nil {
		t.Fatal(err)
	}
	al := f.alokasi(t, h.SaleID)
	if len(al) != 1 {
		t.Fatalf("alokasi %+v", al)
	}
	sib := al[0].ID

	retur := func(qty float64, status string) (*HasilRetur, error) {
		return f.svc.CatatRetur(ctx, f.userID, ReturInput{
			SaleItemBatchID:   sib,
			QtyDasar:          qty,
			JenisPengembalian: JenisSalahMembeli,
			Alasan:            "Uji integrasi",
			StatusKelayakan:   status,
		})
	}

	// 1. Status dari client diabaikan: retur selalu MENUNGGU, stok belum bertambah.
	r1, err := retur(4, StatusLayakJual)
	if err != nil {
		t.Fatal(err)
	}
	if r1.NilaiRefund != 4000 || r1.StatusKelayakan != StatusMenunggu || r1.StokDitambah || f.sisa(t, rA) != 90 {
		t.Fatalf("retur 1 tidak sesuai: %+v, sisa %v", r1, f.sisa(t, rA))
	}

	// 2. Menunggu pemeriksaan: stok tidak berubah.
	r2, err := retur(3, "")
	if err != nil {
		t.Fatal(err)
	}
	if r2.StatusKelayakan != StatusMenunggu || r2.StokDitambah || f.sisa(t, rA) != 90 {
		t.Fatalf("retur 2 tidak sesuai: %+v, sisa %v", r2, f.sisa(t, rA))
	}

	// 3. Status TIDAK_LAYAK_JUAL dari client juga diabaikan; tetap MENUNGGU.
	r3, err := retur(3, StatusTidakLayakJual)
	if err != nil {
		t.Fatal(err)
	}
	if r3.StatusKelayakan != StatusMenunggu || r3.StokDitambah || f.sisa(t, rA) != 90 {
		t.Fatalf("retur 3 tidak sesuai: %+v, sisa %v", r3, f.sisa(t, rA))
	}

	// 4. Melebihi batas kumulatif: ditolak, tidak ada baris baru.
	if _, err := retur(1, ""); !errors.Is(err, ErrValidasi) {
		t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
	}
	if n := f.jumlahRetur(t, sib); n != 3 {
		t.Fatalf("jumlah retur %d, seharusnya 3", n)
	}

	// 5. Owner menetapkan r1 LAYAK_JUAL: stok +4 (90 -> 94).
	k1, err := f.svc.TetapkanKelayakan(ctx, r1.ReturID, StatusLayakJual)
	if err != nil {
		t.Fatal(err)
	}
	if !k1.StokDitambah || f.sisa(t, rA) != 94 {
		t.Fatalf("kelayakan r1 tidak sesuai: %+v, sisa %v", k1, f.sisa(t, rA))
	}

	// 6. Owner menetapkan r2 LAYAK_JUAL: stok +3 (94 -> 97).
	k2, err := f.svc.TetapkanKelayakan(ctx, r2.ReturID, StatusLayakJual)
	if err != nil {
		t.Fatal(err)
	}
	if !k2.StokDitambah || f.sisa(t, rA) != 97 {
		t.Fatalf("kelayakan r2 tidak sesuai: %+v, sisa %v", k2, f.sisa(t, rA))
	}

	// 7. Diulang: ditolak, stok tidak bertambah lagi.
	if _, err := f.svc.TetapkanKelayakan(ctx, r2.ReturID, StatusLayakJual); !errors.Is(err, ErrStatusSudahFinal) {
		t.Fatalf("seharusnya ErrStatusSudahFinal, dapat %v", err)
	}

	// 8. Owner menetapkan r3 TIDAK_LAYAK_JUAL: stok tidak berubah.
	k3, err := f.svc.TetapkanKelayakan(ctx, r3.ReturID, StatusTidakLayakJual)
	if err != nil {
		t.Fatal(err)
	}
	if k3.StokDitambah || f.sisa(t, rA) != 97 {
		t.Fatalf("kelayakan r3 tidak sesuai: %+v, sisa %v", k3, f.sisa(t, rA))
	}
	if _, err := f.svc.TetapkanKelayakan(ctx, r3.ReturID, StatusLayakJual); !errors.Is(err, ErrStatusSudahFinal) {
		t.Fatalf("seharusnya ErrStatusSudahFinal, dapat %v", err)
	}
	if got := f.sisa(t, rA); got != 97 {
		t.Fatalf("stok %v, seharusnya tetap 97", got)
	}

	f.periksaKonsistensi(t)
}

func TestIntegrasi_ReturBersamaan(t *testing.T) {
	db := bukaDBUji(t)
	f := buatFixture(t, db)
	ctx := context.Background()
	f.tambahLapisan(t, "A", tglKedaluwarsa(6, 0), "AKTIF", 100, 500, time.Now().AddDate(0, 0, -5))

	h, err := f.jual(f.unitTablet, 10, 100000)
	if err != nil {
		t.Fatal(err)
	}
	sib := f.alokasi(t, h.SaleID)[0].ID

	// 5 retur serentak masing-masing 4 dari 10 yang dibeli.
	// Yang benar: tepat 2 berhasil (8), 3 ditolak.
	const jumlah = 5
	mulai := make(chan struct{})
	hasil := make(chan error, jumlah)
	var wg sync.WaitGroup
	for i := 0; i < jumlah; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-mulai
			_, err := f.svc.CatatRetur(ctx, f.userID, ReturInput{
				SaleItemBatchID:   sib,
				QtyDasar:          4,
				JenisPengembalian: JenisSalahMembeli,
				Alasan:            "Uji retur bersamaan",
			})
			hasil <- err
		}()
	}
	close(mulai)
	wg.Wait()
	close(hasil)

	berhasil, ditolak := 0, 0
	for err := range hasil {
		switch {
		case err == nil:
			berhasil++
		case errors.Is(err, ErrValidasi):
			ditolak++
		default:
			t.Errorf("hasil tak terduga: %v", err)
		}
	}
	if berhasil != 2 || ditolak != 3 {
		t.Fatalf("berhasil %d, ditolak %d; seharusnya 2 dan 3", berhasil, ditolak)
	}

	var total float64
	if err := f.db.QueryRow(
		`SELECT COALESCE(SUM(qty_dasar), 0) FROM returns WHERE sale_item_batch_id = ?`, sib,
	).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if total != 8 {
		t.Fatalf("total retur %v, seharusnya 8 (tidak boleh melebihi 10)", total)
	}

	f.periksaKonsistensi(t)
}

func TestIntegrasi_Laba(t *testing.T) {
	db := bukaDBUji(t)
	f := buatFixture(t, db)
	ctx := context.Background()
	f.tambahLapisan(t, "A", tglKedaluwarsa(6, 0), "AKTIF", 100, 500, time.Now().AddDate(0, 0, -5))

	h, err := f.jual(f.unitTablet, 10, 100000) // total 10.000, HPP 5.000
	if err != nil {
		t.Fatal(err)
	}
	sib := f.alokasi(t, h.SaleID)[0].ID

	permintaanRetur := []struct {
		qty    float64
		status string
	}{
		{4, StatusLayakJual},      // refund 4.000, HPP retur 2.000
		{2, StatusTidakLayakJual}, // refund 2.000, HPP retur 1.000, kerugian 1.000
	}
	hasilRetur := make([]*HasilRetur, 0, len(permintaanRetur))
	for _, r := range permintaanRetur {
		h, err := f.svc.CatatRetur(ctx, f.userID, ReturInput{
			SaleItemBatchID:   sib,
			QtyDasar:          r.qty,
			JenisPengembalian: JenisSalahMembeli,
			Alasan:            "Uji laba",
			StatusKelayakan:   r.status,
		})
		if err != nil {
			t.Fatal(err)
		}
		hasilRetur = append(hasilRetur, h)
	}
	// Simulasi Owner menetapkan kelayakan setelah retur dicatat.
	if _, err := f.svc.TetapkanKelayakan(ctx, hasilRetur[0].ReturID, StatusLayakJual); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.TetapkanKelayakan(ctx, hasilRetur[1].ReturID, StatusTidakLayakJual); err != nil {
		t.Fatal(err)
	}

	lap, err := f.svc.LaporanLaba(ctx, "", "", 500, 0)
	if err != nil {
		t.Fatal(err)
	}
	var tr *LabaTransaksi
	for i := range lap.Transaksi {
		if lap.Transaksi[i].SaleID == h.SaleID {
			tr = &lap.Transaksi[i]
		}
	}
	if tr == nil {
		t.Fatal("transaksi uji tidak muncul di laporan")
	}

	if tr.Pendapatan != 10000 || tr.HPP != 5000 || tr.LabaKotor != 5000 ||
		tr.ReturRefund != 6000 || tr.ReturHPP != 3000 || tr.LabaRetur != 3000 ||
		tr.LabaBersih != 2000 || tr.KerugianRetur != 1000 {
		t.Fatalf("laba tidak sesuai: %+v", tr.Laba)
	}
}

func TestIntegrasi_BatasKedaluwarsaSatuBulan(t *testing.T) {
	db := bukaDBUji(t)
	f := buatFixture(t, db)
	terima := time.Now().AddDate(0, 0, -5)

	// Batas dihitung oleh MariaDB sendiri, sama seperti aturan di kode.
	var batas string
	if err := f.db.QueryRow(
		`SELECT DATE_FORMAT(DATE_ADD(CURDATE(), INTERVAL 1 MONTH), '%Y-%m-%d')`,
	).Scan(&batas); err != nil {
		t.Fatal(err)
	}
	tglBatas, err := time.Parse(layoutTanggalSQL, batas)
	if err != nil {
		t.Fatal(err)
	}

	f.tambahLapisan(t, "TEPAT", batas, "AKTIF", 40, 500, terima)                                              // tepat satu bulan: tidak boleh dijual
	f.tambahLapisan(t, "LEBIH", tglBatas.AddDate(0, 0, 1).Format(layoutTanggalSQL), "AKTIF", 25, 500, terima) // sehari lebih: boleh

	stok, err := StokKasir{}.StokTersedia(context.Background(), f.db, f.medicineID, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if stok != 25 {
		t.Fatalf("stok tersedia %v, seharusnya 25 (batch tepat satu bulan tidak boleh dijual)", stok)
	}
}
