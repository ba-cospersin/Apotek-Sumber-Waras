package warehouse

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

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
// Semua route wajib login dan hanya boleh diakses role GUDANG dan OWNER.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, auth *middleware.Auth) {
	gudang := func(next http.HandlerFunc) http.Handler {
		return auth.RequireLogin(
			middleware.RequireRole(model.RoleWarehouse, model.RoleOwner)(next),
		)
	}

	mux.Handle("POST /gudang/penerimaan", gudang(h.CreatePenerimaan))
	mux.Handle("GET /gudang/penerimaan", gudang(h.ListPenerimaan))
	mux.Handle("GET /gudang/penerimaan/{id}", gudang(h.DetailPenerimaan))
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

// ListPenerimaan menangani GET /gudang/penerimaan.
//
// Query string (semuanya opsional):
// supplier_id, dari, sampai (YYYY-MM-DD), page, per_page.
func (h *Handler) ListPenerimaan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	supplierID, err := angkaQuery(q, "supplier_id")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	page, err := angkaQuery(q, "page")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	perPage, err := angkaQuery(q, "per_page")
	if err != nil {
		writeServiceError(w, err)
		return
	}

	hasil, err := h.svc.DaftarPenerimaan(r.Context(), FilterPenerimaan{
		SupplierID: supplierID,
		Dari:       q.Get("dari"),
		Sampai:     q.Get("sampai"),
		Page:       int(page),
		PerPage:    int(perPage),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, hasil)
}

// DetailPenerimaan menangani GET /gudang/penerimaan/{id}.
func (h *Handler) DetailPenerimaan(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id faktur harus berupa angka")
		return
	}

	hasil, err := h.svc.DetailPenerimaan(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": hasil})
}

// angkaQuery membaca parameter query berupa angka.
// Parameter yang tidak diisi dianggap 0.
func angkaQuery(q url.Values, kunci string) (int64, error) {
	nilai := q.Get(kunci)
	if nilai == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(nilai, 10, 64)
	if err != nil {
		return 0, validasi("%s harus berupa angka", kunci)
	}
	return n, nil
}

// writeServiceError mengubah error dari service menjadi response HTTP.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidasi):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrFakturDuplikat),
		errors.Is(err, ErrKedaluwarsaBeda):
		writeError(w, http.StatusConflict, err.Error())

	case errors.Is(err, ErrTidakDitemukan):
		writeError(w, http.StatusNotFound, err.Error())

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
