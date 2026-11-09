package router

import (
	"context"
	"database/sql"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/auth"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/view"
)

type previewPageConfig struct {
	name        string
	title       string
	subtitle    string
	css         string
	js          string
	active      string
	defaultRole string
}

var previewPages = []previewPageConfig{
	{name: "dashboard", title: "Dashboard", subtitle: "Ringkasan pengelolaan stok apotek", css: "/static/css/dashboard.css", active: "dashboard", defaultRole: model.RoleOwner},
	{name: "kasir", title: "Kasir", subtitle: "Pencatatan transaksi penjualan", css: "/static/css/kasir.css", js: "/static/js/kasir.js", active: "kasir", defaultRole: model.RoleCashier},
	{name: "penerimaan", title: "Penerimaan Barang", subtitle: "Pencatatan penerimaan obat dari PBF", css: "/static/css/penerimaan.css", active: "penerimaan", defaultRole: model.RoleWarehouse},
	{name: "cek_stok", title: "Cek Stok & Batch", subtitle: "Penelusuran persediaan berdasarkan batch fisik", css: "/static/css/cek_stok.css", active: "cek_stok", defaultRole: model.RoleWarehouse},
	{name: "retur", title: "Retur Konsumen", subtitle: "Pencatatan dan pemeriksaan pengembalian obat", css: "/static/css/retur.css", active: "retur", defaultRole: model.RoleCashier},
	{name: "stock_opname", title: "Stock Opname", subtitle: "Rekonsiliasi stok fisik dengan stok sistem", css: "/static/css/stock_opname.css", js: "/static/js/stock_opname.js", active: "stock_opname", defaultRole: model.RoleWarehouse},
	{name: "kedaluwarsa", title: "Kedaluwarsa & Blokir", subtitle: "Tindak lanjut batch yang tidak boleh dijual", css: "/static/css/kedaluwarsa.css", active: "kedaluwarsa", defaultRole: model.RoleWarehouse},
	{name: "laporan", title: "Laporan", subtitle: "Laporan persediaan, penjualan, dan kerugian", css: "/static/css/laporan.css", active: "laporan", defaultRole: model.RoleOwner},
	{name: "data_master", title: "Data Master", subtitle: "Pengelolaan obat, satuan, harga jual, dan PBF", css: "/static/css/data_master.css", js: "/static/js/data_master.js", active: "data_master", defaultRole: model.RoleOwner},
	{name: "manajemen_pengguna", title: "Manajemen Pengguna", subtitle: "Pengelolaan akun internal apotek", css: "/static/css/manajemen_pengguna.css", active: "manajemen_pengguna", defaultRole: model.RoleOwner},
	{name: "struk_penjualan", title: "Struk Penjualan", subtitle: "Pratinjau bukti transaksi", css: "/static/css/struk_penjualan.css", active: "kasir", defaultRole: model.RoleCashier},
}

func New(db *sql.DB) (*http.ServeMux, error) {
	renderer, err := view.NewRenderer()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	staticFiles := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", staticFiles))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	})

	// Handler login tetap disediakan dan dimiliki modul Backend Owner.
	// Backend Lead hanya mendaftarkan handler yang sudah tersedia.
	authHandler := auth.NewHandler(renderer)
	mux.HandleFunc("/login", authHandler.Login)

	// Route preview untuk pemeriksaan frontend lokal. Semua endpoint ini
	// memakai data statis dan sengaja belum dipasangi middleware autentikasi.
	for _, config := range previewPages {
		pageConfig := config
		path := "/__preview/" + pageConfig.name
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != path {
				http.NotFound(w, r)
				return
			}
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", http.MethodGet)
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}

			role := requestedPreviewRole(r, pageConfig.defaultRole)
			user, roleLabel, avatar := previewUser(role)
			routes := make(map[string]string, len(previewPages))
			for _, item := range previewPages {
				routes[item.name] = "/__preview/" + item.name + "?role=" + url.QueryEscape(role)
			}

			renderer.RenderPage(w, http.StatusOK, pageConfig.name, view.PageData{
				Title:       pageConfig.title,
				Subtitle:    pageConfig.subtitle,
				PageCSS:     pageConfig.css,
				PageJS:      pageConfig.js,
				ActivePage:  pageConfig.active,
				CurrentDate: "Sabtu, 7 November 2026",
				RoleLabel:   roleLabel,
				AvatarText:  avatar,
				HomeURL:     previewHomeURL(role),
				User:        user,
				Routes:      routes,
			})
		})
	}

	// Endpoint pemeriksaan koneksi server dan database.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Server and database are running"))
	})

	return mux, nil
}

func requestedPreviewRole(r *http.Request, fallback string) string {
	role := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("role")))
	switch role {
	case model.RoleOwner, model.RoleCashier, model.RoleWarehouse:
		return role
	default:
		return fallback
	}
}

func previewUser(role string) (model.User, string, string) {
	user := model.User{ID: 1, IsActive: true, Role: role}
	switch role {
	case model.RoleCashier:
		user.Nama, user.Username = "Budi Santoso", "kasir"
		return user, "Kasir", "B"
	case model.RoleWarehouse:
		user.Nama, user.Username = "Siti Rahmawati", "gudang"
		return user, "Petugas Gudang", "S"
	default:
		user.Role = model.RoleOwner
		user.Nama, user.Username = "apt. Hendra Kurniawan, S.Farm", "owner"
		return user, "Owner", "H"
	}
}

func previewHomeURL(role string) string {
	page := "dashboard"
	switch role {
	case model.RoleCashier:
		page = "kasir"
	case model.RoleWarehouse:
		page = "penerimaan"
	}
	return "/__preview/" + page + "?role=" + url.QueryEscape(role)
}
