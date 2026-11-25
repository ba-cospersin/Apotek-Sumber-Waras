package cashier

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
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

func TestIdDariNomor(t *testing.T) {
	tests := []struct {
		nomor string
		id    int64
		ok    bool
	}{
		{"42", 42, true},
		{" 42 ", 42, true},
		{"TRX-20261107-0042", 42, true},
		{"trx-20261107-0042", 42, true},
		{"TRX-20261107-42", 42, true},
		{"abc", 0, false},
		{"TRX-", 0, false},
		{"TRX-abc", 0, false},
		{"0", 0, false},
		{"-5", 0, false},
		{"", 0, false},
	}
	for _, tc := range tests {
		id, ok := idDariNomor(tc.nomor)
		if id != tc.id || ok != tc.ok {
			t.Errorf("idDariNomor(%q) = %d, %v; seharusnya %d, %v", tc.nomor, id, ok, tc.id, tc.ok)
		}
	}
}

func TestNoRetur(t *testing.T) {
	got := noRetur(time.Date(2026, 11, 6, 9, 0, 0, 0, time.UTC), 12)
	if got != "RET-20261106-0012" {
		t.Fatalf("nomor retur %q, seharusnya RET-20261106-0012", got)
	}
}

func TestCariPenjualanRetur_NomorTidakValid(t *testing.T) {
	s := NewService(nil, nil) // validasi terjadi sebelum repo dipakai
	_, err := s.CariPenjualanRetur(context.Background(), "abc")
	if !errors.Is(err, ErrValidasi) {
		t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
	}
}

func TestRiwayatRetur_LimitTidakValid(t *testing.T) {
	s := NewService(nil, nil)
	for _, limit := range []int{-1, riwayatMaks + 1} {
		if _, err := s.RiwayatRetur(context.Background(), limit); !errors.Is(err, ErrValidasi) {
			t.Fatalf("limit %d: seharusnya ErrValidasi, dapat %v", limit, err)
		}
	}
}