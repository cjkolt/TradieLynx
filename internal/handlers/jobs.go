package handlers

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"

	"tradielynx/internal/db"
	"tradielynx/internal/models"
	"tradielynx/internal/repo"
	"tradielynx/internal/session"
	"tradielynx/internal/templates/base"
	"tradielynx/internal/templates/pages"
	"tradielynx/internal/templates/shared"

	"github.com/a-h/templ"
)

func JobsNew(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseForm(); err != nil {
			_ = shared.Error("Invalid form submission").Render(r.Context(), w)
			return
		}

		title := strings.TrimSpace(r.FormValue("job-title"))
		description := strings.TrimSpace(r.FormValue("job-description"))
		postcode := strings.TrimSpace(r.FormValue("job-postcode"))
		preferredTimeframe := strings.TrimSpace(r.FormValue("job-timeframe"))
		budgetMin := strings.TrimSpace(r.FormValue("job-min"))
		budgetMax := strings.TrimSpace(r.FormValue("job-max"))

		if title == "" || description == "" || postcode == "" {
			_ = shared.Error("Title, description, and postal code are required.").Render(r.Context(), w)
			return
		}

		budgetMinMax := ""
		switch {
		case budgetMin != "" && budgetMax != "":
			budgetMinMax = budgetMin + " - " + budgetMax
		case budgetMin != "":
			budgetMinMax = "From " + budgetMin
		case budgetMax != "":
			budgetMinMax = "Up to " + budgetMax
		}

		// get user id from SCS (RequireAuth already guaranteed it's present)
		userID := session.Manager.GetInt64(r.Context(), "userID")

		job := models.Job{
			HomeownerID:        userID,
			Title:              title,
			Description:        description,
			PostCode:           postcode,
			PreferredTimeframe: preferredTimeframe,
			BudgetMinMax:       budgetMinMax,
			Status:             models.JobOpen,
		}

		if err := db.InsertJob(r.Context(), dbConn, job); err != nil {
			_ = shared.Error("Could not post job. Please try again.").Render(r.Context(), w)
			return
		}

		// Success
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("HX-Redirect", "/success")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.Redirect(w, r, "/success", http.StatusSeeOther)
	}
}

func JobListings(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// session -> userID (auth.Guard/RequireAuth should guarantee this)
		userID := session.Manager.GetInt64(r.Context(), "userID")
		if userID == 0 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		// load user (for nav/header + page)
		user, err := db.RetrieveUserByID(r.Context(), dbConn, userID)
		if err != nil {
			// HTMX: fragment error; non-HTMX: full base error
			if r.Header.Get("HX-Request") == "true" {
				templ.Handler(shared.Error("Could not load user")).ServeHTTP(w, r)
			} else {
				templ.Handler(base.Base(models.Data{Title: "Error - "},
					shared.Error("Could not load user"))).ServeHTTP(w, r)
			}
			return
		}

		loggedUser := models.LoggedUser{
			ID:        user.ID,
			FirstName: user.FirstName,
			Role:      user.Role,
			IsActive:  user.IsActive,
		}

		// Keep the query bounded even though pagination is not shown in the MVP UI yet.
		jobs, err := repo.ListOpenJobs(r.Context(), dbConn, 50, 0)
		if err != nil {
			if r.Header.Get("HX-Request") == "true" {
				templ.Handler(shared.Error("Failed to load jobs")).ServeHTTP(w, r)
			} else {
				templ.Handler(base.Base(models.Data{Title: "Job Listings - ", User: loggedUser},
					shared.Error("Failed to load jobs"))).ServeHTTP(w, r)
			}
			return
		}

		page := pages.JobListingsPage(loggedUser, jobs)

		// HTMX nav -> fragment-only
		if r.Header.Get("HX-Request") == "true" {
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		// full GET -> wrap in base layout (this brings back nav + CSS)
		templ.Handler(base.Base(models.Data{Title: "Job Listings - ", User: loggedUser}, page)).
			ServeHTTP(w, r)
	}
}

func JobDetail(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Expect /jobs/{id}
		path := strings.TrimPrefix(r.URL.Path, "/jobs/")
		if path == "" || path == r.URL.Path {
			http.NotFound(w, r)
			return
		}

		// If someone hits /jobs/123/anything, treat as not found for now
		if strings.Contains(path, "/") {
			http.NotFound(w, r)
			return
		}

		jobID, err := strconv.ParseInt(path, 10, 64)
		if err != nil || jobID <= 0 {
			http.NotFound(w, r)
			return
		}

		// Load logged user for base layout/nav
		userID := session.Manager.GetInt64(r.Context(), "userID")
		if userID == 0 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		u, err := db.RetrieveUserByID(r.Context(), dbConn, userID)
		if err != nil {
			if r.Header.Get("HX-Request") == "true" {
				templ.Handler(shared.Error("Could not load user")).ServeHTTP(w, r)
			} else {
				templ.Handler(base.Base(models.Data{Title: "Error - "},
					shared.Error("Could not load user"))).ServeHTTP(w, r)
			}
			return
		}

		logged := models.LoggedUser{
			ID:        u.ID,
			FirstName: u.FirstName,
			Role:      u.Role,
			IsActive:  u.IsActive,
		}

		// Load job
		job, err := repo.GetJobByID(r.Context(), dbConn, jobID)
		if err != nil {
			if err == repo.ErrJobNotFound {
				w.WriteHeader(http.StatusNotFound)
				msg := "Job not found"

				if r.Header.Get("HX-Request") == "true" {
					templ.Handler(shared.Error(msg)).ServeHTTP(w, r)
					return
				}

				templ.Handler(
					base.Base(
						models.Data{Title: "Job not found - ", User: logged},
						shared.Error(msg),
					),
				).ServeHTTP(w, r)
				return
			}

			// non-404 errors (DB down, etc)
			if r.Header.Get("HX-Request") == "true" {
				templ.Handler(shared.Error("Failed to load job")).ServeHTTP(w, r)
			} else {
				templ.Handler(
					base.Base(
						models.Data{Title: "Job - ", User: logged},
						shared.Error("Failed to load job"),
					),
				).ServeHTTP(w, r)
			}
			return
		}

		page := pages.JobDetailPage(logged, job)

		// HTMX swap: return body only
		if r.Header.Get("HX-Request") == "true" {
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		// Full page: wrap in base (nav + CSS)
		templ.Handler(base.Base(models.Data{Title: "Job - ", User: logged}, page)).
			ServeHTTP(w, r)
	}
}

func BidForm(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jobIDStr := r.URL.Query().Get("job_id")
		jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
		if err != nil || jobID <= 0 {
			templ.Handler(shared.Error("Invalid job id")).ServeHTTP(w, r)
			return
		}

		job, err := repo.GetJobByID(r.Context(), dbConn, jobID)
		if err != nil || job.Status != models.JobOpen {
			templ.Handler(shared.Error("This job is not available for bidding.")).ServeHTTP(w, r)
			return
		}

		templ.Handler(pages.BidFormFragment(jobID)).ServeHTTP(w, r)
	}
}

func BidCreate(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			templ.Handler(shared.Error("Invalid form")).ServeHTTP(w, r)
			return
		}

		// auth
		userID := session.Manager.GetInt64(r.Context(), "userID")
		if userID == 0 {
			// HTMX-friendly: redirect to login
			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/login")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		u, err := db.RetrieveUserByID(r.Context(), dbConn, userID)
		if err != nil {
			templ.Handler(shared.Error("Could not load user")).ServeHTTP(w, r)
			return
		}

		// Only verified tradespeople
		if u.Role != models.RoleTradesperson || !u.IsActive {
			templ.Handler(shared.Error("Only verified tradespeople can place bids.")).ServeHTTP(w, r)
			return
		}

		jobID, err := strconv.ParseInt(r.FormValue("job_id"), 10, 64)
		if err != nil || jobID <= 0 {
			templ.Handler(shared.Error("Invalid job id")).ServeHTTP(w, r)
			return
		}

		job, err := repo.GetJobByID(r.Context(), dbConn, jobID)
		if err != nil || job.Status != models.JobOpen {
			templ.Handler(shared.Error("This job is not available for bidding.")).ServeHTTP(w, r)
			return
		}

		amount := strings.TrimSpace(r.FormValue("amount"))
		message := strings.TrimSpace(r.FormValue("message"))

		if amount == "" {
			// Return form again as fragment with error
			templ.Handler(pages.BidFormFragmentWithError(jobID, "Amount is required")).ServeHTTP(w, r)
			return
		}

		bid := models.Bid{
			JobID:          jobID,
			TradespersonID: userID,
			Amount:         amount,
			Message:        message,
			Status:         models.BidSubmitted,
		}

		bidID, err := db.InsertBid(r.Context(), dbConn, bid)
		if err != nil {
			templ.Handler(pages.BidFormFragmentWithError(jobID,
				"Could not submit bid. You may have already bid on this job.")).ServeHTTP(w, r)
			return
		}

		// PRG redirect
		redirectURL := "/bids/success?bid_id=" + strconv.FormatInt(bidID, 10)

		if r.Header.Get("HX-Request") == "true" {
			// HTMX: tell the browser to navigate (full page)
			w.Header().Set("HX-Redirect", redirectURL)
			w.WriteHeader(http.StatusNoContent) // 204: nothing to swap
			return
		}

		// Non-HTMX fallback (normal browser POST)
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
	}
}

func BidSuccess(dbConn *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// auth
		userID := session.Manager.GetInt64(r.Context(), "userID")
		if userID == 0 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		bidIDStr := r.URL.Query().Get("bid_id")
		bidID, err := strconv.ParseInt(bidIDStr, 10, 64)
		if err != nil || bidID <= 0 {
			templ.Handler(shared.Error("Invalid bid id")).ServeHTTP(w, r)
			return
		}

		bid, err := db.RetrieveBidByID(r.Context(), dbConn, bidID)
		if err != nil {
			templ.Handler(shared.Error("Bid not found")).ServeHTTP(w, r)
			return
		}

		// Security check: only the bid owner can view
		if bid.TradespersonID != userID {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// logged user data for nav
		u, err := db.RetrieveUserByID(r.Context(), dbConn, userID)
		if err != nil {
			templ.Handler(shared.Error("Could not load user")).ServeHTTP(w, r)
			return
		}
		logged := models.LoggedUser{
			ID:        u.ID,
			FirstName: u.FirstName,
			Role:      u.Role,
			IsActive:  u.IsActive,
		}

		page := pages.BidsSuccessPage(logged, bid)

		templ.Handler(
			base.Base(models.Data{Title: "Bid posted - ", User: logged}, page),
		).ServeHTTP(w, r)
	}
}

func MyJobs(dbConn *sql.DB) http.HandlerFunc {
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

		logged := models.LoggedUser{
			ID:        user.ID,
			FirstName: user.FirstName,
			Role:      user.Role,
			IsActive:  user.IsActive,
		}

		jobs, err := repo.ListJobsByOwner(r.Context(), dbConn, userID)
		if err != nil {
			templ.Handler(shared.Error("Could not load your jobs")).ServeHTTP(w, r)
			return
		}

		page := pages.MyJobsPage(logged, jobs)
		if r.Header.Get("HX-Request") == "true" {
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		templ.Handler(base.Base(models.Data{Title: "My Jobs - ", User: logged}, page)).ServeHTTP(w, r)
	}
}

func MyBids(dbConn *sql.DB) http.HandlerFunc {
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

		logged := models.LoggedUser{
			ID:        user.ID,
			FirstName: user.FirstName,
			Role:      user.Role,
			IsActive:  user.IsActive,
		}

		bids, err := repo.ListBidsByTradesperson(r.Context(), dbConn, userID)
		if err != nil {
			templ.Handler(shared.Error("Could not load your bids")).ServeHTTP(w, r)
			return
		}

		page := pages.MyBidsPage(logged, bids)
		if r.Header.Get("HX-Request") == "true" {
			templ.Handler(page).ServeHTTP(w, r)
			return
		}

		templ.Handler(base.Base(models.Data{Title: "My Bids - ", User: logged}, page)).ServeHTTP(w, r)
	}
}
