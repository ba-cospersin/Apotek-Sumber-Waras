package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/sessions"
)

const (
	sessionName   = "apotek_sumber_waras"
	sessionMaxAge = 8 * 60 * 60 // 8 jam
	loginPath     = "/login"
)

// AuthUser menyimpan identitas pengguna yang sudah login.
type AuthUser struct {
	ID   int64
	Role string
}

// Key khusus untuk menyimpan user pada request context.
type authUserContextKey struct{}

// Auth mengelola session dan autentikasi pengguna.
type Auth struct {
	store *sessions.CookieStore
}

// NewAuth membuat session store menggunakan secret key.
func NewAuth(secret string) (*Auth, error) {
	secret = strings.TrimSpace(secret)

	if len([]byte(secret)) < 32 {
		return nil, fmt.Errorf(
			"SESSION_KEY harus memiliki minimal 32 byte",
		)
	}

	store := sessions.NewCookieStore([]byte(secret))

	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   sessionMaxAge,
		HttpOnly: true,
		Secure:   false, // Untuk localhost. Aktifkan true saat menggunakan HTTPS.
		SameSite: http.SameSiteLaxMode,
	}

	return &Auth{
		store: store,
	}, nil
}

// SetUserSession dipanggil setelah username dan password berhasil diverifikasi.
func (a *Auth) SetUserSession(
	w http.ResponseWriter,
	r *http.Request,
	userID int64,
	role string,
) error {
	if userID <= 0 {
		return fmt.Errorf("ID pengguna tidak valid")
	}

	role = normalizeRole(role)

	if !isValidRole(role) {
		return fmt.Errorf("role pengguna tidak valid")
	}

	session, err := a.store.Get(r, sessionName)
	if err != nil {
		return fmt.Errorf("gagal mengambil session: %w", err)
	}

	session.Values["user_id"] = strconv.FormatInt(userID, 10)
	session.Values["role"] = role

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("gagal menyimpan session: %w", err)
	}

	return nil
}

// ClearUserSession menghapus session ketika pengguna logout.
func (a *Auth) ClearUserSession(
	w http.ResponseWriter,
	r *http.Request,
) error {
	session, err := a.store.Get(r, sessionName)
	if err != nil {
		return fmt.Errorf("gagal mengambil session: %w", err)
	}

	session.Values = make(map[interface{}]interface{})

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
	}

	if err := session.Save(r, w); err != nil {
		return fmt.Errorf("gagal menghapus session: %w", err)
	}

	return nil
}

// RequireLogin memastikan pengguna sudah memiliki session yang valid.
func (a *Auth) RequireLogin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		session, err := a.store.Get(r, sessionName)
		if err != nil {
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}

		idValue, idOK := session.Values["user_id"].(string)
		roleValue, roleOK := session.Values["role"].(string)

		if !idOK || !roleOK {
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}

		userID, err := strconv.ParseInt(idValue, 10, 64)
		role := normalizeRole(roleValue)

		if err != nil || userID <= 0 || !isValidRole(role) {
			http.Redirect(w, r, loginPath, http.StatusSeeOther)
			return
		}

		user := AuthUser{
			ID:   userID,
			Role: role,
		}

		ctx := context.WithValue(
			r.Context(),
			authUserContextKey{},
			user,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentUser mengambil identitas pengguna dari request context.
func CurrentUser(r *http.Request) (AuthUser, bool) {
	user, ok := r.Context().Value(authUserContextKey{}).(AuthUser)
	return user, ok
}
