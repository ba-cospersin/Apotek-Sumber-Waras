package cashier

import (
	"math"
	"sort"
)

// Kuantitas dihitung dalam seperseribu (sesuai DECIMAL(15,3) di database)
// supaya penjumlahan desimal tidak menghasilkan selisih.
func keMilli(x float64) int64   { return int64(math.Round(x * 1000)) }
func dariMilli(x int64) float64 { return float64(x) / 1000 }

// Alokasikan membagi qtyDasar ke beberapa lapisan menurut FEFO:
// kedaluwarsa terdekat dulu, lalu yang lebih dulu diterima, lalu id terkecil.
//
// Fungsi ini murni: tidak menyentuh database dan tidak mengubah slice masukan.
// Lapisan yang diberikan harus sudah disaring (hanya yang boleh dijual).
func Alokasikan(lapisan []Lapisan, qtyDasar float64) ([]Alokasi, error) {
	butuh := keMilli(qtyDasar)
	if butuh <= 0 {
		return nil, validasi("jumlah harus lebih dari 0")
	}

	urut := make([]Lapisan, len(lapisan))
	copy(urut, lapisan)
	sort.SliceStable(urut, func(i, j int) bool {
		a, b := urut[i], urut[j]
		if !a.TglKadaluarsa.Equal(b.TglKadaluarsa) {
			return a.TglKadaluarsa.Before(b.TglKadaluarsa)
		}
		if !a.TanggalTerima.Equal(b.TanggalTerima) {
			return a.TanggalTerima.Before(b.TanggalTerima)
		}
		return a.ReceiptID < b.ReceiptID
	})

	hasil := make([]Alokasi, 0, len(urut))
	for _, l := range urut {
		sisa := keMilli(l.QtySisa)
		if sisa <= 0 {
			continue
		}
		ambil := min(sisa, butuh)
		hasil = append(hasil, Alokasi{
			ReceiptID:         l.ReceiptID,
			QtyDasar:          dariMilli(ambil),
			HargaBeliPerDasar: l.HargaBeliPerDasar,
		})
		butuh -= ambil
		if butuh == 0 {
			return hasil, nil
		}
	}
	return nil, ErrStokTidakCukup
}