package middleware

import (
	"net/http"
	"strings"

	"github.com/ba-cospersin/Apotek-Sumber-Waras/internal/model"
)

// RequireRole membatasi akses berdasarkan role.
func RequireRole(
	allowedRoles ...string,
) func(http.Handler) http.Handler {

	allowed := make(map[string]bool)

	for _, role := range allowedRoles {
		role = strings.ToUpper(strings.TrimSpace(role))

		switch role {
		case model.RoleOwner,
			model.RoleCashier,
			model.RoleWarehouse:
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
