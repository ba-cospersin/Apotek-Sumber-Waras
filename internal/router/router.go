package router

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/auth"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/view"
)

type pageConfig struct {
	name        string
	title       string
	subtitle    string
	css         string
	js          string
	active      string
	defaultRole string
}

// pagePaths adalah kontrak URL halaman aplikasi. URL tidak membawa role;
// hak akses nantinya ditentukan backend berdasarkan session login.
var pagePaths = map[string]string{
	"dashboard":          "/dashboard",
	"kasir":              "/kasir",
	"penerimaan":         "/penerimaan",
	"cek_stok":           "/cek-stok",
	"retur":              "/retur",
	"stock_opname":       "/stock-opname",
	"kedaluwarsa":        "/kedaluwarsa",
	"laporan":            "/laporan",
	"data_master":        "/data-master",
	"manajemen_pengguna": "/manajemen-pengguna",
	"struk_penjualan":    "/struk-penjualan",
}

var pageConfigs = []pageConfig{
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

// New mendaftarkan route aplikasi menggunakan URL permanen.
// db boleh nil hanya untuk pengujian halaman; endpoint /health akan mengembalikan 503.
func New(db *sql.DB, renderer *view.Renderer) *http.ServeMux {
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

	// Handler login tetap dimiliki modul Backend Owner.
	// Saat ini GET login tersedia; POST akan aktif setelah service autentikasi selesai.
	authHandler := auth.NewHandler(renderer)
	mux.HandleFunc("/login", authHandler.Login)

	// PERHATIAN: halaman di bawah masih menggunakan identitas dan data demo.
	// Route sudah memakai URL final agar frontend dapat diuji tanpa __preview.
	// Route ini BELUM dilindungi session/role dan tidak boleh dianggap siap produksi.
	for _, config := range pageConfigs {
		pageConfig := config
		path := pagePaths[pageConfig.name]
		if path == "" {
			continue
		}

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

			user, roleLabel, avatar := previewUser(pageConfig.defaultRole)
			routes := make(map[string]string, len(pagePaths))
			for name, routePath := range pagePaths {
				routes[name] = routePath
			}

			renderer.RenderPage(w, http.StatusOK, pageConfig.name, view.PageData{
				Title:       pageConfig.title,
				Subtitle:    pageConfig.subtitle,
				PageCSS:     pageConfig.css,
				PageJS:      pageConfig.js,
				ActivePage:  pageConfig.active,
				CurrentDate: currentDateIndonesia(),
				RoleLabel:   roleLabel,
				AvatarText:  avatar,
				HomeURL:     previewHomeURL(pageConfig.defaultRole),
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
		if db == nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
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

	return mux
}

// previewUser hanya menyediakan identitas statis untuk pemeriksaan tampilan.
// Fungsi ini akan digantikan oleh user dari middleware session saat auth diintegrasikan.
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
	return pagePaths[page]
}

// currentDateIndonesia menampilkan tanggal mengikuti zona waktu operasional Indonesia barat.
func currentDateIndonesia() string {
	location, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		location = time.Local
	}

	now := time.Now().In(location)
	weekdays := [...]string{"Minggu", "Senin", "Selasa", "Rabu", "Kamis", "Jumat", "Sabtu"}
	months := [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

	return fmt.Sprintf("%s, %d %s %d", weekdays[now.Weekday()], now.Day(), months[int(now.Month())-1], now.Year())
}
