package warehouse

import (
	"encoding/json"
	"errors"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"

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

	// Persediaan dan pemblokiran batch.
	// Catatan: URL ini sementara. Rute final mengikuti kontrak Backend Lead
	// dan cukup diganti di sini.
	mux.Handle("GET /gudang/persediaan", gudang(h.ListPersediaan))
	mux.Handle("GET /gudang/persediaan/ringkasan", gudang(h.RingkasanPersediaan))
	mux.Handle("GET /gudang/persediaan/batch/{id}", gudang(h.DetailBatch))
	mux.Handle("POST /gudang/persediaan/batch/{id}/blokir", gudang(h.BlokirBatch))

	// Pemusnahan dan retur ke PBF (URL sementara, sama seperti di atas).
	mux.Handle("POST /gudang/tindak-lanjut", gudang(h.CatatTindakLanjut))
	mux.Handle("GET /gudang/tindak-lanjut", gudang(h.ListTindakLanjut))
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

// ListPersediaan menangani GET /gudang/persediaan.
//
// Query string (semuanya opsional):
// q (cari obat atau batch), status, semua=1 (sertakan batch yang stoknya habis),
// page, per_page.
func (h *Handler) ListPersediaan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

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

	semua := q.Get("semua") == "1" || q.Get("semua") == "true"

	hasil, err := h.svc.DaftarBatch(r.Context(), FilterBatch{
		Kata:           q.Get("q"),
		Status:         q.Get("status"),
		TermasukKosong: semua,
		Page:           int(page),
		PerPage:        int(perPage),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, hasil)
}

// RingkasanPersediaan menangani GET /gudang/persediaan/ringkasan.
func (h *Handler) RingkasanPersediaan(w http.ResponseWriter, r *http.Request) {
	hasil, err := h.svc.Ringkasan(r.Context())
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": hasil})
}

// DetailBatch menangani GET /gudang/persediaan/batch/{id}.
func (h *Handler) DetailBatch(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id batch harus berupa angka")
		return
	}

	hasil, err := h.svc.DetailBatch(r.Context(), id)
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": hasil})
}

// BlokirBatch menangani POST /gudang/persediaan/batch/{id}/blokir.
// Datanya berupa form biasa dengan satu field: alasan.
// Jika berhasil, pengguna diarahkan (303) ke halaman detail batch.
func (h *Handler) BlokirBatch(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id batch harus berupa angka")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid: "+err.Error())
		return
	}

	if err := h.svc.BlokirBatch(r.Context(), user.ID, id, r.PostFormValue("alasan")); err != nil {
		writeServiceError(w, err)
		return
	}

	http.Redirect(w, r, "/gudang/persediaan/batch/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

// CatatTindakLanjut menangani POST /gudang/tindak-lanjut.
//
// Data berupa form biasa. Field: batch_receipt_id, jenis, sumber_tindak_lanjut,
// alasan, qty_dasar, nilai_kompensasi (opsional), catatan (opsional).
// Angka memakai titik desimal. Jika berhasil, pengguna diarahkan (303) ke
// halaman detail batch supaya sisa stoknya terlihat.
func (h *Handler) CatatTindakLanjut(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.CurrentUser(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusBadRequest, "form tidak valid: "+err.Error())
		return
	}

	receiptID, err := angkaBulatForm(r, "batch_receipt_id")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	qty, err := angkaDesimalForm(r, "qty_dasar")
	if err != nil {
		writeServiceError(w, err)
		return
	}
	kompensasi, err := angkaDesimalForm(r, "nilai_kompensasi")
	if err != nil {
		writeServiceError(w, err)
		return
	}

	hasil, err := h.svc.CatatTindakLanjut(r.Context(), user.ID, TindakLanjutInput{
		BatchReceiptID:     receiptID,
		Jenis:              r.PostFormValue("jenis"),
		SumberTindakLanjut: r.PostFormValue("sumber_tindak_lanjut"),
		Alasan:             r.PostFormValue("alasan"),
		QtyDasar:           qty,
		NilaiKompensasi:    kompensasi,
		Catatan:            r.PostFormValue("catatan"),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	http.Redirect(w, r, "/gudang/persediaan/batch/"+strconv.FormatInt(hasil.BatchID, 10), http.StatusSeeOther)
}

// ListTindakLanjut menangani GET /gudang/tindak-lanjut.
//
// Query string (semuanya opsional): jenis, batch_id, dari, sampai, page, per_page.
func (h *Handler) ListTindakLanjut(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	batchID, err := angkaQuery(q, "batch_id")
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

	hasil, err := h.svc.DaftarTindakLanjut(r.Context(), FilterTindakLanjut{
		Jenis:   q.Get("jenis"),
		BatchID: batchID,
		Dari:    q.Get("dari"),
		Sampai:  q.Get("sampai"),
		Page:    int(page),
		PerPage: int(perPage),
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, hasil)
}

// angkaBulatForm membaca field form berupa bilangan bulat. Kosong dianggap 0.
func angkaBulatForm(r *http.Request, kunci string) (int64, error) {
	nilai := strings.TrimSpace(r.PostFormValue(kunci))
	if nilai == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(nilai, 10, 64)
	if err != nil {
		return 0, validasi("%s harus berupa angka bulat", kunci)
	}
	return n, nil
}

// angkaDesimalForm membaca field form berupa angka desimal (pakai titik).
// Kosong dianggap 0. Nilai NaN dan Inf ditolak.
func angkaDesimalForm(r *http.Request, kunci string) (float64, error) {
	nilai := strings.TrimSpace(r.PostFormValue(kunci))
	if nilai == "" {
		return 0, nil
	}
	n, err := strconv.ParseFloat(nilai, 64)
	if err != nil || math.IsNaN(n) || math.IsInf(n, 0) {
		return 0, validasi("%s harus berupa angka (gunakan titik sebagai desimal)", kunci)
	}
	return n, nil
}

// writeServiceError mengubah error dari service menjadi response HTTP.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrValidasi):
		writeError(w, http.StatusBadRequest, err.Error())

	case errors.Is(err, ErrFakturDuplikat),
		errors.Is(err, ErrKedaluwarsaBeda),
		errors.Is(err, ErrSudahDiblokir),
		errors.Is(err, ErrStokTidakCukup):
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
