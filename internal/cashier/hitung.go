package cashier

import (
	"math"
	"strings"
)

const (
	MetodeTunai    = "TUNAI"
	MetodeNontunai = "NONTUNAI"
)

// bulatkan membulatkan x ke sejumlah desimal tertentu.
func bulatkan(x float64, desimal int) float64 {
	p := math.Pow10(desimal)
	return math.Round(x*p) / p
}

// HasilItem adalah hasil hitung satu baris keranjang.
type HasilItem struct {
	QtyDasar          float64 // disimpan ke sale_items.qty_dasar
	HargaJualPerDasar float64 // disimpan ke sale_items.harga_jual_snapshot
	Subtotal          float64 // HargaJual satuan x qtyInput
}

// HitungItem mengonversi jumlah dari satuan pilihan ke satuan dasar
// dan menghitung subtotalnya.
func HitungItem(u SatuanJual, qtyInput float64) (HasilItem, error) {
	qtyInput = bulatkan(qtyInput, 3)
	if qtyInput <= 0 {
		return HasilItem{}, validasi("jumlah harus lebih dari 0")
	}
	if u.IsiPerSatuanDasar <= 0 {
		return HasilItem{}, validasi("konversi satuan tidak valid")
	}
	return HasilItem{
		QtyDasar:          bulatkan(qtyInput*u.IsiPerSatuanDasar, 3),
		HargaJualPerDasar: bulatkan(u.HargaJual/u.IsiPerSatuanDasar, 2),
		Subtotal:          bulatkan(u.HargaJual*qtyInput, 2),
	}, nil
}

// HitungPembayaran memeriksa pembayaran dan mengembalikan kembalian.
// Tunai: jumlah bayar harus >= total. Nontunai: jumlah bayar harus sama dengan total.
func HitungPembayaran(metode string, total, jumlahBayar float64) (float64, error) {
	total = bulatkan(total, 2)
	jumlahBayar = bulatkan(jumlahBayar, 2)

	if total <= 0 {
		return 0, validasi("total transaksi harus lebih dari 0")
	}

	switch strings.ToUpper(strings.TrimSpace(metode)) {
	case MetodeTunai:
		if jumlahBayar < total {
			return 0, validasi("jumlah bayar kurang Rp%.2f dari total", total-jumlahBayar)
		}
		return bulatkan(jumlahBayar-total, 2), nil
	case MetodeNontunai:
		if jumlahBayar != total {
			return 0, validasi("pembayaran nontunai harus sama dengan total")
		}
		return 0, nil
	default:
		return 0, validasi("metode pembayaran harus %s atau %s", MetodeTunai, MetodeNontunai)
	}
}
