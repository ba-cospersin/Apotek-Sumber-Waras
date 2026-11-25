package cashier

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/middleware"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

// Handler menangani request HTTP untuk modul kasir.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mendaftarkan route modul kasir ke router.
// Semua route wajib login. Penjualan dan pencatatan retur boleh diakses KASIR
// dan OWNER. Penetapan kelayakan retur dan laporan laba hanya untuk OWNER.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *middleware.Auth) {
	kasir := func(next http.HandlerFunc) http.Handler {
		return auth.RequireLogin(
			middleware.RequireRole(model.RoleCashier, model.RoleOwner)(next),
		)
	}
	kasirDanOwner := kasir // nama lama dipertahankan untuk route struk
	owner := func(next http.HandlerFunc) http.Handler {
		return auth.RequireLogin(
			middleware.RequireRole(model.RoleOwner)(next),
		)
	}

	mux.Handle("GET /kasir/obat", kasir(h.CariObat))
	mux.Handle("POST /kasir/transaksi", kasir(h.Transaksi))
	mux.Handle("GET /kasir/struk/{id}", kasirDanOwner(h.Struk))
	mux.Handle("GET /retur/penjualan/{id}", kasir(h.PenjualanRetur))
	mux.Handle("POST /retur", kasir(h.CatatRetur))
	mux.Handle("GET /retur/riwayat", kasir(h.RiwayatRetur))
	mux.Handle("POST /retur/{id}/kelayakan", owner(h.TetapkanKelayakan))
	mux.Handle("GET /laporan/laba", owner(h.LaporanLaba))
}

// CariObat menangani GET /kasir/obat?q=para
func (h *Handler) CariObat(w http.ResponseWriter, r *http.Request) {
	daftar, err := h.svc.CariObat(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": daftar})
}

// Transaksi menangani POST /kasir/transaksi.
func (h *Handler) Transaksi(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Batasi ukuran body 1 MB dan tolak field yang tidak dikenal.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var in TransaksiInput
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "format JSON tidak valid: "+err.Error())
		return
	}

	hasil, err := h.svc.Jual(r.Context(), user.ID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": hasil})
}

// Struk menangani GET /kasir/struk/{id}.
func (h *Handler) Struk(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id penjualan harus berupa angka")
		return
	}

	st, err := h.svc.Struk(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": st})
}

// writeServiceError mengubah error dari service menjadi response HTTP.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidasi):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrStokTidakCukup),
		errors.Is(err, ErrStatusSudahFinal):
		writeError(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())

	default:
		log.Printf("cashier: kesalahan tak terduga: %v", err)
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("cashier: gagal menulis response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

// bacaJSON membaca body JSON dengan batas 1 MB dan menolak field yang tidak dikenal.
// Jika gagal, respons 400 sudah ditulis dan hasilnya false.
func bacaJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "format JSON tidak valid: "+err.Error())
		return false
	}
	return true
}

// PenjualanRetur menangani GET /retur/penjualan/{id}.
// {id} boleh berupa angka atau nomor transaksi berformat TRX-YYYYMMDD-NNNN.
func (h *Handler) PenjualanRetur(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.CariPenjualanRetur(r.Context(), r.PathValue("id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": p})
}

// CatatRetur menangani POST /retur.
func (h *Handler) CatatRetur(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var in ReturInput
	if !bacaJSON(w, r, &in) {
		return
	}

	hasil, err := h.svc.CatatRetur(r.Context(), user.ID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"data": hasil})
}

// TetapkanKelayakan menangani POST /retur/{id}/kelayakan.
func (h *Handler) TetapkanKelayakan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id retur harus berupa angka")
		return
	}

	var in KelayakanInput
	if !bacaJSON(w, r, &in) {
		return
	}

	hasil, err := h.svc.TetapkanKelayakan(r.Context(), id, in.StatusKelayakan)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": hasil})
}

// LaporanLaba menangani GET /laporan/laba?dari=YYYY-MM-DD&sampai=YYYY-MM-DD&limit=&offset=
func (h *Handler) LaporanLaba(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	limit, err := angkaQuery(q.Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "limit harus berupa angka")
		return
	}
	offset, err := angkaQuery(q.Get("offset"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "offset harus berupa angka")
		return
	}

	hasil, err := h.svc.LaporanLaba(r.Context(), q.Get("dari"), q.Get("sampai"), limit, offset)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": hasil})
}

// angkaQuery mengubah nilai query string menjadi int. Kosong berarti 0.
func angkaQuery(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// RiwayatRetur menangani GET /retur/riwayat?limit=10
func (h *Handler) RiwayatRetur(w http.ResponseWriter, r *http.Request) {
	limit, err := angkaQuery(r.URL.Query().Get("limit"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "limit harus berupa angka")
		return
	}

	daftar, err := h.svc.RiwayatRetur(r.Context(), limit)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": daftar})
}
