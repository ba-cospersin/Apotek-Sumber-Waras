package router

import (
	"context"
	"database/sql"
	"html/template"
	"net/http"
	"time"
)

func New(db *sql.DB) *http.ServeMux {
	mux := http.NewServeMux()

	// 1. Mengizinkan file CSS, JS, dan Gambar diakses oleh HTML
	// Pastikan folder Anda bernama "web/static"
	fileServer := http.FileServer(http.Dir("web/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	// 2. Rute Health Check (Bawaan Anda)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			http.Error(w, "Database unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Server and database are running"))
	})

	// 3. Rute Halaman Login
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		// Halaman login berdiri sendiri, tidak butuh base, sidebar, atau header.
		// Sesuaikan "dashboardlogin.html" dengan nama file login asli Anda.
		tmpl, err := template.ParseFiles("web/templates/dashboardlogin.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	})

	// 4. Rute Halaman Dashboard Utama
	mux.HandleFunc("/dashboard", func(w http.ResponseWriter, r *http.Request) {
		// Untuk dashboard, WAJIB memuat keempat file ini bersamaan
		// karena base.html membutuhkan sidebar, header, dan content.
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/dashboard.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Mengeksekusi template dengan base.html sebagai kerangka utamanya
		err = tmpl.ExecuteTemplate(w, "base.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/datamaster/dataobat", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/header.html",
			"web/templates/partials/sidebar.html",
			"web/templates/datamaster/dataobat.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "base.html", nil)
	})

	mux.HandleFunc("/datamaster/datasatuan", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/header.html",
			"web/templates/partials/sidebar.html",
			"web/templates/datamaster/datasatuan.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "base.html", nil)
	})

	// Rute Halaman Data Master -> Konversi Satuan
	mux.HandleFunc("/datamaster/konversisatuan", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/datamaster/konversisatuan.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "base.html", nil)
	})

	mux.HandleFunc("/datamaster/hargajual", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/datamaster/hargajual.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "base.html", nil)
	})

	mux.HandleFunc("/datamaster/datapbf", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/datamaster/datapbf.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.ExecuteTemplate(w, "base.html", nil)
	})

	mux.HandleFunc("/penerimaan/penerimaan", func(w http.ResponseWriter, r *http.Request) {
		// Untuk dashboard, WAJIB memuat keempat file ini bersamaan
		// karena base.html membutuhkan sidebar, header, dan content.
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/penerimaan/penerimaan.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Mengeksekusi template dengan base.html sebagai kerangka utamanya
		err = tmpl.ExecuteTemplate(w, "base.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/pemusnahan/pemusnahan", func(w http.ResponseWriter, r *http.Request) {
		// Untuk dashboard, WAJIB memuat keempat file ini bersamaan
		// karena base.html membutuhkan sidebar, header, dan content.
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/pemusnahan/pemusnahan.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Mengeksekusi template dengan base.html sebagai kerangka utamanya
		err = tmpl.ExecuteTemplate(w, "base.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/pengembalian/pengembalian", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/pengembalian/pengembalian.html",
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Mengeksekusi template dengan base.html sebagai kerangka utamanya
		err = tmpl.ExecuteTemplate(w, "base.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

	})

	mux.HandleFunc("/pengembalian/formpengembalian", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/pengembalian/formpengembalian.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/pengembalian/detailpengembalian", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/pengembalian/detailpengembalian.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/persediaan/stock", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/persediaan/stock.html",
		)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Mengeksekusi template dengan base.html sebagai kerangka utamanya
		err = tmpl.ExecuteTemplate(w, "base.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Rute Halaman Lacak Batch (Penelusuran CDOB BPOM)
	mux.HandleFunc("/persediaan/lacak", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/persediaan/lacak.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Rute Halaman Riwayat Penyesuaian & Log Mutasi Stok
	mux.HandleFunc("/persediaan/riwayat", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/persediaan/riwayat.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/persediaan/audit", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/persediaan/audit.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/persediaan/kedaluwarsa", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/persediaan/kedaluwarsa.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/datamaster/tambahobat", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/partials/header.html",
			"web/templates/datamaster/tambahobat.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/datamaster/tambahkonversi", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/datamaster/tambahkonversi.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/datamaster/tambahhargajual", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/datamaster/tambahhargajual.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/datamaster/hargajual/store", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Proses data form di sini (e.g. r.ParseForm())
			// Setelah simpan ke database, redirect kembali ke halaman utama Harga Jual
			http.Redirect(w, r, "/datamaster/hargajual", http.StatusSeeOther)
			return
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})

	// Rute Halaman Form Tambah Data PBF
	mux.HandleFunc("/datamaster/tambahpbf", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/datamaster/tambahpbf.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// Handler Simpan Data PBF (POST)
	mux.HandleFunc("/datamaster/pbf/store", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			// Redirect kembali ke halaman utama Data PBF setelah berhasil simpan
			http.Redirect(w, r, "/datamaster/datapbf", http.StatusSeeOther)
			return
		}
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	})

	mux.HandleFunc("/pemusnahan/formpemusnahan", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/pemusnahan/formpemusnahan.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/pemusnahan/detail", func(w http.ResponseWriter, r *http.Request) {
		tmpl, err := template.ParseFiles(
			"web/templates/layouts/base.html",
			"web/templates/partials/sidebar.html",
			"web/templates/pemusnahan/detailpemusnahan.html",
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmpl.ExecuteTemplate(w, "base.html", nil); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
	return mux
}
