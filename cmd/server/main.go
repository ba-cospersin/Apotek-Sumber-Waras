package main

import (
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/app"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/config"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/database"
)

func main() {
	// 1. Membaca konfigurasi aplikasi.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// 2. Membuat satu connection pool MySQL untuk seluruh aplikasi.
	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Println("Koneksi database berhasil.")

	// 3. Menyerahkan inisialisasi renderer dan router ke internal/app.
	application, err := app.New(db)
	if err != nil {
		log.Fatalf("Gagal menyiapkan aplikasi: %v", err)
	}

	// 4. Menjalankan HTTP server.
	server := &http.Server{
		Addr:              ":" + cfg.AppPort,
		Handler:           application.Handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("Server berjalan di http://localhost:%s\n", cfg.AppPort)

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Printf("Server berhenti karena error: %v", err)
	}
}
