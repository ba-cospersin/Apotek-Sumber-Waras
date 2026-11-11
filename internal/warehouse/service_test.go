package warehouse

import (
	"errors"
	"testing"
	"time"
)

// serviceUji membuat Service dengan "hari ini" yang tetap: 7 November 2026.
// repo tidak dipakai karena validasiPenerimaan tidak menyentuh database.
func serviceUji() *Service {
	return &Service{
		now: func() time.Time {
			return time.Date(2026, 11, 7, 10, 0, 0, 0, time.Local)
		},
	}
}

// inputValid mengembalikan input yang lolos validasi.
func inputValid() PenerimaanInput {
	return PenerimaanInput{
		SupplierID: 1,
		NoFaktur:   "FK-001",
		TglFaktur:  "2026-11-07",
		Items: []ItemPenerimaanInput{
			{
				MedicineID:    1,
				NoBatch:       "AMX-C-1126",
				TglKadaluarsa: "2028-01-31",
				Qty:           10,
				HargaBeli:     5000,
			},
		},
	}
}

func TestValidasiPenerimaan_Valid(t *testing.T) {
	noFaktur, _, items, err := serviceUji().validasiPenerimaan(inputValid())
	if err != nil {
		t.Fatalf("seharusnya valid, tapi error: %v", err)
	}
	if noFaktur != "FK-001" || len(items) != 1 {
		t.Fatalf("hasil tidak sesuai: %q, %d item", noFaktur, len(items))
	}
}

func TestValidasiPenerimaan_Tolak(t *testing.T) {
	kasus := []struct {
		nama string
		ubah func(in *PenerimaanInput)
	}{
		{"supplier kosong", func(in *PenerimaanInput) { in.SupplierID = 0 }},
		{"no faktur kosong", func(in *PenerimaanInput) { in.NoFaktur = "   " }},
		{"tgl faktur salah format", func(in *PenerimaanInput) { in.TglFaktur = "07-11-2026" }},
		{"tgl faktur masa depan", func(in *PenerimaanInput) { in.TglFaktur = "2026-11-08" }},
		{"items kosong", func(in *PenerimaanInput) { in.Items = nil }},
		{"medicine_id kosong", func(in *PenerimaanInput) { in.Items[0].MedicineID = 0 }},
		{"unit_id negatif", func(in *PenerimaanInput) { in.Items[0].UnitID = -1 }},
		{"no batch kosong", func(in *PenerimaanInput) { in.Items[0].NoBatch = "" }},
		{"kedaluwarsa salah format", func(in *PenerimaanInput) { in.Items[0].TglKadaluarsa = "besok" }},
		{"sudah kedaluwarsa", func(in *PenerimaanInput) { in.Items[0].TglKadaluarsa = "2026-11-07" }},
		{"qty nol", func(in *PenerimaanInput) { in.Items[0].Qty = 0 }},
		{"qty negatif", func(in *PenerimaanInput) { in.Items[0].Qty = -5 }},
		{"harga negatif", func(in *PenerimaanInput) { in.Items[0].HargaBeli = -1 }},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			in := inputValid()
			k.ubah(&in)

			_, _, _, err := serviceUji().validasiPenerimaan(in)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}

func TestBulatkan(t *testing.T) {
	if got := bulatkan(1666.666666, 2); got != 1666.67 {
		t.Fatalf("bulatkan 2 desimal: dapat %v", got)
	}
	if got := bulatkan(2.00049, 3); got != 2.0 {
		t.Fatalf("bulatkan 3 desimal: dapat %v", got)
	}
}

func TestSiapkanFilter_Bawaan(t *testing.T) {
	q, page, perPage, err := siapkanFilter(FilterPenerimaan{})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if page != 1 || perPage != 20 || q.Limit != 20 || q.Offset != 0 {
		t.Fatalf("bawaan salah: page=%d perPage=%d limit=%d offset=%d",
			page, perPage, q.Limit, q.Offset)
	}
}

func TestSiapkanFilter_Halaman(t *testing.T) {
	q, page, perPage, err := siapkanFilter(FilterPenerimaan{Page: 3, PerPage: 10})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if page != 3 || perPage != 10 || q.Limit != 10 || q.Offset != 20 {
		t.Fatalf("paginasi salah: page=%d perPage=%d limit=%d offset=%d",
			page, perPage, q.Limit, q.Offset)
	}
}

func TestSiapkanFilter_PerPageDibatasi(t *testing.T) {
	_, _, perPage, err := siapkanFilter(FilterPenerimaan{PerPage: 5000})
	if err != nil {
		t.Fatalf("seharusnya valid: %v", err)
	}
	if perPage != 100 {
		t.Fatalf("per_page harus dibatasi 100, dapat %d", perPage)
	}
}

func TestSiapkanFilter_Tolak(t *testing.T) {
	kasus := []struct {
		nama   string
		filter FilterPenerimaan
	}{
		{"supplier negatif", FilterPenerimaan{SupplierID: -1}},
		{"page negatif", FilterPenerimaan{Page: -1}},
		{"per_page negatif", FilterPenerimaan{PerPage: -5}},
		{"dari salah format", FilterPenerimaan{Dari: "07/11/2026"}},
		{"sampai salah format", FilterPenerimaan{Sampai: "kemarin"}},
		{"dari lebih besar dari sampai", FilterPenerimaan{Dari: "2026-11-10", Sampai: "2026-11-01"}},
	}

	for _, k := range kasus {
		t.Run(k.nama, func(t *testing.T) {
			_, _, _, err := siapkanFilter(k.filter)
			if !errors.Is(err, ErrValidasi) {
				t.Fatalf("seharusnya ErrValidasi, dapat: %v", err)
			}
		})
	}
}
