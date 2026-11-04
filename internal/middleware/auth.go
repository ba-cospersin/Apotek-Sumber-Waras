package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

const SessionCookieName = "session"

// SessionValidator merupakan kontrak untuk memeriksa session.
// Implementasinya akan dibuat oleh Backend Owner
// melalui repository autentikasi.
type SessionValidator interface {
	ValidateAndTouch(
		ctx context.Context,
		token string,
	) (model.User, error)
}

type userContextKey struct{}

// Auth mengelola pemeriksaan session pengguna.
type Auth struct {
	Sessions     SessionValidator
	CookieSecure bool
}

func NewAuth(
	sessions SessionValidator,
	cookieSecure bool,
) *Auth {
	return &Auth{
		Sessions:     sessions,
		CookieSecure: cookieSecure,
	}
}

// SetSessionCookie digunakan setelah login berhasil.
func (a *Auth) SetSessionCookie(
	w http.ResponseWriter,
	token string,
) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   8 * 60 * 60,
		HttpOnly: true,
		Secure:   a.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie digunakan saat logout
// atau ketika session sudah tidak valid.
func (a *Auth) ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

// RequireLogin memeriksa session sebelum
// request diteruskan ke handler.
func (a *Auth) RequireLogin(
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		if a.Sessions == nil {
			http.Error(
				w,
				"Authentication belum dikonfigurasi",
				http.StatusInternalServerError,
			)
			return
		}

		cookie, err := r.Cookie(SessionCookieName)
		if err != nil || strings.TrimSpace(cookie.Value) == "" {
			a.ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		user, err := a.Sessions.ValidateAndTouch(
			r.Context(),
			cookie.Value,
		)

		if errors.Is(err, model.ErrInvalidSession) {
			a.ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		if err != nil {
			log.Printf("gagal memeriksa session: %v", err)
			http.Error(
				w,
				"Terjadi kesalahan pada server",
				http.StatusInternalServerError,
			)
			return
		}

		if !user.IsActive {
			a.ClearSessionCookie(w)
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			userContextKey{},
			user,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CurrentUser mengambil data pengguna
// dari context request.
func CurrentUser(r *http.Request) (model.User, bool) {
	user, ok := r.Context().Value(userContextKey{}).(model.User)
	return user, ok
}
