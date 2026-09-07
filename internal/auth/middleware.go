package auth

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strings"
	"tradielynx/internal/models"
)

// Abstracts the session store
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

// Holds the package-level session manager (injected by main)
var sess struct{ Manager SessionReader }

// Called once from main() during startup.
func UseSessionManager(m SessionReader) { sess.Manager = m }

// The data source contract for fetching a user’s role.
// Keeping it as an interface lets us swap DB, cache, or a mock in tests
type RoleLookup interface {
	LookupUserRole(ctx context.Context, userID int64) (models.Role, error)
}

// The active RoleLookup used by middleware
var roleSource RoleLookup

// Called once from main() during startup.
func UseRoleSource(src RoleLookup) { roleSource = src }

var ErrUserNotFound = errors.New("user not found")

// Ensures the request is from an authenticated user
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := sess.Manager.GetInt64(r.Context(), "userID")
		roleStr := sess.Manager.GetString(r.Context(), "userRole")
		isActive := sess.Manager.GetBool(r.Context(), "userIsActive")

		// no session user -> treat as logged out
		if userID == 0 || roleStr == "" {
			RedirectToLogin(w, r)
			return
		}

		// Tradesperson !verified -> redirect to wait page.
		if !isActive {
			RedirectToAwaitVerif(w, r)
			return
		}

		// build claims from what's in the session first
		claims := models.Claims{
			UserID:   userID,
			Role:     models.Role(roleStr),
			IsActive: isActive,
		}

		if roleSource != nil {
			if role, err := roleSource.LookupUserRole(r.Context(), userID); err == nil {
				// DB role is authoritative when the lookup succeeds
				claims.Role = role
				sess.Manager.Put(r.Context(), "userRole", string(role))
			} else {
				// keep session role on a DB error
				log.Printf("[RequireAuth] role lookup failed; falling back to session role: uid=%d err=%v sessRole=%q",
					userID, err, roleStr)
			}
		}

		// attach claims to the request context so downstream handlers can read them
		r = r.WithContext(WithClaims(r.Context(), claims))
		next.ServeHTTP(w, r) // continue to the next handler in the chain
	})
}

// Gates a handler by one or more roles.
// - accepts the given role(s) as a dynamic array of strings
// - must be used *after* RequireAuth
func RequireRole(allowedRoles ...models.Role) func(http.Handler) http.Handler {
	if len(allowedRoles) == 0 { // Variadic slice is empty -> only auth required.
		return func(h http.Handler) http.Handler { return h }
	}

	// build a set for membership checks instead of scanning a slice each request
	allowed := make(map[models.Role]struct{}, len(allowedRoles))
	for _, ar := range allowedRoles {
		allowed[ar] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// pull auth claims the RequireAuth middleware attached to the context
			claims, ok := ClaimsFromContext(r.Context())
			if !ok || claims.UserID == 0 {
				// No user in context -> treat as unauthenticated and bounce to login
				RedirectToLogin(w, r)
				return
			}

			// “is member?” test against the set of allowed roles
			if _, ok := allowed[claims.Role]; !ok {
				forbidden(w, r) // if user's role not in allowed set
				return
			}
			next.ServeHTTP(w, r) // authorized: pass request down the chain
		})
	}
}

// Returns true if the session contains a non-zero userID
func UserIsAuthed(userRole string, requiredRole string, r *http.Request) bool {
	userID := sess.Manager.GetInt64(r.Context(), "userID")
	return userID != 0 && userRole == requiredRole
}

// Reports whether the request came from HTMX (HX-Request: true or boosted)
func IsHTMX(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("HX-Request"), "true")
}

// Returns a safe post-login redirect target (uses ?next=…,
// falls back to "/"; only relative paths).
func nextURL(r *http.Request) string {
	return url.QueryEscape(r.URL.RequestURI())
}

// Sends the user to /login, preserving ?next=<original path>.
func RedirectToLogin(w http.ResponseWriter, r *http.Request) {
	location := "/login?next=" + nextURL(r)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, location, http.StatusSeeOther)
}

// Sends the user to /await-verif, preserving ?next=<original path>.
func RedirectToAwaitVerif(w http.ResponseWriter, r *http.Request) {
	location := "/await-verif?next=" + nextURL(r)
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", location)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	http.Redirect(w, r, location, http.StatusSeeOther)
}

// Writes a 403 Forbidden response for role violations.
func forbidden(w http.ResponseWriter, r *http.Request) {
	if IsHTMX(r) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(([]byte("Access denied.")))
		return
	}
	http.Error(w, "Forbidden", http.StatusForbidden)
}

// sends a 302 to the login page, preserving the current URL as ?next=…
func UnauthorizedRedirect(w http.ResponseWriter, r *http.Request) {
	// 302 + Location
	http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.RequestURI()), http.StatusFound)
}

// Returns a 401 and instructs HTMX to redirect to login (e.g., via HX-Redirect header)
func UnauthorizedHTMX(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("HX-Redirect", "/login?next="+url.QueryEscape(r.URL.RequestURI()))
	w.WriteHeader(http.StatusUnauthorized) // 401
	// optional tiny body
	_, _ = w.Write([]byte("Unauthorized"))
}
