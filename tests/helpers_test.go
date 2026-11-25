package tests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/auth"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/middleware"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/router"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/view"
)

type stubSessionValidator struct {
	user model.User
}

func (s stubSessionValidator) ValidateAndTouch(ctx context.Context, token string) (model.User, error) {
	return s.user, nil
}

func newMuxForUser(t *testing.T, user model.User) http.Handler {
	t.Helper()

	renderer, err := view.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}

	authMw := middleware.NewAuth(stubSessionValidator{user: user}, false)
	authHandler := auth.NewHandler(nil, authMw, renderer)

	return router.New(nil, renderer, authHandler, authMw)
}

func newAuthenticatedMux(t *testing.T) http.Handler {
	t.Helper()

	return newMuxForUser(t, model.User{
		ID:       1,
		Nama:     "Owner Demo",
		Username: "owner",
		Email:    "owner@example.test",
		Role:     model.RoleOwner,
		IsActive: true,
	})
}

func authenticatedRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, path, nil)
	req.AddCookie(&http.Cookie{
		Name:  middleware.SessionCookieName,
		Value: "token-uji",
	})
	return req
}
