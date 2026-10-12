package auth

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"

	"tradielynx/internal/models"
)

type SessionReader interface {
	GetInt(ctx context.Context, key string) int
	GetInt64(ctx context.Context, key string) int64
	GetString(ctx context.Context, key string) string
	GetBool(ctx context.Context, key string) bool
	Put(ctx context.Context, key string, val any)
	Exists(ctx context.Context, key string) bool
	Remove(ctx context.Context, key string)
	Destroy(ctx context.Context) error
}

var sess struct{ Manager SessionReader }

func UseSessionManager(m SessionReader) { sess.Manager = m }

type AuthLookup interface {
	LookupUserAuth(ctx context.Context, userID int64) (models.Role, bool, error)
}

var authSource AuthLookup

func UseAuthSource(src AuthLookup) { authSource = src }

func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := sess.Manager.GetInt64(r.Context(), "userID")
		roleStr := sess.Manager.GetString(r.Context(), "userRole")
		isActive := sess.Manager.GetBool(r.Context(), "userIsActive")

		if userID == 0 || roleStr == "" {
			RedirectToLogin(w, r)
			return
		}

		claims := models.Claims{
			UserID:   userID,
			Role:     models.Role(roleStr),
			IsActive: isActive,
		}

		if authSource != nil {
			role, active, err := authSource.LookupUserAuth(r.Context(), userID)
			if err != nil {
				log.Printf("[RequireAuth] auth lookup failed: uid=%d err=%v", userID, err)
				http.Error(w, "Could not verify account", http.StatusServiceUnavailable)
				return
			}

			// DB values replace cached session values.
			claims.Role = role
			claims.IsActive = active
			sess.Manager.Put(r.Context(), "userRole", string(role))
			sess.Manager.Put(r.Context(), "userIsActive", active)
		}

		if !claims.IsActive {
			RedirectToAwaitVerif(w, r)
			return
		}

		r = r.WithContext(WithClaims(r.Context(), claims))
		next.ServeHTTP(w, r)
	})
}

func RequireRole(allowedRoles ...models.Role) func(http.Handler) http.Handler {
	if len(allowedRoles) == 0 {
		return func(h http.Handler) http.Handler { return h }
	}

	allowed := make(map[models.Role]struct{}, len(allowedRoles))
	for _, role := range allowedRoles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || claims.UserID == 0 {
				RedirectToLogin(w, r)
				return
			}

			if _, ok := allowed[claims.Role]; !ok {
				forbidden(w, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func IsHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

func nextURL(r *http.Request) string {
	return url.QueryEscape(r.URL.RequestURI())
}

func RedirectToLogin(w http.ResponseWriter, r *http.Request) {
	location := "/login?next=" + nextURL(r)
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, location, http.StatusSeeOther)
}

func RedirectToAwaitVerif(w http.ResponseWriter, r *http.Request) {
	location := "/await-verif?next=" + nextURL(r)
	if IsHTMX(r) {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, location, http.StatusSeeOther)
}

func forbidden(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("Access denied."))
		return
	}

	http.Error(w, "Forbidden", http.StatusForbidden)
}
