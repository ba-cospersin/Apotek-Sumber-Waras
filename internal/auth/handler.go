package auth

import (
	"errors"
	"log"
	"net/http"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/middleware"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/view"
)

var _ middleware.SessionValidator = (*Service)(nil)

type LoginRenderer interface {
	RenderLogin(w http.ResponseWriter, status int, data view.LoginData)
}

type Handler struct {
	Svc  *Service
	Auth *middleware.Auth
	View LoginRenderer
}

func NewHandler(svc *Service, mw *middleware.Auth, v LoginRenderer) *Handler {
	return &Handler{Svc: svc, Auth: mw, View: v}
}

func (h *Handler) FormLogin(w http.ResponseWriter, r *http.Request) {
	h.View.RenderLogin(w, http.StatusOK, view.LoginData{})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	identifier := r.PostFormValue("identifier")

	token, user, err := h.Svc.Login(r.Context(), identifier, r.PostFormValue("password"))
	if errors.Is(err, ErrLoginGagal) {
		h.View.RenderLogin(w, http.StatusUnauthorized, view.LoginData{
			Identifier: identifier,
			Error:      ErrLoginGagal.Error(),
		})
		return
	}
	if err != nil {
		log.Printf("login: %v", err)
		http.Error(w, "Terjadi kesalahan pada server", http.StatusInternalServerError)
		return
	}

	h.Auth.SetSessionCookie(w, token)
	http.Redirect(w, r, halamanAwal(user.Role), http.StatusSeeOther)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(middleware.SessionCookieName); err == nil {
		if err := h.Svc.Logout(r.Context(), c.Value); err != nil {
			log.Printf("logout: %v", err)
		}
	}
	h.Auth.ClearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func halamanAwal(role string) string {
	switch role {
	case model.RoleOwner:
		return "/dashboard"
	case model.RoleCashier:
		return "/kasir"
	case model.RoleWarehouse:
		return "/cek-stok"
	default:
		return "/login"
	}
}
