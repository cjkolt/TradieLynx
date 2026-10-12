# TradieLynx

TradieLynx is a marketplace MVP built to connect homeowners with local tradespeople.

The current version focuses on the core marketplace workflow: homeowners post jobs and review submitted bids, while verified tradespeople search open jobs, place bids, and track whether those bids are accepted or rejected.

This is an active portfolio project and is **not production-ready**. Features that are not part of the current MVP are kept in the roadmap instead of being shown as if they already work.

## Current MVP Features

### Homeowners
- Register and log in
- Post a new job
- Browse open jobs
- View job details
- View their own posted jobs
- Review bids on their jobs
- Award a job to one bidder

### Tradespeople
- Register and wait for account verification
- Log in after admin approval
- Browse and search open jobs by keyword or postal code
- View job details
- Submit one bid per job
- View their submitted bids and bid status

### Admin
- Admin accounts cannot be created through public registration
- Optional local admin bootstrap through environment variables
- View pending tradesperson accounts
- Approve tradespeople for access

### Application / Backend
- Session-based authentication using SCS
- Password hashing with bcrypt
- Role-based authorization for Homeowner, Tradesperson, and Administrator accounts
- PostgreSQL persistence
- Transactional bid awarding so the winner, other bids, and job status update together
- Go server-side rendering with templ
- HTMX for partial page interactions and redirects
- Embedded static assets
- Docker Compose development environment
- PostgreSQL healthcheck before the app starts

## Tech Stack

- **Go 1.24+**
- **PostgreSQL 16**
- **templ**
- **HTMX**
- **SCS** for sessions
- **bcrypt** for password hashing
- **Docker + Docker Compose**

## Project Structure

```text
tradielynx/
├── cmd/app/                 # Application entry point and routes
├── internal/
│   ├── auth/                # Auth middleware, role guards, password helpers
│   ├── config/              # JSON/environment configuration loading
│   ├── db/                  # Database setup and direct DB operations
│   ├── handlers/            # HTTP handlers
│   ├── models/              # Shared application models
│   ├── repo/                # Job/bid read queries
│   ├── session/             # SCS session configuration
│   └── templates/           # templ pages, layouts, and navigation
├── static/                  # CSS and images
├── Dockerfile
├── docker-compose.yml
└── .env.example
```

## Getting Started

### Prerequisites

The easiest way to run the project is with:

- Docker Desktop
- Docker Compose

### 1. Clone the repository

```bash
git clone https://github.com/cjkolt/TradieLynx.git
cd TradieLynx
```

### 2. Create local environment variables

Copy the example file:

```bash
cp .env.example .env
```

Set your local PostgreSQL password in `.env`.

If you want to test the tradesperson verification flow, also fill in:

```text
BOOTSTRAP_ADMIN_EMAIL=
BOOTSTRAP_ADMIN_PASSWORD=
```

Those values are used only to create the first local admin account when one does not already exist. Public registration never accepts the Administrator role.

### 3. Start the app

```bash
docker compose up --build
```

Then open:

```text
http://localhost:8080
```

The app container waits for PostgreSQL to become healthy before starting.

## Local Configuration

`config.json` is still supported for local development, but it is ignored by Git and should never contain credentials that are committed to the repository.

Docker uses environment variables for database credentials. The public `config.example.json` remains available as a reference for the JSON config structure.

## Database Note

The current bid schema links a bid directly to a registered user and allows only one bid per tradesperson per job.

If you are updating an older local development database created from a previous version of `initial_db.sql`, the easiest option during development is to rebuild the PostgreSQL volume:

```bash
docker compose down -v
docker compose up --build
```

**Warning:** `docker compose down -v` deletes the local PostgreSQL development data in that Docker volume.

## Running Without Docker

The Docker workflow generates templ code automatically before running the app.

For direct local Go development, generate the templates first:

```bash
templ generate
go run ./cmd/app
```

You will also need a reachable PostgreSQL database and either `config.json` or the required database environment variables.

## Automated Checks

GitHub Actions generates the templates, compiles and tests all packages, and
runs static analysis on pull requests and pushes to main.

To run the same checks locally with Go 1.24.3 or newer:

```bash
go install github.com/a-h/templ/cmd/templ@v0.3.924
"$(go env GOPATH)/bin/templ" generate
go test ./...
go vet ./...
go build ./cmd/app
```

The unit tests do not require PostgreSQL. They cover password verification,
role parsing, anonymous and inactive access, allowed and forbidden roles,
browser and HTMX redirects, preservation of the original request URL, and
refreshing account role/activity from the database before authorization.

These checks are a baseline. Database-backed registration, job posting, bid
ownership, and complete browser workflows still need integration tests.

## Security Notes

- Passwords are hashed with bcrypt before storage.
- `config.json` and `.env` are ignored by Git.
- Public registration accepts only Homeowner and Tradesperson roles.
- Protected routes use authentication and role middleware.
- Role and active account state are refreshed from PostgreSQL before protected requests are authorized.
- Bid ownership is checked before displaying a submitted bid confirmation.
- Database credentials are supplied through local environment variables rather than being hardcoded in Docker Compose.

This is still an MVP. Production deployment would need additional hardening, HTTPS-only cookies, stronger input validation, CSRF protections, rate limiting, and production secrets management.

## Possible Future Work

- Direct messaging after a job is awarded
- Richer tradesperson profiles and qualification uploads
- Ratings and reviews after job completion
- Database-backed integration tests and browser workflow tests
- Production deployment and security hardening

## Project Status

**MVP / active development.**

The portfolio MVP is intentionally kept small: posting, verification, job discovery, bidding, bid awarding, and role-based access are the focus.
