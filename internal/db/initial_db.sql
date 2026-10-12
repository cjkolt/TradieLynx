-- extensions
CREATE EXTENSION IF NOT EXISTS citext;


-- enums
CREATE TYPE user_role_enum AS ENUM ('Homeowner','Tradesperson','Administrator');

CREATE TYPE status_enum AS ENUM ('Open','Awarded','Completed','Cancelled');

CREATE TYPE bids_status_enum AS ENUM ('Submitted','Accepted','Rejected');


-- tables
CREATE TABLE IF NOT EXISTS users (
  id               BIGSERIAL PRIMARY KEY,
  first_name       VARCHAR(32) NOT NULL,
  last_name        VARCHAR(32),
  email            CITEXT NOT NULL UNIQUE,
  password_hash    TEXT NOT NULL,
  user_role        user_role_enum NOT NULL,
  active           BOOLEAN NOT NULL DEFAULT FALSE, -- false for tradespeople until admin verification
  user_timestamps  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

  CHECK (char_length(trim(first_name)) > 0),
  CHECK (char_length(trim(email::text)) > 0)
);


CREATE TABLE IF NOT EXISTS jobs (
  id                  BIGSERIAL PRIMARY KEY,
  homeowner_user_id   BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  title               TEXT NOT NULL,
  description         TEXT NOT NULL,
  postcode            TEXT NOT NULL,
  preferred_timeframe TEXT,
  budget_min_max      TEXT,
  status              status_enum NOT NULL DEFAULT 'Open',

  CHECK (char_length(trim(title)) > 0),
  CHECK (char_length(trim(description)) > 0),
  CHECK (char_length(trim(postcode)) > 0)
);


CREATE TABLE IF NOT EXISTS bids (
  id                   BIGSERIAL PRIMARY KEY,
  job_id               BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  tradesperson_user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  amount               NUMERIC(10,2) NOT NULL CHECK (amount > 0),
  message              TEXT,
  status               bids_status_enum NOT NULL DEFAULT 'Submitted',
  bids_timestamp       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,

  UNIQUE (job_id, tradesperson_user_id) -- one bid per tradesperson per job
);


-- indexes
CREATE INDEX IF NOT EXISTS idx_jobs_homeowner
  ON jobs (homeowner_user_id, id DESC);

CREATE INDEX IF NOT EXISTS idx_jobs_status
  ON jobs (status, id DESC);

CREATE INDEX IF NOT EXISTS idx_jobs_postcode
  ON jobs (postcode);

CREATE INDEX IF NOT EXISTS idx_bids_tradesperson
  ON bids (tradesperson_user_id, bids_timestamp DESC);
