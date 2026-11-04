package model

import "errors"

const (
	RoleOwner     = "OWNER"
	RoleCashier   = "KASIR"
	RoleWarehouse = "GUDANG"
)

// ErrInvalidSession digunakan ketika session
// tidak ditemukan atau sudah kedaluwarsa.
var ErrInvalidSession = errors.New("session tidak valid")

// User merupakan struktur data pengguna
// yang aman digunakan oleh lapisan aplikasi.
type User struct {
	ID       int64
	Nama     string
	Username string
	Email    string
	Role     string
	IsActive bool
}
