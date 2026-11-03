package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/config"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("gagal membaca konfigurasi: %v", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("gagal terhubung ke database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Tabel teknis untuk mencatat migration yang sudah pernah dijalankan.
	// Tabel ini bukan bagian dari 14 tabel inti pada ERD.
	const createMigrationTable = `
	CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(100) NOT NULL PRIMARY KEY,
		applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`

	if _, err := db.ExecContext(ctx, createMigrationTable); err != nil {
		log.Fatalf("gagal membuat tabel pencatatan migration: %v", err)
	}

	const version = "001_init"

	var alreadyApplied int
	err = db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
		version,
	).Scan(&alreadyApplied)
	if err != nil {
		log.Fatalf("gagal memeriksa status migration: %v", err)
	}

	if alreadyApplied > 0 {
		log.Printf("Migration %s sudah pernah dijalankan; dilewati.", version)
		return
	}

	sqlBytes, err := os.ReadFile("migrations/001_init.sql")
	if err != nil {
		log.Fatalf("gagal membaca migrations/001_init.sql: %v", err)
	}

	// File migration ini berisi perintah SQL biasa yang dipisahkan titik koma.
	// Jangan gunakan titik koma di dalam nilai string atau stored procedure.
	statements := strings.Split(string(sqlBytes), ";")
	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		if _, err := db.ExecContext(ctx, statement); err != nil {
			log.Fatalf("migration %s gagal pada perintah ke-%d: %v", version, i+1, err)
		}
	}

	if _, err := db.ExecContext(
		ctx,
		"INSERT INTO schema_migrations (version) VALUES (?)",
		version,
	); err != nil {
		log.Fatalf("gagal mencatat migration %s: %v", version, err)
	}

	fmt.Printf("Migration %s berhasil diterapkan.\n", version)
}
