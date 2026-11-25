package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHeaderHasProfileMenuAndLogoutForm(t *testing.T) {
	mux := newAuthenticatedMux(t)
	req := authenticatedRequest(http.MethodGet, "/dashboard")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /dashboard status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	for _, want := range []string{
		`id="user-menu-toggle"`,
		`aria-controls="user-menu-panel"`,
		`aria-expanded="false"`,
		`href="/profil"`,
		`action="/logout"`,
		`method="post"`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("header tidak memuat %q", want)
		}
	}
}
