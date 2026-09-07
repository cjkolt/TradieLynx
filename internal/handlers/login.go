package handlers

import (
	"database/sql"
	"net/http"

	"tradielynx/internal/auth"
	"tradielynx/internal/db"
	"tradielynx/internal/session"
	"tradielynx/internal/templates/base"
	"tradielynx/internal/templates/pages"
	"tradielynx/internal/templates/shared"

	"github.com/a-h/templ"
)

// Handles call to login a user
func Login(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// HTMX nav -> return fragment only
			hxReq := r.Header.Get("HX-Request") == "true"
			hxBoosted := r.Header.Get("HX-Boosted") == "true"
			hxTarget := r.Header.Get("HX-Target")

			// Only fragment when it’s an explicit swap into #content
			if hxReq && !hxBoosted && hxTarget == "content" {
				templ.Handler(pages.LoginPage()).ServeHTTP(w, r)
				return
			}

			// Otherwise (normal or boosted) send full layout
			templ.Handler(base.AuthBase("Tradielynx", pages.LoginPage())).ServeHTTP(w, r)
			return

		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				_ = shared.Error("Invalid form submission").Render(r.Context(), w)
				return
			}
			email := db.NormalizeString(r.FormValue("email"))
			password := r.FormValue("password")

			// auth lookup
			authUser, err := db.RetrieveUserByEmail(r.Context(), dbConn, email)
			if err != nil {
				_ = shared.Error("Invalid credentials").Render(r.Context(), w)
				return
			}
			if err := auth.CompareHashAndPassword(authUser.HashedPassword, password); err != nil {
				_ = shared.Error("Invalid credentials").Render(r.Context(), w)
				return
			}

			// Tradesperson exists, but is still waiting for verification.
			if !authUser.IsActive {
				auth.RedirectToAwaitVerif(w, r)
				return
			}

			// store session
			_ = session.Manager.RenewToken(r.Context())
			session.Manager.Put(r.Context(), "userID", int64(authUser.ID))
			session.Manager.Put(r.Context(), "userRole", string(authUser.Role))
			session.Manager.Put(r.Context(), "userIsActive", authUser.IsActive)

			// Success -> redirect (HTMX vs non-HTMX)
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/dashboard")
				w.WriteHeader(http.StatusNoContent) // 204, no body
				return
			}

			http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func Logout(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Clear Cookie
		_ = session.Manager.Destroy(r.Context())

		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/login")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// fallback to normal browser
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func ServeAwaitVerif(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		page := pages.VerifWaitPage()

		if r.Header.Get("HX-Request") == "true" {
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		templ.Handler(base.AuthBase("Verification - Tradielynx", page)).ServeHTTP(w, r)
	}
}
