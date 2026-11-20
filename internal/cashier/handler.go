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
// Semua route wajib login. Penjualan hanya untuk KASIR,
// struk juga boleh dilihat OWNER.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *middleware.Auth) {
	kasir := func(next http.HandlerFunc) http.Handler {
		return auth.RequireLogin(
			middleware.RequireRole(model.RoleCashier)(next),
		)
	}
	kasirDanOwner := func(next http.HandlerFunc) http.Handler {
		return auth.RequireLogin(
			middleware.RequireRole(model.RoleCashier, model.RoleOwner)(next),
		)
	}

	mux.Handle("GET /kasir/obat", kasir(h.CariObat))
	mux.Handle("POST /kasir/transaksi", kasir(h.Transaksi))
	mux.Handle("GET /kasir/struk/{id}", kasirDanOwner(h.Struk))
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

	case errors.Is(err, ErrStokTidakCukup):
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