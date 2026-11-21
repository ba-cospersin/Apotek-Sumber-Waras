package tests

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/router"
	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/view"
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
	renderer, err := view.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}

	mux := router.New(nil, renderer)
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
			req := httptest.NewRequest(http.MethodGet, path, nil)
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
	renderer, err := view.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}

	mux := router.New(nil, renderer)
	req := httptest.NewRequest(http.MethodGet, "/__preview/dashboard", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("legacy preview route status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestPageRoutesRejectNonGetMethods(t *testing.T) {
	renderer, err := view.NewRenderer()
	if err != nil {
		t.Fatalf("NewRenderer() error = %v", err)
	}

	mux := router.New(nil, renderer)
	req := httptest.NewRequest(http.MethodPost, "/dashboard", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /dashboard status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
	if got := rec.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("Allow header = %q, want %q", got, http.MethodGet)
	}
}
