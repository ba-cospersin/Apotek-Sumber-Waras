package cashier

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestJual_TolakInputTidakValid(t *testing.T) {
	s := NewService(nil, nil) // validasi terjadi sebelum repo dipakai
	ctx := context.Background()

	banyak := make([]ItemInput, maxItemTransaksi+1)
	for i := range banyak {
		banyak[i] = ItemInput{UnitID: 1, Qty: 1}
	}

	tests := []struct {
		nama string
		in   TransaksiInput
	}{
		{"keranjang kosong", TransaksiInput{MetodeBayar: "TUNAI", JumlahBayar: 1000}},
		{"metode kosong", TransaksiInput{Items: []ItemInput{{UnitID: 1, Qty: 1}}}},
		{"metode tidak dikenal", TransaksiInput{MetodeBayar: "KARTU", Items: []ItemInput{{UnitID: 1, Qty: 1}}}},
		{"unit_id nol", TransaksiInput{MetodeBayar: "TUNAI", Items: []ItemInput{{UnitID: 0, Qty: 1}}}},
		{"qty nol", TransaksiInput{MetodeBayar: "TUNAI", Items: []ItemInput{{UnitID: 1, Qty: 0}}}},
		{"qty negatif", TransaksiInput{MetodeBayar: "TUNAI", Items: []ItemInput{{UnitID: 1, Qty: -2}}}},
		{"terlalu banyak item", TransaksiInput{MetodeBayar: "TUNAI", Items: banyak}},
	}

	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			_, err := s.Jual(ctx, 1, tc.in)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
			}
		})
	}
}

func TestJual_UserTidakDikenali(t *testing.T) {
	s := NewService(nil, nil)
	_, err := s.Jual(context.Background(), 0, TransaksiInput{})
	if err == nil {
		t.Fatal("seharusnya error untuk user 0")
	}
}

func TestCariObat_KataKunciTerlaluPendek(t *testing.T) {
	s := NewService(nil, nil)
	_, err := s.CariObat(context.Background(), " a ")
	if !errors.Is(err, ErrValidasi) {
		t.Fatalf("seharusnya ErrValidasi, dapat %v", err)
	}
}

func TestLengkapiStruk_SeedPenjualan1(t *testing.T) {
	// Sama dengan seed penjualan #1: 11 strip (110 tablet) @ Rp900 per tablet.
	st := &Struk{
		Total:       99000,
		JumlahBayar: 100000,
		Items: []StrukItem{
			{NamaObat: "Paracetamol", NamaSatuan: "strip", Qty: 11, qtyDasar: 110, hargaDasar: 900},
		},
	}

	lengkapiStruk(st)

	it := st.Items[0]
	if it.Subtotal != 99000 || it.HargaSatuan != 9000 {
		t.Fatalf("baris struk tidak sesuai: %+v", it)
	}
	if st.Kembalian != 1000 {
		t.Fatalf("kembalian %v, seharusnya 1000", st.Kembalian)
	}
}

func TestNoTransaksi(t *testing.T) {
	got := noTransaksi(time.Date(2026, 11, 7, 10, 42, 0, 0, time.UTC), 42)
	if got != "TRX-20261107-0042" {
		t.Fatalf("nomor transaksi %q, seharusnya TRX-20261107-0042", got)
	}
}