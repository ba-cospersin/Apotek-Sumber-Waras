package cashier

import (
	"errors"
	"testing"
	"time"
)

func tgl(tahun int, bulan time.Month, hari int) time.Time {
	return time.Date(tahun, bulan, hari, 0, 0, 0, 0, time.UTC)
}

func TestAlokasikan_SatuLapisanCukup(t *testing.T) {
	l := []Lapisan{{ReceiptID: 1, TglKadaluarsa: tgl(2027, 6, 1), TanggalTerima: tgl(2026, 1, 1), QtySisa: 100}}
	got, err := Alokasikan(l, 40)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ReceiptID != 1 || got[0].QtyDasar != 40 {
		t.Fatalf("hasil tidak sesuai: %+v", got)
	}
}

func TestAlokasikan_UrutKedaluwarsaTerdekat(t *testing.T) {
	l := []Lapisan{
		{ReceiptID: 2, TglKadaluarsa: tgl(2027, 12, 1), TanggalTerima: tgl(2026, 1, 1), QtySisa: 50},
		{ReceiptID: 1, TglKadaluarsa: tgl(2027, 3, 1), TanggalTerima: tgl(2026, 2, 1), QtySisa: 30},
	}
	got, err := Alokasikan(l, 40)
	if err != nil {
		t.Fatal(err)
	}
	// 30 dari receipt 1 (kedaluwarsa lebih dekat), sisanya 10 dari receipt 2.
	if len(got) != 2 || got[0].ReceiptID != 1 || got[0].QtyDasar != 30 ||
		got[1].ReceiptID != 2 || got[1].QtyDasar != 10 {
		t.Fatalf("hasil tidak sesuai: %+v", got)
	}
}

func TestAlokasikan_KedaluwarsaSamaUrutTerima(t *testing.T) {
	l := []Lapisan{
		{ReceiptID: 5, TglKadaluarsa: tgl(2027, 3, 1), TanggalTerima: tgl(2026, 5, 1), QtySisa: 10},
		{ReceiptID: 9, TglKadaluarsa: tgl(2027, 3, 1), TanggalTerima: tgl(2026, 2, 1), QtySisa: 10},
	}
	got, err := Alokasikan(l, 5)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].ReceiptID != 9 {
		t.Fatalf("seharusnya receipt 9 (diterima lebih dulu), dapat %d", got[0].ReceiptID)
	}
}

func TestAlokasikan_StokKurang(t *testing.T) {
	l := []Lapisan{{ReceiptID: 1, TglKadaluarsa: tgl(2027, 6, 1), TanggalTerima: tgl(2026, 1, 1), QtySisa: 10}}
	if _, err := Alokasikan(l, 11); !errors.Is(err, ErrStokTidakCukup) {
		t.Fatalf("seharusnya ErrStokTidakCukup, dapat %v", err)
	}
}

func TestAlokasikan_JumlahTidakValid(t *testing.T) {
	l := []Lapisan{{ReceiptID: 1, TglKadaluarsa: tgl(2027, 6, 1), QtySisa: 10}}
	for _, q := range []float64{0, -1} {
		if _, err := Alokasikan(l, q); !errors.Is(err, ErrValidasi) {
			t.Fatalf("qty %v: seharusnya ErrValidasi, dapat %v", q, err)
		}
	}
}

func TestAlokasikan_DesimalTidakSelisih(t *testing.T) {
	l := []Lapisan{
		{ReceiptID: 1, TglKadaluarsa: tgl(2027, 1, 1), QtySisa: 0.1},
		{ReceiptID: 2, TglKadaluarsa: tgl(2027, 2, 1), QtySisa: 0.2},
	}
	got, err := Alokasikan(l, 0.3)
	if err != nil {
		t.Fatalf("0.1 + 0.2 seharusnya cukup untuk 0.3: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("hasil tidak sesuai: %+v", got)
	}
}

func TestAlokasikan_TidakMengubahMasukan(t *testing.T) {
	l := []Lapisan{
		{ReceiptID: 2, TglKadaluarsa: tgl(2027, 12, 1), QtySisa: 5},
		{ReceiptID: 1, TglKadaluarsa: tgl(2027, 3, 1), QtySisa: 5},
	}
	if _, err := Alokasikan(l, 3); err != nil {
		t.Fatal(err)
	}
	if l[0].ReceiptID != 2 {
		t.Fatal("slice masukan tidak boleh diubah urutannya")
	}
}