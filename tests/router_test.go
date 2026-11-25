package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

func TestMain(m *testing.M) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		os.Exit(1)
	}

	projectRoot := filepath.Dir(filepath.Dir(sourceFile))
	if err := os.Chdir(projectRoot); err != nil {
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestPageRoutesUseCanonicalPaths(t *testing.T) {
	mux := newAuthenticatedMux(t)
	paths := []string{
		"/dashboard",
		"/kasir",
		"/penerimaan",
		"/cek-stok",
		"/retur",
		"/stock-opname",
		"/kedaluwarsa",
		"/laporan",
		"/data-master",
		"/manajemen-pengguna",
		"/struk-penjualan",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := authenticatedRequest(http.MethodGet, path)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s status = %d, want %d; body: %s", path, rec.Code, http.StatusOK, rec.Body.String())
			}
			if strings.Contains(rec.Body.String(), "__preview") || strings.Contains(rec.Body.String(), "?role=") {
				t.Fatalf("GET %s still renders a preview URL or role query", path)
			}
		})
	}
}

func TestLegacyPreviewPathIsNotRegistered(t *testing.T) {
	mux := newAuthenticatedMux(t)
	req := authenticatedRequest(http.MethodGet, "/__preview/dashboard")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("legacy preview route status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPageRoutesRejectNonGetMethods(t *testing.T) {
	mux := newAuthenticatedMux(t)
	req := authenticatedRequest(http.MethodPost, "/dashboard")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /dashboard status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow header = %q, want %q", got, http.MethodGet)
	}
}

func TestPageRoutesRejectWrongRole(t *testing.T) {
	cases := []struct {
		role string
		path string
	}{
		{model.RoleCashier, "/penerimaan"},
		{model.RoleCashier, "/stock-opname"},
		{model.RoleCashier, "/kedaluwarsa"},
		{model.RoleWarehouse, "/kasir"},
		{model.RoleWarehouse, "/retur"},
		{model.RoleWarehouse, "/laporan"},
	}

	for _, tc := range cases {
		t.Run(tc.role+" "+tc.path, func(t *testing.T) {
			mux := newMuxForUser(t, model.User{
				ID:       2,
				Nama:     "Pengguna Uji",
				Username: "uji",
				Role:     tc.role,
				IsActive: true,
			})

			req := authenticatedRequest(http.MethodGet, tc.path)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Fatalf("GET %s sebagai %s status = %d, want %d", tc.path, tc.role, rec.Code, http.StatusForbidden)
			}
		})
	}
}

func TestPageRoutesAllowCorrectRole(t *testing.T) {
	cases := []struct {
		role string
		path string
	}{
		{model.RoleCashier, "/kasir"},
		{model.RoleCashier, "/retur"},
		{model.RoleWarehouse, "/penerimaan"},
		{model.RoleWarehouse, "/cek-stok"},
		{model.RoleWarehouse, "/stock-opname"},
	}

	for _, tc := range cases {
		t.Run(tc.role+" "+tc.path, func(t *testing.T) {
			mux := newMuxForUser(t, model.User{
				ID:       2,
				Nama:     "Pengguna Uji",
				Username: "uji",
				Role:     tc.role,
				IsActive: true,
			})

			req := authenticatedRequest(http.MethodGet, tc.path)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s sebagai %s status = %d, want %d", tc.path, tc.role, rec.Code, http.StatusOK)
			}
		})
	}
}
