# TradieLynx

TradieLynx is a marketplace MVP built to connect homeowners with local tradespeople.

The current version focuses on the core workflow only: homeowners can post jobs, verified tradespeople can browse those jobs and submit bids, and each user can view the jobs or bids that belong to them.

This is an active portfolio project and is **not production-ready**. Features that are not part of the current MVP are kept in the roadmap instead of being shown as if they already work.

## Current MVP Features

### Homeowners
- Register and log in
- Post a new job
- Browse open jobs
- View job details
- View their own posted jobs

### Tradespeople
- Register and wait for account verification
- Log in after admin approval
- Browse open jobs
- View job details
- Submit one bid per job
- View their submitted bids

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
git clone <your-repository-url>
cd tradielynx
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

## Security Notes

- Passwords are hashed with bcrypt before storage.
- `config.json` and `.env` are ignored by Git.
- Public registration accepts only Homeowner and Tradesperson roles.
- Protected routes use authentication and role middleware.
- Bid ownership is checked before displaying a submitted bid confirmation.
- Database credentials are supplied through local environment variables rather than being hardcoded in Docker Compose.

This is still an MVP. Production deployment would need additional hardening, HTTPS-only cookies, stronger input validation, CSRF protections, rate limiting, and production secrets management.

## Roadmap

Not part of the current MVP yet:

- Tradesperson profile and qualification uploads
- Homeowner accepting / awarding bids
- Direct messaging
- Ratings and reviews
- Job images
- Notifications
- Search and filtering
- More complete admin tools
- Automated tests and CI
- Production deployment configuration

## Project Status

**MVP / active development.**

The current goal is to keep the core marketplace workflow small, functional, and understandable before expanding into the roadmap features.
