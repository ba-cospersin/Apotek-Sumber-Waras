package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/config"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/database"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/router"
)

func main() {
	// 1. Membaca konfigurasi.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Menghubungkan aplikasi ke MySQL.
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Koneksi database berhasil.")

	// 3. Menyiapkan router dan seluruh route aplikasi.
	mux, err := router.New(db)
	if err != nil {
		log.Fatalf("Gagal menyiapkan router: %v", err)
	}

	// 4. Menjalankan HTTP server.
	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Server berjalan di http://localhost:%s\n", cfg.AppPort)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Printf("Server berhenti karena error: %v", err)
	}
}
