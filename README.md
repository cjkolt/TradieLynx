# TradieLynx

A small marketplace for homeowners to post jobs and tradespeople to bid on them.

I built this as a portfolio project to work through a complete web application, including the parts you don't see in a screenshot: accounts, permissions, database state, and what happens when someone awards a job.

**Status:** Completed portfolio MVP. The core job-and-bid workflows are implemented and backed by automated checks. Not a live marketplace or production-ready service.

## What you can do

A homeowner can create an account, post a job, review bids, and choose a tradesperson. Tradespeople register separately, require admin approval, and can browse jobs, filter by keyword or postal code, and submit a bid. Each tradesperson can bid once per job.

There's a basic admin flow for approving tradesperson accounts. Public registration cannot create admins.

## The part I'd look at first

**[Bid awarding](internal/repo/bids.go)** is the backend workflow I spent the most time getting right.

Accepting a bid isn't just changing one field. The chosen bid needs to become accepted, competing submitted bids need to become rejected, and the job needs to move out of the open state. Those updates happen in a PostgreSQL transaction. The code also locks the relevant job and bid rows, checks ownership, and rejects an award if the job is no longer open.

The other area worth exploring is **[authentication and authorization](internal/auth/)**: separate user roles, bcrypt password hashing, server-side sessions, and access checks on protected routes.

## See the workflow

I haven't added screenshots of the real application yet. The [demo checklist](docs/demo-checklist.md) covers the short walkthrough I'd like to capture without exposing real user data.

## Stack

- Go, PostgreSQL
- `templ` and HTMX for server-rendered pages and smaller interactions
- SCS sessions, bcrypt
- Docker Compose for local development
- GitHub Actions for build, test, and static checks

The application is server-rendered rather than split into a separate JavaScript frontend and API.

## Run it locally

You'll need Docker and Docker Compose.

```bash
git clone https://github.com/cjkolt/TradieLynx.git
cd TradieLynx
cp .env.example .env
```

Set your local database password in `.env`, then run:

```bash
docker compose up --build
```

Open **http://localhost:8080**.

To try admin approval, set `BOOTSTRAP_ADMIN_EMAIL` and `BOOTSTRAP_ADMIN_PASSWORD` in `.env` before starting with a fresh local database. These are for local bootstrap only.

For non-Docker development, generate templates with `templ generate` and run `go run ./cmd/app` with PostgreSQL and configuration available.

## Tests and current limits

The CI workflow generates templates, compiles the application, runs unit tests, and checks the code with `go vet`. It also starts a disposable PostgreSQL service and runs integration tests for the bid-awarding transaction.

The database tests cover a successful award, incorrect ownership, mismatched bids, an already-awarded job, rollback on a database write error, and two simultaneous award attempts. You can run them locally with a disposable PostgreSQL database:

```bash
TRADIELYNX_TEST_DATABASE_URL='postgres://postgres:postgres@localhost:5432/tradielynx_ci?sslmode=disable' \
  go test -tags=integration -count=1 ./internal/repo
```

The test database account needs permission to create and remove schemas. Don't point the test suite at a production database.

These tests aren't a complete browser walkthrough. Account registration, job posting, and the entire browser flow still need end-to-end coverage. Production deployment would also need CSRF protection, HTTPS-only cookie settings, rate limiting, stronger input validation, and secrets management.

The local PostgreSQL data is disposable during development. **`docker compose down -v` deletes its volume and all data in it.**

Payments, messaging, and reviews were outside the scope of this portfolio build.

---
*Completed portfolio MVP. Not an operating trades marketplace.*
