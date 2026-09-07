package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	// These imports allow the funcs from each path to be used
	"tradielynx"
	"tradielynx/internal/auth"
	"tradielynx/internal/config"
	"tradielynx/internal/db"
	"tradielynx/internal/handlers"
	"tradielynx/internal/models"
	"tradielynx/internal/session"
	"tradielynx/internal/templates/pages"
)

func main() {
	// Load Configuration
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	// config.json is still supported for simple local development.
	// Docker can override it with CONFIG_PATH + DB environment variables.
	configFilePath := os.Getenv("CONFIG_PATH")
	if configFilePath == "" {
		configFilePath = filepath.Join(currentWorkingDir, "config.json")
	}

	cfg, err := config.LoadConfig(configFilePath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	config.AppConfig = cfg

	// Make sure log and data directories exist
	if err := os.MkdirAll(cfg.LogDirectory, 0755); err != nil {
		log.Printf("Warning: Could not create log directory %s: %v", cfg.LogDirectory, err)
	}
	if err := os.MkdirAll(cfg.DataDirectory, 0755); err != nil {
		log.Printf("Warning: Could not create data directory %s: %v", cfg.DataDirectory, err)
	}

	// Called to allow css styling to be used
	tradielynx.UseEmbeddedContent()

	// Initializes the database
	dbConnection, err := db.InitializeDB()
	if err != nil {
		log.Fatal("DB connection failed:", err)
	}
	defer dbConnection.Close()

	// Optional local-dev admin bootstrap. Public registration can never create admins.
	if err := bootstrapAdmin(context.Background(), dbConnection); err != nil {
		log.Fatalf("Failed to bootstrap admin: %v", err)
	}

	// --- Session + auth plumbing -----------------------------------------------

	// 1) Initialize SCS session manager (cookie settings, idle/lifetime, SameSite)
	// Must run once at startup *before* any handlers are registered
	session.InitSession()

	// 2) Provide the session manager to the auth package so middleware/handlers
	// can read/write session values. We store:
	//   - "userID"  (int64)   -> authenticated user id
	//   - "userRole" (string) -> last known role (used as a fast fallback)
	//   - "userIsActive"      -> verified/active account state
	auth.UseSessionManager(session.Manager) // inject scs manager

	// 3) Register a role lookup source used by auth.RequireAuth/RequireRole to
	// refresh the user's role from the DB on requests (decouples auth from data)
	repo := &db.UserRepo{DB: dbConnection}
	auth.UseRoleSource(repo) // register repository for role lookup

	/* ------------------------
	HTTP routes:
	Public      -> no middleware (e.g., /login, /register, assets)
	Auth-only   -> RequireAuth
	Role-gated  -> RequireAuth + RequireRole(...)
	Each protected handler assumes claims are in context (set by RequireAuth).
	*/ //----------------------

	// Handles requests at site's root address
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if session.Manager.Exists(r.Context(), "userID") {
			http.Redirect(w, r, "/dashboard", http.StatusFound)
			return
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})

	// Public (/login + /register + verification wait page)
	http.HandleFunc("/login", handlers.Login(dbConnection))
	http.HandleFunc("/register", handlers.ServeRegister(dbConnection))
	http.HandleFunc("/await-verif", handlers.ServeAwaitVerif(dbConnection))

	// /logout — visible to any authenticated user
	http.Handle("/logout", auth.Guard(http.HandlerFunc(handlers.Logout(dbConnection))))

	// /dashboard — visible to Homeowner + Tradesperson + Admins
	http.Handle("/dashboard",
		auth.Guard(http.HandlerFunc(handlers.ServeDashboard(dbConnection))),
	)

	// /job-listings — visible to Homeowner + Tradesperson + Admins
	http.Handle("/job-listings",
		auth.Guard(http.HandlerFunc(handlers.JobListings(dbConnection)),
			models.RoleAdmin, models.RoleHomeowner, models.RoleTradesperson),
	)

	// /post-job - visible to Homeowner only
	http.Handle("/post-job",
		auth.Guard(
			http.HandlerFunc(
				handlers.ServeNavLink(dbConnection, pages.PostJobPage, "Post Job - "),
			),
			models.RoleHomeowner,
		),
	)

	// /profile — visible to Homeowner + Tradesperson
	http.Handle("/profile",
		auth.Guard(http.HandlerFunc(handlers.ServeNavLink(dbConnection, pages.ProfilePage,
			"Profile - ")), models.RoleHomeowner, models.RoleTradesperson),
	)

	// /my-jobs - visible to Homeowner only
	http.Handle("/my-jobs",
		auth.Guard(http.HandlerFunc(handlers.MyJobs(dbConnection)), models.RoleHomeowner),
	)

	// /new-job - visible to Homeowners only
	http.Handle("/new-job",
		auth.Guard(http.HandlerFunc(handlers.JobsNew(dbConnection)), models.RoleHomeowner),
	)

	// /jobs/{id} — job detail page
	http.Handle("/jobs/",
		auth.Guard(
			http.HandlerFunc(handlers.JobDetail(dbConnection)),
			models.RoleAdmin, models.RoleHomeowner, models.RoleTradesperson,
		),
	)

	// /success - visible to Homeowners only
	http.Handle("/success",
		auth.Guard(
			http.HandlerFunc(handlers.ServeNavLink(dbConnection, pages.PostJobSuccessPage,
				"Success! - ")),
			models.RoleHomeowner,
		),
	)

	// /my-bids - visible to Tradesperson only
	http.Handle("/my-bids",
		auth.Guard(http.HandlerFunc(handlers.MyBids(dbConnection)), models.RoleTradesperson),
	)

	// Small admin surface for the current MVP: approve tradesperson accounts.
	http.Handle("/verif-queue",
		auth.Guard(http.HandlerFunc(handlers.AdminVerifQueue(dbConnection)), models.RoleAdmin),
	)
	http.Handle("/verif-queue/verify",
		auth.Guard(http.HandlerFunc(handlers.VerifyTradesperson(dbConnection)), models.RoleAdmin),
	)

	// Wrap the default mux with the session middleware so handlers can call
	// session.Manager.Get/Put/GetInt64, etc. without manual plumbing.
	mux := http.DefaultServeMux

	// Bid routes are role-gated as well as checked inside the handlers.
	mux.Handle("GET /bids/form",
		auth.Guard(http.HandlerFunc(handlers.BidForm(dbConnection)), models.RoleTradesperson),
	)
	mux.Handle("POST /bids",
		auth.Guard(http.HandlerFunc(handlers.BidCreate(dbConnection)), models.RoleTradesperson),
	)
	mux.Handle("GET /bids/success",
		auth.Guard(http.HandlerFunc(handlers.BidSuccess(dbConnection)), models.RoleTradesperson),
	)

	sHandler := session.Manager.LoadAndSave(mux)
	log.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", sHandler))
}

// Creates the first local admin only when bootstrap credentials are supplied.
// This keeps admin creation out of the public registration form.
func bootstrapAdmin(ctx context.Context, dbConnection *sql.DB) error {
	adminEmail := os.Getenv("BOOTSTRAP_ADMIN_EMAIL")
	adminPassword := os.Getenv("BOOTSTRAP_ADMIN_PASSWORD")

	if adminEmail == "" && adminPassword == "" {
		return nil
	}
	if adminEmail == "" || adminPassword == "" {
		return errors.New("BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD must both be set")
	}

	existing, err := db.RetrieveUserByEmail(ctx, dbConnection, adminEmail)
	if err == nil {
		if existing.Role != models.RoleAdmin {
			return errors.New("bootstrap admin email already belongs to a non-admin account")
		}
		return nil
	}
	if !errors.Is(err, db.ErrUserNotFound) {
		return err
	}

	passwordHash, err := auth.HashPassword(adminPassword)
	if err != nil {
		return err
	}

	admin := models.User{
		FirstName:      "Admin",
		Email:          adminEmail,
		HashedPassword: passwordHash,
		Role:           models.RoleAdmin,
		IsActive:       true,
	}

	if err := db.InsertUser(ctx, dbConnection, admin); err != nil {
		return err
	}

	log.Printf("Local admin account created for %s", adminEmail)
	return nil
}
