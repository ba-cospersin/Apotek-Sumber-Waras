package view

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

// Renderer memuat dan merender template HTML aplikasi.
type Renderer struct {
	login *template.Template
	pages map[string]*template.Template
}

// LoginData adalah data yang ditampilkan pada halaman login.
// Kontrak ini dipertahankan agar internal/auth tidak perlu diubah.
type LoginData struct {
	Error      string
	Identifier string
}

// PageData adalah data bersama yang dikirim ke layout, partial, dan halaman.
type PageData struct {
	Title       string
	Subtitle    string
	PageCSS     string
	PageJS      string
	ActivePage  string
	CurrentDate string
	RoleLabel   string
	AvatarText  string
	HomeURL     string
	User        model.User
	Routes      map[string]string
}

// DashboardData dipertahankan untuk kompatibilitas dengan pemanggil lama.
type DashboardData struct {
	Title        string
	PageCSS      string
	ActivePage   string
	DashboardURL string
	CurrentDate  string
	User         model.User
}

var pageFiles = map[string]string{
	"dashboard":          "web/templates/dashboard.html",
	"kasir":              "web/templates/kasir.html",
	"penerimaan":         "web/templates/penerimaan.html",
	"cek_stok":           "web/templates/cek_stok.html",
	"retur":              "web/templates/retur.html",
	"stock_opname":       "web/templates/stock_opname.html",
	"kedaluwarsa":        "web/templates/kedaluwarsa.html",
	"laporan":            "web/templates/laporan.html",
	"data_master":        "web/templates/data_master.html",
	"manajemen_pengguna": "web/templates/manajemen_pengguna.html",
	"struk_penjualan":    "web/templates/cashier/struk_penjualan.html",
}

// NewRenderer memuat setiap halaman bersama layout dan partial yang dipakai ulang.
// Jalankan server dari root repository agar path web/templates ditemukan.
func NewRenderer() (*Renderer, error) {
	loginTemplate, err := template.ParseFiles("web/templates/auth/login.html")
	if err != nil {
		return nil, fmt.Errorf("gagal memuat template login: %w", err)
	}

	commonFiles := []string{
		"web/templates/layouts/base.html",
		"web/templates/partials/header.html",
		"web/templates/partials/sidebar.html",
	}

	pages := make(map[string]*template.Template, len(pageFiles))
	for name, pageFile := range pageFiles {
		files := append(append([]string{}, commonFiles...), pageFile)
		pageTemplate, parseErr := template.ParseFiles(files...)
		if parseErr != nil {
			return nil, fmt.Errorf("gagal memuat template halaman %q: %w", name, parseErr)
		}
		pages[name] = pageTemplate
	}

	return &Renderer{login: loginTemplate, pages: pages}, nil
}

// RenderLogin merender halaman login dengan status HTTP yang ditentukan.
func (rd *Renderer) RenderLogin(w http.ResponseWriter, status int, data LoginData) {
	var output bytes.Buffer
	if err := rd.login.ExecuteTemplate(&output, "login.html", data); err != nil {
		http.Error(w, "Terjadi kesalahan saat menampilkan halaman", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = output.WriteTo(w)
}

// RenderPage merender salah satu halaman dengan layout utama yang sama.
func (rd *Renderer) RenderPage(w http.ResponseWriter, status int, pageName string, data PageData) {
	pageTemplate, ok := rd.pages[pageName]
	if !ok {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	var output bytes.Buffer
	if err := pageTemplate.ExecuteTemplate(&output, "base", data); err != nil {
		http.Error(w, "Terjadi kesalahan saat menampilkan halaman", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = output.WriteTo(w)
}

// RenderDashboard dipertahankan sebagai wrapper agar integrasi lama tidak langsung rusak.
func (rd *Renderer) RenderDashboard(w http.ResponseWriter, status int, data DashboardData) {
	routes := map[string]string{
		"dashboard":          data.DashboardURL,
		"kasir":              "/__preview/kasir?role=OWNER",
		"penerimaan":         "/__preview/penerimaan?role=OWNER",
		"cek_stok":           "/__preview/cek_stok?role=OWNER",
		"retur":              "/__preview/retur?role=OWNER",
		"stock_opname":       "/__preview/stock_opname?role=OWNER",
		"kedaluwarsa":        "/__preview/kedaluwarsa?role=OWNER",
		"laporan":            "/__preview/laporan?role=OWNER",
		"data_master":        "/__preview/data_master?role=OWNER",
		"manajemen_pengguna": "/__preview/manajemen_pengguna?role=OWNER",
		"struk_penjualan":    "/__preview/struk_penjualan?role=OWNER",
	}
	homeURL := data.DashboardURL
	if homeURL == "" {
		homeURL = routes["dashboard"]
	}
	rd.RenderPage(w, status, "dashboard", PageData{
		Title:       data.Title,
		PageCSS:     data.PageCSS,
		ActivePage:  data.ActivePage,
		CurrentDate: data.CurrentDate,
		RoleLabel:   "Owner",
		AvatarText:  "H",
		HomeURL:     homeURL,
		User:        data.User,
		Routes:      routes,
	})
}
