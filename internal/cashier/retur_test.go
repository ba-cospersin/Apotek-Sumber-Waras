package cashier

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestSisaRetur(t *testing.T) {
	tests := []struct {
		nama          string
		dialokasikan  float64
		sudahDiretur  float64
		seharusnya    float64
	}{
		{"belum ada retur", 10, 0, 10},
		{"retur sebagian", 10, 2, 8},
		{"retur habis", 10, 10, 0},
		{"melebihi tidak jadi negatif", 10, 12, 0},
		{"desimal tanpa selisih", 0.3, 0.1, 0.2},
	}
	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			got := sisaRetur(tc.dialokasikan, tc.sudahDiretur)
			if got != tc.seharusnya {
				t.Fatalf("sisa %v, seharusnya %v", got, tc.seharusnya)
			}
		})
	}
}

func TestHitungRefund(t *testing.T) {
	// Sama dengan seed retur #1: 2 tablet x Rp900.
	if got := hitungRefund(2, 900); got != 1800 {
		t.Fatalf("refund %v, seharusnya 1800", got)
	}
	// Pembulatan 2 desimal.
	if got := hitungRefund(3, 333.33); got != 999.99 {
		t.Fatalf("refund %v, seharusnya 999.99", got)
	}
}

func inputReturValid() ReturInput {
	return ReturInput{
		SaleItemBatchID:   1,
		QtyDasar:          2,
		JenisPengembalian: JenisSalahMembeli,
		Alasan:            "Salah membeli",
	}
}

func TestCatatRetur_TolakInputTidakValid(t *testing.T) {
	s := NewService(nil, nil) // validasi terjadi sebelum repo dipakai
	ctx := context.Background()

	tests := []struct {
		nama  string
		ubah  func(*ReturInput)
	}{
		{"batch id nol", func(i *ReturInput) { i.SaleItemBatchID = 0 }},
		{"jumlah nol", func(i *ReturInput) { i.QtyDasar = 0 }},
		{"jumlah negatif", func(i *ReturInput) { i.QtyDasar = -1 }},
		{"jenis kosong", func(i *ReturInput) { i.JenisPengembalian = "" }},
		{"jenis tidak dikenal", func(i *ReturInput) { i.JenisPengembalian = "LAIN" }},
		{"alasan kosong", func(i *ReturInput) { i.Alasan = "   " }},
		{"alasan terlalu panjang", func(i *ReturInput) { i.Alasan = strings.Repeat("a", maxPanjangAlasan+1) }},
		{"kondisi terlalu panjang", func(i *ReturInput) { i.KondisiObat = strings.Repeat("a", maxPanjangKondisi+1) }},
		{"status tidak dikenal", func(i *ReturInput) { i.StatusKelayakan = "SELESAI" }},
	}

	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			in := inputReturValid()
			tc.ubah(&in)
			_, err := s.CatatRetur(ctx, 1, in)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
			}
		})
	}
}

func TestCatatRetur_UserTidakDikenali(t *testing.T) {
	s := NewService(nil, nil)
	if _, err := s.CatatRetur(context.Background(), 0, inputReturValid()); err == nil {
		t.Fatal("seharusnya error untuk user 0")
	}
}

func TestTetapkanKelayakan_TolakInputTidakValid(t *testing.T) {
	s := NewService(nil, nil)
	ctx := context.Background()

	tests := []struct {
		nama   string
		id     int64
		status string
	}{
		{"id nol", 0, StatusLayakJual},
		{"status kosong", 1, ""},
		{"status menunggu tidak boleh ditetapkan", 1, StatusMenunggu},
		{"status tidak dikenal", 1, "ASAL"},
	}
	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			_, err := s.TetapkanKelayakan(ctx, tc.id, tc.status)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
			}
		})
	}
}