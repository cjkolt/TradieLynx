package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"tradielynx/internal/db"
	"tradielynx/internal/models"
	"tradielynx/internal/session"
	"tradielynx/internal/templates/base"
	"tradielynx/internal/templates/pages"
	"tradielynx/internal/templates/shared"

	"github.com/a-h/templ"
)

// Handles call to render dashboard from login
func ServeDashboard(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// get user id from SCS (requireAuth already guaranteed it's present)
		userID := session.Manager.GetInt64(r.Context(), "userID")

		// load user data
		user, err := db.RetrieveUserByID(r.Context(), dbConn, userID)
		if err != nil {
			templ.Handler(base.Base(
				models.Data{Title: "Error"},
				shared.Error("Could not load user"),
			)).ServeHTTP(w, r)
			return
		}

		loggedUser := models.LoggedUser{
			ID:        user.ID,
			FirstName: user.FirstName,
			Role:      user.Role,
			IsActive:  user.IsActive,
		}
		data := models.Data{
			Title: "",
			User:  loggedUser,
		}
		page := pages.DashboardPage(data)

		hxReq := r.Header.Get("HX-Request") == "true"
		hxBoosted := r.Header.Get("HX-Boosted") == "true"
		hxTarget := r.Header.Get("HX-Target")

		if hxReq && !hxBoosted && hxTarget == "content" {
			// fragment-only response for targeted swaps
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		// normal GET -> Base + Page
		templ.Handler(base.Base(data, page)).ServeHTTP(w, r)
	}
}

// Handles call to render a nav-linked page for a logged-in user
func ServeNavLink(dbConn *sql.DB, pageFunc func(models.LoggedUser) templ.Component,
	title string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// session -> userID (requireAuth guarantees session exists)
		userID := session.Manager.GetInt64(r.Context(), "userID")

		// load user data for header/nav + page needs
		user, err := db.RetrieveUserByID(r.Context(), dbConn, userID)
		if err != nil {
			// HTMX: send fragment error (keeps the rest of the page intact)
			if r.Header.Get("HX-Request") == "true" {
				templ.Handler(shared.Error("Could not load user")).ServeHTTP(w, r)
			} else {
				// full-page error (includes base layout)
				templ.Handler(base.Base(models.Data{Title: "Error - "},
					shared.Error("Could not load user"))).
					ServeHTTP(w, r)
			}
			return
		}

		loggedUser := models.LoggedUser{
			ID:        user.ID,
			FirstName: user.FirstName,
			Role:      user.Role,
			IsActive:  user.IsActive,
		}
		page := pageFunc(loggedUser) // builds page content w/ user context

		// HTMX nav -> return only the fragment (no layout)
		if r.Header.Get("HX-Request") == "true" {
			templ.Handler(page).ServeHTTP(w, r)
			return
		}
		//  Full GET -> wrap content in base layout (title + user in chrome)
		templ.Handler(base.Base(models.Data{Title: title, User: loggedUser}, page)).
			ServeHTTP(w, r)
	}
}

// Handles the small admin verification queue used by this MVP stage.
func AdminVerifQueue(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID := session.Manager.GetInt64(r.Context(), "userID")
		user, err := db.RetrieveUserByID(r.Context(), dbConn, userID)
		if err != nil {
			templ.Handler(shared.Error("Could not load user")).ServeHTTP(w, r)
			return
		}

		loggedUser := models.LoggedUser{
			ID:        user.ID,
			FirstName: user.FirstName,
			Role:      user.Role,
			IsActive:  user.IsActive,
		}

		pending, err := db.ListPendingTradespeople(r.Context(), dbConn)
		if err != nil {
			templ.Handler(shared.Error("Could not load pending verifications")).ServeHTTP(w, r)
			return
		}

		page := pages.AdminVerifQueuePage(loggedUser, pending)
		if r.Header.Get("HX-Request") == "true" {
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		templ.Handler(base.Base(models.Data{Title: "Verifications - ", User: loggedUser}, page)).
			ServeHTTP(w, r)
	}
}

func VerifyTradesperson(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form submission", http.StatusBadRequest)
			return
		}

		userID, err := strconv.ParseInt(r.FormValue("user_id"), 10, 64)
		if err != nil || userID <= 0 {
			http.Error(w, "Invalid user id", http.StatusBadRequest)
			return
		}

		if err := db.ActivateTradesperson(r.Context(), dbConn, userID); err != nil {
			http.Error(w, "Could not verify tradesperson", http.StatusBadRequest)
			return
		}

		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/verif-queue")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.Redirect(w, r, "/verif-queue", http.StatusSeeOther)
	}
}
