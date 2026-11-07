package warehouse

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/middleware"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

// Handler menangani request HTTP untuk modul gudang.
// Tidak menjalankan SQL dan tidak berisi aturan bisnis.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mendaftarkan semua route modul gudang ke router.
// Semua route wajib login dan hanya boleh diakses role GUDANG.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *middleware.Auth) {
	gudang := func(next http.HandlerFunc) http.Handler {
		return auth.RequireLogin(
			middleware.RequireRole(model.RoleWarehouse)(next),
		)
	}

	mux.Handle("POST /gudang/penerimaan", gudang(h.CreatePenerimaan))
}

// CreatePenerimaan menangani POST /gudang/penerimaan.
func (h *Handler) CreatePenerimaan(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	// Batasi ukuran body 1 MB dan tolak field yang tidak dikenal.
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var in PenerimaanInput
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "format JSON tidak valid: "+err.Error())
		return
	}

	hasil, err := h.svc.CatatPenerimaan(r.Context(), user.ID, in)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": hasil})
}

// writeServiceError mengubah error dari service menjadi response HTTP.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidasi):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrFakturDuplikat),
		errors.Is(err, ErrBatchDiblokir),
		errors.Is(err, ErrKedaluwarsaBeda):
		writeError(w, http.StatusConflict, err.Error())

	default:
		log.Printf("warehouse: kesalahan tak terduga: %v", err)
		writeError(w, http.StatusInternalServerError, "Terjadi kesalahan pada server")
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("warehouse: gagal menulis response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
