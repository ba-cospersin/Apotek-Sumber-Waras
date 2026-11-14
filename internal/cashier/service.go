package cashier

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const batasHasilCari = 20

// Stok adalah kebutuhan kasir dari modul gudang.
// Implementasinya nanti memanggil fungsi stok milik Hanan.
type Stok interface {
	StokTersedia(ctx context.Context, medicineID int64, hariIni time.Time) (float64, error)
}

// Service berisi aturan bisnis modul kasir.
// Tidak bergantung pada net/http dan tidak menjalankan SQL langsung.
type Service struct {
	repo *Repository
	stok Stok
	now  func() time.Time // diganti saat pengujian
}

func NewService(repo *Repository, stok Stok) *Service {
	return &Service{repo: repo, stok: stok, now: time.Now}
}

// hariIni mengembalikan tanggal hari ini tanpa jam.
func (s *Service) hariIni() time.Time {
	n := s.now()
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, n.Location())
}

// CariObat mencari obat untuk kasir, lengkap dengan satuan jual dan
// stok yang boleh dijual.
func (s *Service) CariObat(ctx context.Context, kata string) ([]Obat, error) {
	kata = strings.TrimSpace(kata)
	if len([]rune(kata)) < 2 {
		return nil, validasi("kata kunci minimal 2 karakter")
	}

	daftar, err := s.repo.CariObat(ctx, kata, batasHasilCari)
	if err != nil {
		return nil, err
	}

	hari := s.hariIni()
	for i := range daftar {
		stok, err := s.stok.StokTersedia(ctx, daftar[i].ID, hari)
		if err != nil {
			return nil, fmt.Errorf("gagal mengambil stok obat %d: %w", daftar[i].ID, err)
		}
		daftar[i].StokTersedia = stok
	}
	return daftar, nil
}