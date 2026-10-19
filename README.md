# TradieLynx

A small marketplace for homeowners to post jobs and tradespeople to bid on them.

I built this as a portfolio project to work through a complete web application, including the parts you don't see in a screenshot: accounts, permissions, database state, and what happens when someone awards a job.

**Status:** Working MVP under development. Not a live marketplace or a production-ready service.

## What you can do

A homeowner can create an account, post a job, review bids, and choose a tradesperson. Tradespeople register separately, require admin approval, and can browse jobs, filter by keyword or postal code, and submit a bid. Each tradesperson can bid once per job.

There's a basic admin flow for approving tradesperson accounts. Public registration cannot create admins.

## The part I'd look at first

**[Bid awarding](internal/repo/bids.go)** is the backend workflow I'm most interested in getting right.

Accepting a bid isn't just changing one field. The chosen bid needs to become accepted, competing submitted bids need to become rejected, and the job needs to move out of the open state. Those updates happen in a PostgreSQL transaction. The code also locks the relevant job and bid rows, checks ownership, and rejects an award if the job is no longer open.

The other area worth exploring is **[authentication and authorization](internal/auth/)**: separate user roles, bcrypt password hashing, server-side sessions, and access checks on protected routes.

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

The CI workflow runs template generation, `go test ./...`, `go vet ./...`, and a build. Current unit tests focus mainly on authentication and authorization behavior. They **do not** replace database-backed integration tests for registration, bidding, or simultaneous award requests.

This project still needs those tests, a complete browser walkthrough, and production security work (including CSRF protection, production cookie settings, input validation, rate limiting, and secrets management) before it could responsibly be deployed for real users.

The local PostgreSQL data is disposable during development. **`docker compose down -v` deletes its volume and all data in it.**

I haven't added payments, messaging, or reviews. Right now I'd rather make the central job-and-bid workflow dependable than broaden the feature list.

---
*This is a portfolio MVP, not an operating trades marketplace.*
