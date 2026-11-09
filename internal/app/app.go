package app

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/router"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/view"
)

// Application merupakan hasil inisialisasi seluruh komponen utama aplikasi.
// Koneksi database dibuat satu kali di cmd/server, lalu digunakan bersama
// oleh router dan modul yang nantinya dihubungkan.
type Application struct {
	Handler http.Handler
}

// New menyiapkan renderer dan router aplikasi.
// Dependency bisnis (repository, service, dan handler fitur) dapat ditambahkan
// di sini ketika masing-masing modul sudah selesai diimplementasikan.
func New(db *sql.DB) (*Application, error) {
	if db == nil {
		return nil, errors.New("koneksi database tidak boleh nil")
	}

	renderer, err := view.NewRenderer()
	if err != nil {
		return nil, err
	}

	mux := router.New(db, renderer)

	return &Application{
		Handler: mux,
	}, nil
}
