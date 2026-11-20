package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/config"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/database"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("gagal membaca konfigurasi: %w", err)
	}

	db, err := database.Connect(cfg)
	if err != nil {
		return fmt.Errorf("gagal terhubung ke database: %w", err)
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
		return fmt.Errorf("gagal membuat tabel pencatatan migration: %w", err)
	}

	// Baca seluruh file migration dan urutkan berdasarkan nama file.
	files, err := filepath.Glob(filepath.Join("migrations", "*.sql"))
	if err != nil {
		return fmt.Errorf("gagal mencari file migration: %w", err)
	}
	sort.Strings(files)

	if len(files) == 0 {
		return fmt.Errorf("tidak ditemukan file .sql di folder migrations/")
	}

	for _, file := range files {
		name := filepath.Base(file)
		version := strings.TrimSuffix(name, filepath.Ext(name))

		var alreadyApplied int
		err := db.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
			version,
		).Scan(&alreadyApplied)
		if err != nil {
			return fmt.Errorf("gagal memeriksa status migration %s: %w", version, err)
		}

		if alreadyApplied > 0 {
			fmt.Printf("[SKIP] %s sudah pernah dijalankan.\n", version)
			continue
		}

		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("gagal membaca %s: %w", file, err)
		}

		// File migration hanya berisi perintah SQL biasa yang dipisahkan
		// titik koma. Hindari titik koma di dalam string literal atau stored procedure.
		statements := strings.Split(string(sqlBytes), ";")
		for i, statement := range statements {
			statement = stripLeadingComments(statement)
			if statement == "" {
				continue
			}

			if _, err := db.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf(
					"migration %s gagal pada perintah ke-%d: %w",
					version,
					i+1,
					err,
				)
			}
		}

		if _, err := db.ExecContext(
			ctx,
			"INSERT INTO schema_migrations (version) VALUES (?)",
			version,
		); err != nil {
			return fmt.Errorf("gagal mencatat migration %s: %w", version, err)
		}

		fmt.Printf("[OK] Migration %s berhasil diterapkan.\n", version)
	}

	fmt.Println("Semua migration selesai diproses.")
	return nil
}

func stripLeadingComments(statement string) string {
	statement = strings.TrimSpace(statement)
	for strings.HasPrefix(statement, "--") {
		newline := strings.IndexByte(statement, '\n')
		if newline == -1 {
			return ""
		}
		statement = strings.TrimSpace(statement[newline+1:])
	}
	return statement
}
