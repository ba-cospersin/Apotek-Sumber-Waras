package database

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"time"

	"github.com/go-sql-driver/mysql"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/config"
)

func Connect(cfg config.Config) (*sql.DB, error) {
	mysqlCfg := mysql.Config{
		User:      cfg.DBUser,
		Passwd:    cfg.DBPassword,
		Net:       "tcp",
		Addr:      net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		DBName:    cfg.DBName,
		ParseTime: true,
		Loc:       time.Local,

		// Mengizinkan autentikasi mysql_native_password.
		AllowNativePasswords: true,

		Params: map[string]string{
			"charset": "utf8mb4",
		},
	}

	// Membuka connection pool MySQL.
	db, err := sql.Open("mysql", mysqlCfg.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("gagal membuka koneksi database: %w", err)
	}

	// Pengaturan connection pool.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	// Memastikan MySQL benar-benar dapat dihubungi.
	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("gagal terhubung ke MySQL: %w", err)
	}

	return db, nil
}
