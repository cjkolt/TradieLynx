package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"tradielynx/internal/models"
)

// These tests deliberately run serially: auth's session and role providers are
// package globals. Restore both so later tests cannot inherit this fixture.
type testSession struct {
	SessionReader
	userID int64
	role string
	active bool
}

func (s *testSession) GetInt64(context.Context, string) int64 { return s.userID }
func (s *testSession) GetString(context.Context, string) string { return s.role }
func (s *testSession) GetBool(context.Context, string) bool { return s.active }
func (s *testSession) Put(_ context.Context, key string, value any) {
	if key == "userRole" { s.role = value.(string) }
}

type testRoleLookup struct { role models.Role }
func (s testRoleLookup) LookupUserRole(context.Context, int64) (models.Role, error) {
	return s.role, nil
}

func TestGuardAccessDecisions(t *testing.T) {
	tests := []struct {
		name string
		userID int64
		role models.Role
		active bool
		htmx bool
		wantStatus int
		wantRedirect string
		wantHandler bool
	}{
		{"anonymous browser", 0, "", false, false, http.StatusSeeOther, "/login", false},
		{"anonymous HTMX", 0, "", false, true, http.StatusUnauthorized, "/login", false},
		{"missing session role", 42, "", true, false, http.StatusSeeOther, "/login", false},
		{"inactive browser", 42, models.RoleTradesperson, false, false, http.StatusSeeOther, "/await-verif", false},
		{"inactive HTMX", 42, models.RoleTradesperson, false, true, http.StatusUnauthorized, "/await-verif", false},
		{"wrong role browser", 42, models.RoleHomeowner, true, false, http.StatusForbidden, "", false},
		{"wrong role HTMX", 42, models.RoleHomeowner, true, true, http.StatusForbidden, "", false},
		{"allowed browser", 42, models.RoleTradesperson, true, false, http.StatusNoContent, "", true},
		{"allowed HTMX", 42, models.RoleTradesperson, true, true, http.StatusNoContent, "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			oldManager, oldSource := sess.Manager, roleSource
			t.Cleanup(func() { sess.Manager, roleSource = oldManager, oldSource })
			UseSessionManager(&testSession{userID: tc.userID, role: string(tc.role), active: tc.active})
			UseRoleSource(nil)

			called := false
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				claims, ok := ClaimsFromContext(r.Context())
				if !ok || claims.UserID != tc.userID || claims.Role != tc.role || !claims.IsActive {
					t.Errorf("handler claims = %+v, present = %t", claims, ok)
				}
				w.WriteHeader(http.StatusNoContent)
			})
			r := httptest.NewRequest(http.MethodGet, "/jobs/17?tab=bids&sort=new", nil)
			if tc.htmx { r.Header.Set("HX-Request", "true") }
			w := httptest.NewRecorder()
			Guard(next, models.RoleTradesperson).ServeHTTP(w, r)

			if w.Code != tc.wantStatus { t.Errorf("status = %d, want %d", w.Code, tc.wantStatus) }
			if called != tc.wantHandler { t.Errorf("handler called = %t, want %t", called, tc.wantHandler) }
			location, hxRedirect := w.Header().Get("Location"), w.Header().Get("HX-Redirect")
			if tc.wantRedirect == "" {
				if location != "" || hxRedirect != "" { t.Errorf("unexpected redirect: Location=%q HX-Redirect=%q", location, hxRedirect) }
				return
			}
			target := location
			if tc.htmx {
				target = hxRedirect
				if location != "" { t.Errorf("HTMX response unexpectedly has Location=%q", location) }
			} else if hxRedirect != "" { t.Errorf("browser response unexpectedly has HX-Redirect=%q", hxRedirect) }
			u, err := url.Parse(target)
			if err != nil { t.Fatalf("invalid redirect %q: %v", target, err) }
			if u.Path != tc.wantRedirect || u.Query().Get("next") != r.URL.RequestURI() {
				t.Errorf("redirect = %q, want %s preserving %q", target, tc.wantRedirect, r.URL.RequestURI())
			}
		})
	}
}

func TestGuardUsesAuthoritativeRoleBeforeAuthorization(t *testing.T) {
	oldManager, oldSource := sess.Manager, roleSource
	t.Cleanup(func() { sess.Manager, roleSource = oldManager, oldSource })
	session := &testSession{userID: 42, role: string(models.RoleAdmin), active: true}
	UseSessionManager(session)
	UseRoleSource(testRoleLookup{role: models.RoleHomeowner})
	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	w := httptest.NewRecorder()
	Guard(next, models.RoleAdmin).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin", nil))
	if w.Code != http.StatusForbidden || called {
		t.Fatalf("stale admin session authorized: status=%d handler called=%t", w.Code, called)
	}
	if session.role != string(models.RoleHomeowner) {
		t.Errorf("session role = %q, want refreshed homeowner role", session.role)
	}
}

func TestRequireRoleRejectsMissingClaims(t *testing.T) {
	for _, claims := range []models.Claims{{}, {Role: models.RoleAdmin}} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/admin", nil)
		if claims.Role != "" { r = r.WithContext(WithClaims(r.Context(), claims)) }
		called := false
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
		RequireRole(models.RoleAdmin)(next).ServeHTTP(w, r)
		if w.Code != http.StatusSeeOther || called {
			t.Errorf("missing identity authorized: status=%d handler called=%t", w.Code, called)
		}
	}
}
