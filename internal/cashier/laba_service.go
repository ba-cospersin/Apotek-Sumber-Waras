package cashier

import (
	"context"
	"strings"
	"time"
)

const (
	layoutTanggalInput = "2006-01-02"
	maxRentangHari     = 366
	limitBawaan        = 100
	limitMaks          = 500
)

// Hitung mengisi nilai turunan dari komponen mentah.
func (l *Laba) Hitung() {
	l.LabaKotor = bulatkan(l.Pendapatan-l.HPP, 2)
	l.LabaRetur = bulatkan(l.ReturRefund-l.ReturHPP, 2)
	l.LabaBersih = bulatkan(l.LabaKotor-l.LabaRetur, 2)
}

// rentangLaporan menerjemahkan dari/sampai (YYYY-MM-DD, inklusif) menjadi batas waktu [awal, akhir).
// Kosong berarti: dari awal bulan ini sampai hari ini.
func rentangLaporan(dari, sampai string, hariIni time.Time) (time.Time, time.Time, error) {
	lokasi := hariIni.Location()

	awal := time.Date(hariIni.Year(), hariIni.Month(), 1, 0, 0, 0, 0, lokasi)
	if d := strings.TrimSpace(dari); d != "" {
		t, err := time.ParseInLocation(layoutTanggalInput, d, lokasi)
		if err != nil {
			return time.Time{}, time.Time{}, validasi("dari harus berformat YYYY-MM-DD")
		}
		awal = t
	}

	terakhir := hariIni
	if sm := strings.TrimSpace(sampai); sm != "" {
		t, err := time.ParseInLocation(layoutTanggalInput, sm, lokasi)
		if err != nil {
			return time.Time{}, time.Time{}, validasi("sampai harus berformat YYYY-MM-DD")
		}
		terakhir = t
	}

	if terakhir.Before(awal) {
		return time.Time{}, time.Time{}, validasi("sampai tidak boleh sebelum dari")
	}
	if terakhir.After(awal.AddDate(0, 0, maxRentangHari-1)) {
		return time.Time{}, time.Time{}, validasi("rentang tanggal maksimal %d hari", maxRentangHari)
	}
	return awal, terakhir.AddDate(0, 0, 1), nil
}

// normalisasiHalaman memeriksa limit dan offset. Limit 0 berarti bawaan.
func normalisasiHalaman(limit, offset int) (int, int, error) {
	if limit == 0 {
		limit = limitBawaan
	}
	if limit < 0 || limit > limitMaks {
		return 0, 0, validasi("limit harus antara 1 dan %d", limitMaks)
	}
	if offset < 0 {
		return 0, 0, validasi("offset tidak boleh negatif")
	}
	return limit, offset, nil
}

// ringkasan mengambil dan melengkapi ringkasan untuk rentang [awal, akhir).
func (s *Service) ringkasan(ctx context.Context, awal, akhir time.Time) (*RingkasanLaba, error) {
	r, err := s.repo.RingkasanLaba(ctx, awal, akhir)
	if err != nil {
		return nil, err
	}
	r.Hitung()
	r.Dari = awal.Format(layoutTanggalInput)
	r.Sampai = akhir.AddDate(0, 0, -1).Format(layoutTanggalInput)
	return r, nil
}

// RingkasanLaba menghitung ringkasan laba untuk satu periode. 
// Dipakai juga oleh Backend Owner untuk dashboard.
func (s *Service) RingkasanLaba(ctx context.Context, dari, sampai string) (*RingkasanLaba, error) {
	awal, akhir, err := rentangLaporan(dari, sampai, s.hariIni())
	if err != nil {
		return nil, err
	}
	return s.ringkasan(ctx, awal, akhir)
}

// LaporanLaba menyiapkan isi halaman laporan laba: 
// ringkasan seluruh periode dan daftar transaksi (terbaru dulu, dengan paging).
func (s *Service) LaporanLaba(
	ctx context.Context, dari, sampai string, limit, offset int,
) (*LaporanLaba, error) {
	awal, akhir, err := rentangLaporan(dari, sampai, s.hariIni())
	if err != nil {
		return nil, err
	}
	limit, offset, err = normalisasiHalaman(limit, offset)
	if err != nil {
		return nil, err
	}

	daftar, err := s.repo.LabaPerTransaksi(ctx, awal, akhir, limit, offset)
	if err != nil {
		return nil, err
	}
	for i := range daftar {
		daftar[i].Hitung()
	}

	ring, err := s.ringkasan(ctx, awal, akhir)
	if err != nil {
		return nil, err
	}

	return &LaporanLaba{
		Ringkasan: *ring,
		Transaksi: daftar,
		Limit:     limit,
		Offset:    offset,
	}, nil
}