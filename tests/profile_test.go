package tests

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

func TestProfilePageShowsLoggedInUser(t *testing.T) {
	user := model.User{
		ID:       7,
		Nama:     "Siti Rahmawati",
		Username: "siti",
		Email:    "siti@example.test",
		Role:     model.RoleWarehouse,
		IsActive: true,
	}
	mux := newMuxForUser(t, user)

	req := authenticatedRequest(http.MethodGet, "/profil")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /profil status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	body := rec.Body.String()
	for _, want := range []string{"Profil Saya", user.Nama, user.Username, user.Email, "Petugas Gudang"} {
		if !strings.Contains(body, want) {
			t.Fatalf("GET /profil body tidak memuat %q", want)
		}
	}
	if strings.Contains(body, "Budi Santoso") {
		t.Fatalf("GET /profil masih memakai data dummy Budi Santoso")
	}
}

func TestProfilePageIsStandaloneWithoutSidebar(t *testing.T) {
	user := model.User{
		ID:       1,
		Nama:     "Owner Demo",
		Username: "owner",
		Email:    "owner@example.test",
		Role:     model.RoleOwner,
		IsActive: true,
	}
	mux := newMuxForUser(t, user)

	req := authenticatedRequest(http.MethodGet, "/profil")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /profil status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if strings.Contains(body, `id="app-sidebar"`) {
		t.Fatalf("halaman profil tidak boleh memakai sidebar")
	}
	if !strings.Contains(body, `class="profile-topbar"`) {
		t.Fatalf("halaman profil harus memakai topbar standalone")
	}
	if !strings.Contains(body, `class="profile-main"`) {
		t.Fatalf("halaman profil harus memakai konten standalone")
	}
}

func TestKasirPageShowsSessionUserWhenLoggedIn(t *testing.T) {
	user := model.User{
		ID:       9,
		Nama:     "Kasir Sesi",
		Username: "kasir_sesi",
		Email:    "kasir_sesi@example.test",
		Role:     model.RoleCashier,
		IsActive: true,
	}
	mux := newMuxForUser(t, user)

	req := authenticatedRequest(http.MethodGet, "/kasir")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /kasir status = %d, want %d", rec.Code, http.StatusOK)
	}

	body := rec.Body.String()
	if !strings.Contains(body, user.Nama) {
		t.Fatalf("GET /kasir tidak menampilkan user session %q", user.Nama)
	}
	if strings.Contains(body, "Budi Santoso") {
		t.Fatalf("GET /kasir masih menampilkan identitas preview")
	}
}

func TestDashboardRedirectsToLoginWithoutSession(t *testing.T) {
	mux := newAuthenticatedMux(t)

	req := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("GET /dashboard tanpa session status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("GET /dashboard tanpa session Location = %q, want %q", loc, "/login")
	}
}

func TestProfilePageRedirectsToLoginWithoutSession(t *testing.T) {
	mux := newAuthenticatedMux(t)

	req := httptest.NewRequest(http.MethodGet, "/profil", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("GET /profil tanpa session status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	if loc := rec.Header().Get("Location"); loc != "/login" {
		t.Fatalf("GET /profil tanpa session Location = %q, want %q", loc, "/login")
	}
}
