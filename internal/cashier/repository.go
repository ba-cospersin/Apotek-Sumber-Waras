package cashier

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Repository menjadi tempat query SQL untuk modul kasir.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// escapeLike mencegah karakter % dan _ dari pengguna dianggap wildcard.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

// CariObat mencari obat aktif berdasarkan nama atau kode,
// beserta satuan jual yang aktif. Stok belum diisi di sini.
func (r *Repository) CariObat(ctx context.Context, kata string, limit int) ([]Obat, error) {
	pola := "%" + escapeLike(kata) + "%"

	rows, err := r.db.QueryContext(ctx,
		`SELECT id, kode, nama, satuan_dasar
		   FROM medicines
		  WHERE is_active = 1 AND (nama LIKE ? OR kode LIKE ?)
		  ORDER BY nama
		  LIMIT ?`,
		pola, pola, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mencari obat: %w", err)
	}

	hasil := make([]Obat, 0)
	indeks := make(map[int64]int)
	ids := make([]any, 0)
	for rows.Next() {
		var o Obat
		if err := rows.Scan(&o.ID, &o.Kode, &o.Nama, &o.SatuanDasar); err != nil {
			rows.Close()
			return nil, fmt.Errorf("gagal membaca obat: %w", err)
		}
		o.Satuan = make([]SatuanJual, 0)
		indeks[o.ID] = len(hasil)
		hasil = append(hasil, o)
		ids = append(ids, o.ID)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("gagal membaca obat: %w", err)
	}
	rows.Close()

	if len(hasil) == 0 {
		return hasil, nil
	}

	// Tanda tanya dibuat dari jumlah id; nilainya tetap lewat parameter.
	ph := strings.TrimRight(strings.Repeat("?,", len(ids)), ",")
	urows, err := r.db.QueryContext(ctx,
		`SELECT id, medicine_id, nama_satuan, isi_per_satuan_dasar, harga_jual
		   FROM medicine_units
		  WHERE is_active = 1 AND medicine_id IN (`+ph+`)
		  ORDER BY medicine_id, isi_per_satuan_dasar, id`,
		ids...,
	)
	if err != nil {
		return nil, fmt.Errorf("gagal mengambil satuan obat: %w", err)
	}
	defer urows.Close()

	for urows.Next() {
		var s SatuanJual
		var medicineID int64
		if err := urows.Scan(&s.ID, &medicineID, &s.Nama, &s.IsiPerSatuanDasar, &s.HargaJual); err != nil {
			return nil, fmt.Errorf("gagal membaca satuan obat: %w", err)
		}
		i := indeks[medicineID]
		hasil[i].Satuan = append(hasil[i].Satuan, s)
	}
	if err := urows.Err(); err != nil {
		return nil, fmt.Errorf("gagal membaca satuan obat: %w", err)
	}
	return hasil, nil
}