package cashier

import (
	"errors"
	"testing"
)

func TestHitungItem_StripParacetamol(t *testing.T) {
	// Sama dengan seed penjualan #1: 11 strip x Rp9.000, isi 10 tablet.
	u := SatuanJual{ID: 2, Nama: "strip", IsiPerSatuanDasar: 10, HargaJual: 9000}
	got, err := HitungItem(u, 11)
	if err != nil {
		t.Fatal(err)
	}
	if got.QtyDasar != 110 || got.HargaJualPerDasar != 900 || got.Subtotal != 99000 {
		t.Fatalf("hasil tidak sesuai: %+v", got)
	}
}

func TestHitungItem_Tolak(t *testing.T) {
	ok := SatuanJual{IsiPerSatuanDasar: 10, HargaJual: 9000}
	if _, err := HitungItem(ok, 0); !errors.Is(err, ErrValidasi) {
		t.Fatalf("qty 0: seharusnya ErrValidasi, dapat %v", err)
	}
	if _, err := HitungItem(SatuanJual{IsiPerSatuanDasar: 0, HargaJual: 1}, 1); !errors.Is(err, ErrValidasi) {
		t.Fatalf("isi 0: seharusnya ErrValidasi, dapat %v", err)
	}
}

func TestHitungPembayaran(t *testing.T) {
	tests := []struct {
		nama    string
		metode  string
		total   float64
		bayar   float64
		kembali float64
		salah   bool
	}{
		{"tunai pas", "TUNAI", 99000, 99000, 0, false},
		{"tunai ada kembalian", "TUNAI", 99000, 100000, 1000, false},
		{"tunai kurang", "TUNAI", 99000, 98000, 0, true},
		{"nontunai pas", "NONTUNAI", 22000, 22000, 0, false},
		{"nontunai lebih", "NONTUNAI", 22000, 25000, 0, true},
		{"huruf kecil dan spasi", " tunai ", 6000, 10000, 4000, false},
		{"metode tidak dikenal", "KARTU", 6000, 6000, 0, true},
		{"total nol", "TUNAI", 0, 0, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			got, err := HitungPembayaran(tc.metode, tc.total, tc.bayar)
			if tc.salah {
				if !errors.Is(err, ErrValidasi) {
					t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.kembali {
				t.Fatalf("kembalian %v, seharusnya %v", got, tc.kembali)
			}
		})
	}
}