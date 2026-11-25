package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

var (
	// ErrLoginGagal dipakai untuk SEMUA kegagalan login (identifier tidak
	// ada, password salah, akun nonaktif) agar tidak membocorkan informasi.
	ErrLoginGagal = errors.New("username/email atau password salah")

	// ErrUserNotFound hanya dipakai antara repository dan service.
	ErrUserNotFound = errors.New("pengguna tidak ditemukan")
)

type Service struct {
	Repo *Repo
}

// Hash palsu: dibandingkan ketika user tidak ditemukan supaya waktu respons
// mirip dengan kasus password salah.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password"), bcrypt.DefaultCost)

// Login memverifikasi kredensial dan membuat session baru.
// Mengembalikan token untuk disimpan di cookie oleh handler.
func (s *Service) Login(ctx context.Context, identifier, password string) (string, model.User, error) {
	identifier = strings.TrimSpace(identifier)
	if strings.Contains(identifier, "@") {
		identifier = strings.ToLower(identifier)
	}
	if identifier == "" || password == "" {
		return "", model.User{}, ErrLoginGagal
	}

	user, hash, err := s.Repo.FindByIdentifier(ctx, identifier)
	if errors.Is(err, ErrUserNotFound) {
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return "", model.User{}, ErrLoginGagal
	}
	if err != nil {
		return "", model.User{}, err // error server, bukan salah login
	}

	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return "", model.User{}, ErrLoginGagal
	}
	if !user.IsActive {
		return "", model.User{}, ErrLoginGagal
	}

	token, err := newToken()
	if err != nil {
		return "", model.User{}, err
	}
	if err := s.Repo.CreateSession(ctx, user.ID, token); err != nil {
		return "", model.User{}, err
	}
	return token, user, nil
}

// ValidateAndTouch membuat Service memenuhi middleware.SessionValidator.
func (s *Service) ValidateAndTouch(ctx context.Context, token string) (model.User, error) {
	return s.Repo.ValidateAndTouch(ctx, token)
}

// Logout menghapus session dari database.
func (s *Service) Logout(ctx context.Context, token string) error {
	return s.Repo.DeleteSession(ctx, token)
}

// newToken membuat 32 byte acak (crypto/rand) dalam hex = 64 karakter.
func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("buat token: %w", err)
	}
	return hex.EncodeToString(b), nil
}
