package cashier

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestLabaHitung(t *testing.T) {
	l := Laba{Pendapatan: 100000, HPP: 60000, ReturRefund: 9000, ReturHPP: 6000}
	l.Hitung()

	if l.LabaKotor != 40000 || l.LabaRetur != 3000 || l.LabaBersih != 37000 {
		t.Fatalf("hasil tidak sesuai: %+v", l)
	}
}

func TestLabaHitung_SeedTransaksi1(t *testing.T) {
	// Seed penjualan #1: total 99.000, HPP 100x500 + 10x520 = 55.200.
	// Retur 2 tablet: refund 1.800, HPP retur 2x500 = 1.000.
	l := Laba{Pendapatan: 99000, HPP: 55200, ReturRefund: 1800, ReturHPP: 1000}
	l.Hitung()

	if l.LabaKotor != 43800 || l.LabaRetur != 800 || l.LabaBersih != 43000 {
		t.Fatalf("hasil tidak sesuai: %+v", l)
	}
}

func TestLabaHitung_TanpaRetur(t *testing.T) {
	l := Laba{Pendapatan: 22000, HPP: 12500}
	l.Hitung()

	if l.LabaKotor != 9500 || l.LabaRetur != 0 || l.LabaBersih != 9500 {
		t.Fatalf("hasil tidak sesuai: %+v", l)
	}
}

func TestRentangLaporan_Bawaan(t *testing.T) {
	hari := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

	awal, akhir, err := rentangLaporan("", "", hari)
	if err != nil {
		t.Fatal(err)
	}
	if !awal.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("awal %v", awal)
	}
	if !akhir.Equal(time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("akhir (eksklusif) %v", akhir)
	}
}

func TestRentangLaporan_Eksplisit(t *testing.T) {
	hari := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

	awal, akhir, err := rentangLaporan("2026-11-10", "2026-11-12", hari)
	if err != nil {
		t.Fatal(err)
	}
	if !awal.Equal(time.Date(2026, 11, 10, 0, 0, 0, 0, time.UTC)) ||
		!akhir.Equal(time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("rentang salah: %v - %v", awal, akhir)
	}
}

func TestRentangLaporan_Tolak(t *testing.T) {
	hari := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		nama   string
		dari   string
		sampai string
	}{
		{"format dari salah", "10-11-2026", ""},
		{"format sampai salah", "", "besok"},
		{"sampai sebelum dari", "2026-11-12", "2026-11-10"},
		{"rentang terlalu panjang", "2026-01-01", "2027-01-02"},
	}
	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			_, _, err := rentangLaporan(tc.dari, tc.sampai, hari)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
			}
		})
	}
}

func TestRentangLaporan_BatasTepat(t *testing.T) {
	hari := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)

	// 1 Jan 2026 sampai 1 Jan 2027 = 366 hari inklusif, masih diizinkan.
	if _, _, err := rentangLaporan("2026-01-01", "2027-01-01", hari); err != nil {
		t.Fatalf("366 hari seharusnya diizinkan: %v", err)
	}
}

func TestNormalisasiHalaman(t *testing.T) {
	limit, offset, err := normalisasiHalaman(0, 0)
	if err != nil || limit != limitBawaan || offset != 0 {
		t.Fatalf("bawaan salah: %d %d %v", limit, offset, err)
	}

	for _, c := range [][2]int{{-1, 0}, {limitMaks + 1, 0}, {10, -1}} {
		if _, _, err := normalisasiHalaman(c[0], c[1]); !errors.Is(err, ErrValidasi) {
			t.Fatalf("limit %d offset %d: seharusnya ErrValidasi, dapat %v", c[0], c[1], err)
		}
	}
}

func TestLaporanLaba_TolakTanggalTidakValid(t *testing.T) {
	s := NewService(nil, nil) // validasi terjadi sebelum repo dipakai
	_, err := s.LaporanLaba(context.Background(), "bukan-tanggal", "", 0, 0)
	if !errors.Is(err, ErrValidasi) {
		t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
	}
}