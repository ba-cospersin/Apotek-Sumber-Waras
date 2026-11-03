package middleware

import (
	"net/http"
	"strings"
)

const (
	RoleOwner     = "OWNER"
	RoleCashier   = "KASIR"
	RoleWarehouse = "GUDANG"
)

// normalizeRole menyeragamkan penulisan role.
func normalizeRole(role string) string {
	return strings.ToUpper(strings.TrimSpace(role))
}

// isValidRole memeriksa apakah role terdaftar di sistem.
func isValidRole(role string) bool {
	switch normalizeRole(role) {
	case RoleOwner, RoleCashier, RoleWarehouse:
		return true
	default:
		return false
	}
}

// RequireRole membatasi akses berdasarkan satu atau beberapa role.
func RequireRole(allowedRoles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool)

	for _, role := range allowedRoles {
		role = normalizeRole(role)

		if isValidRole(role) {
			allowed[role] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(
			w http.ResponseWriter,
			r *http.Request,
		) {
			user, ok := CurrentUser(r)
			if !ok {
				http.Error(
					w,
					"Unauthorized",
					http.StatusUnauthorized,
				)
				return
			}

			if !allowed[user.Role] {
				http.Error(
					w,
					"Forbidden: akses ditolak",
					http.StatusForbidden,
				)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
