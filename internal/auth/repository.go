package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

const (
	idleTimeoutMinutes = 30
	maxSessionHours    = 8
)

type Repo struct {
	DB *sql.DB
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (r *Repo) FindByIdentifier(ctx context.Context, identifier string) (model.User, string, error) {
	column := "username"
	if strings.Contains(identifier, "@") {
		column = "email"
	}
	query := fmt.Sprintf(`
		SELECT id, nama, username, email, role, is_active, password_hash
		FROM users WHERE %s = ?	`, column)

	var u model.User
	var hash string
	err := r.DB.QueryRowContext(ctx, query, identifier).
		Scan(&u.ID, &u.Nama, &u.Username, &u.Email, &u.Role, &u.IsActive, &hash)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, "", ErrUserNotFound
	}
	if err != nil {
		return model.User{}, "", fmt.Errorf("cari user : %w", err)
	}
	return u, hash, nil
}

func (r *Repo) CreateSession(ctx context.Context, userID int64, token string) error {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("Mulai Session: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
	INSERT INTO sessions (user_id, token_hash, expires_at)
	VALUES (?, ?, NOW() + INTERVAL ? HOUR)`,
		userID, hashToken(token), maxSessionHours); err != nil {
		return fmt.Errorf("commit session: %w", err)
	}
	return nil
}

func (r *Repo) ValidateAndTouch(ctx context.Context, token string) (model.User, error) {
	var u model.User
	err := r.DB.QueryRowContext(ctx, `
	SELECT u.id, u.nama, u.username, u.email, u.role, u.is_active
	FROM sessions s
	JOIN users u ON u.id = s.user_id
	WHERE s.token_hash = ?
		AND s.expires_at > NOW()
		AND s.last_activity_at > NOW() - INTERVAL ? MINUTE`,
		hashToken(token), idleTimeoutMinutes,
	).Scan(&u.ID, &u.Nama, &u.Username, &u.Email, &u.Role, &u.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		return model.User{}, model.ErrInvalidSession
	}
	if err != nil {
		return model.User{}, fmt.Errorf("cek session: %w", err)
	}

	if _, err := r.DB.ExecContext(ctx,
		"UPDATE sessions SET last_activity_at = NOW() WHERE token_hash = ? ", hashToken(token)); err != nil {
		return model.User{}, fmt.Errorf("perbarui session: %w", err)
	}
	return u, nil
}

func (r *Repo) DeleteSession(ctx context.Context, token string) error {
	if _, err := r.DB.ExecContext(ctx,
		"DELETE FROM sessions WHERE token_hash = ?", hashToken(token)); err != nil {
		return fmt.Errorf("hapus session: %w", err)
	}
	return nil
}
