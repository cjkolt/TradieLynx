package handlers

import (
	"database/sql"
	"net/http"
	"strings"

	"tradielynx/internal/auth"
	"tradielynx/internal/db"
	"tradielynx/internal/models"
	"tradielynx/internal/templates/base"
	"tradielynx/internal/templates/pages"
	"tradielynx/internal/templates/shared"

	"github.com/a-h/templ"
)

// Handles call to register new user
func ServeRegister(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			// HTMX nav → return fragment only
			if r.Header.Get("HX-Request") == "true" {
				templ.Handler(pages.RegisterPage()).ServeHTTP(w, r)
				return
			}
			// Full page load
			templ.Handler(base.AuthBase("Tradielynx", pages.RegisterPage())).ServeHTTP(w, r)
			return

		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				_ = shared.Error("Invalid form submission").Render(r.Context(), w)
				return
			}
			firstName := strings.TrimSpace(r.FormValue("first_name"))
			lastName := strings.TrimSpace(r.FormValue("last_name"))
			email := db.NormalizeString(r.FormValue("email"))
			password := r.FormValue("password")
			role := r.FormValue("role")

			roleVal, ok := models.ParseRole(role)
			if !ok || roleVal == models.RoleAdmin {
				// Admin accounts are never created from the public registration form.
				http.Error(w, "invalid role", http.StatusBadRequest)
				return
			}

			passHash, err := auth.HashPassword(password)
			if err != nil {
				_ = shared.Error("Could not process password.").Render(r.Context(), w)
				return
			}

			// boolean for verified tradesperson status at this MVP stage
			isActive := true
			if roleVal == models.RoleTradesperson {
				isActive = false
			}

			user := models.User{
				FirstName:      firstName,
				LastName:       lastName,
				Email:          email,
				HashedPassword: passHash,
				Role:           roleVal,
				IsActive:       isActive,
			}

			err = db.InsertUser(r.Context(), dbConn, user)
			if err != nil {
				_ = shared.Error("This email is already taken.").Render(r.Context(), w)
				return
			}

			if user.IsActive {
				// Success -> go to login (HTMX vs non-HTMX)
				if r.Header.Get("HX-Request") == "true" {
					w.Header().Set("HX-Redirect", "/login")
					w.WriteHeader(http.StatusNoContent)
					return
				}
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}

			// Tradesperson waits for verification before logging in.
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/await-verif")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Redirect(w, r, "/await-verif", http.StatusSeeOther)
			return
		}

		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
