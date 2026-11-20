package main

import (
	"context"
	"database/sql"
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

	// Pastikan seluruh migration yang dibutuhkan seed sudah diterapkan.
	requiredMigrations := []string{
		"001_init",
		"002_add_email_to_users",
	}

	for _, version := range requiredMigrations {
		var migrationApplied int
		if err := db.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM schema_migrations WHERE version = ?",
			version,
		).Scan(&migrationApplied); err != nil {
			return fmt.Errorf("gagal memeriksa migration %s; jalankan 'go run ./cmd/migrate' terlebih dahulu: %w", version, err)
		}
		if migrationApplied == 0 {
			return fmt.Errorf("migration %s belum diterapkan; jalankan 'go run ./cmd/migrate' terlebih dahulu", version)
		}
	}

	// Tabel teknis pencatatan seed; bukan bagian dari 14 tabel inti ERD.
	const createSeedTable = `
    CREATE TABLE IF NOT EXISTS seed_runs (
        seed_name VARCHAR(100) NOT NULL PRIMARY KEY,
        seeded_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci`
	if _, err := db.ExecContext(ctx, createSeedTable); err != nil {
		return fmt.Errorf("gagal membuat tabel pencatatan seed: %w", err)
	}

	// Deteksi seed lama yang sebelumnya berupa satu file seed.sql.
	// Jangan menjalankan seed terpisah di atas data lama karena ID dan
	// username dummy-nya sama dan dapat menyebabkan duplikasi.
	var legacyApplied int
	if err := db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM seed_runs WHERE seed_name = ?",
		"001_demo_data",
	).Scan(&legacyApplied); err != nil {
		return fmt.Errorf("gagal memeriksa seed lama: %w", err)
	}
	if legacyApplied > 0 {
		return fmt.Errorf("database ini sudah menggunakan seed lama '001_demo_data'. Data dummy lama tetap dapat digunakan; untuk beralih ke seed terpisah, reset hanya database pengembangan lokal lalu jalankan migrate dan seed kembali")
	}

	files, err := filepath.Glob(filepath.Join("seed", "*.sql"))
	if err != nil {
		return fmt.Errorf("gagal mencari file seed: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("tidak ditemukan file .sql di folder seed/")
	}

	for _, file := range files {
		name := filepath.Base(file)

		var alreadyApplied int
		if err := db.QueryRowContext(
			ctx,
			"SELECT COUNT(*) FROM seed_runs WHERE seed_name = ?",
			name,
		).Scan(&alreadyApplied); err != nil {
			return fmt.Errorf("gagal memeriksa status %s: %w", name, err)
		}
		if alreadyApplied > 0 {
			fmt.Printf("[SKIP] %s sudah pernah dijalankan.\n", name)
			continue
		}

		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("gagal membaca %s: %w", file, err)
		}

		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("gagal memulai transaksi %s: %w", name, err)
		}

		if err := executeStatements(ctx, txExecutor{tx}, string(sqlBytes)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("seed %s gagal: %w", name, err)
		}

		if _, err := tx.ExecContext(
			ctx,
			"INSERT INTO seed_runs (seed_name) VALUES (?)",
			name,
		); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("gagal mencatat seed %s: %w", name, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("gagal menyimpan seed %s: %w", name, err)
		}

		fmt.Printf("[OK] %s berhasil diterapkan.\n", name)
	}

	fmt.Println("Semua file seed selesai diproses. Data dummy siap digunakan.")
	return nil
}

type txExecutor struct {
	tx *sql.Tx
}

func (e txExecutor) ExecContext(ctx context.Context, query string, args ...any) error {
	_, err := e.tx.ExecContext(ctx, query, args...)
	return err
}

func executeStatements(ctx context.Context, exec txExecutor, content string) error {
	// Semua file seed hanya berisi INSERT sederhana, setiap statement
	// dipisahkan titik koma. Hindari titik koma di dalam string literal.
	statements := strings.Split(content, ";")
	for i, statement := range statements {
		statement = stripLeadingComments(statement)
		if statement == "" {
			continue
		}
		if err := exec.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("statement ke-%d: %w", i+1, err)
		}
	}
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
