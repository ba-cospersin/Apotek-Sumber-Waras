package cashier

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWriteServiceError(t *testing.T) {
	tests := []struct {
		nama   string
		err    error
		status int
	}{
		{"validasi", validasi("x"), http.StatusBadRequest},
		{"stok kurang", fmt.Errorf("%w untuk Paracetamol", ErrStokTidakCukup), http.StatusConflict},
		{"tidak ditemukan", ErrTidakDitemukan, http.StatusNotFound},
		{"error tak terduga", errors.New("db mati"), http.StatusInternalServerError},
		{"status sudah final", fmt.Errorf("%w (status sekarang LAYAK_JUAL)", ErrStatusSudahFinal), http.StatusConflict},
	}
	for _, tc := range tests {
		t.Run(tc.nama, func(t *testing.T) {
			w := httptest.NewRecorder()
			writeServiceError(w, tc.err)
			if w.Code != tc.status {
				t.Fatalf("status %d, seharusnya %d", w.Code, tc.status)
			}
		})
	}
}

func TestHandlerCariObat_KataKunciPendek(t *testing.T) {
	h := NewHandler(NewService(nil, nil)) // validasi terjadi sebelum repo dipakai
	req := httptest.NewRequest(http.MethodGet, "/kasir/obat?q=a", nil)
	w := httptest.NewRecorder()

	h.CariObat(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, seharusnya 400", w.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] == "" {
		t.Fatal("respons error harus berisi field error")
	}
}

func TestHandlerStruk_IDTidakValid(t *testing.T) {
	h := NewHandler(NewService(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/kasir/struk/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	h.Struk(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, seharusnya 400", w.Code)
	}
}

func TestHandlerTransaksi_TanpaLogin(t *testing.T) {
	h := NewHandler(NewService(nil, nil))
	req := httptest.NewRequest(http.MethodPost, "/kasir/transaksi", nil)
	w := httptest.NewRecorder()

	h.Transaksi(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, seharusnya 401", w.Code)
	}
}

func TestHandlerPenjualanRetur_IDTidakValid(t *testing.T) {
	h := NewHandler(NewService(nil, nil))
	req := httptest.NewRequest(http.MethodGet, "/retur/penjualan/abc", nil)
	req.SetPathValue("id", "abc")
	w := httptest.NewRecorder()

	h.PenjualanRetur(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, seharusnya 400", w.Code)
	}
}

func TestHandlerCatatRetur_TanpaLogin(t *testing.T) {
	h := NewHandler(NewService(nil, nil))
	req := httptest.NewRequest(http.MethodPost, "/retur", nil)
	w := httptest.NewRecorder()

	h.CatatRetur(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status %d, seharusnya 401", w.Code)
	}
}

func TestHandlerTetapkanKelayakan_IDTidakValid(t *testing.T) {
	h := NewHandler(NewService(nil, nil))
	req := httptest.NewRequest(http.MethodPost, "/retur/x/kelayakan", nil)
	req.SetPathValue("id", "x")
	w := httptest.NewRecorder()

	h.TetapkanKelayakan(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status %d, seharusnya 400", w.Code)
	}
}