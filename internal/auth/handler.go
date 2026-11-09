package auth

import (
	"net/http"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/view"
)

// Handler menangani request HTTP yang berkaitan dengan autentikasi.
// Saat ini hanya menampilkan halaman login; proses autentikasi belum diimplementasikan.
type Handler struct {
	View *view.Renderer
}

func NewHandler(renderer *view.Renderer) *Handler {
	return &Handler{View: renderer}
}

// Login menampilkan halaman login dan menyediakan respons sementara
// untuk POST sampai service serta repository autentikasi selesai dibuat.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.View.RenderLogin(w, http.StatusOK, view.LoginData{})
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			h.View.RenderLogin(w, http.StatusBadRequest, view.LoginData{
				Error: "Data formulir tidak dapat dibaca. Silakan coba kembali.",
			})
			return
		}

		// Jangan memberikan kesan bahwa login berhasil sebelum backend auth tersedia.
		h.View.RenderLogin(w, http.StatusNotImplemented, view.LoginData{
			Error:      "Tampilan login sudah tersedia, tetapi proses autentikasi belum dihubungkan ke backend.",
			Identifier: r.FormValue("identifier"),
		})
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
