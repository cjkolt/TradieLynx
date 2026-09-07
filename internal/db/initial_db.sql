\connect tradielynx;

-- extensions
CREATE EXTENSION IF NOT EXISTS citext;

-- enums (DROP + plain CREATE)
DROP TYPE IF EXISTS user_role_enum;
CREATE TYPE user_role_enum AS ENUM ('Homeowner','Tradesperson','Administrator');

DROP TYPE IF EXISTS verification_status_enum;
CREATE TYPE verification_status_enum AS ENUM ('Pending','Verified','Rejected','Pending - More Info Requested');

DROP TYPE IF EXISTS status_enum;
CREATE TYPE status_enum AS ENUM ('Open','Awarded','Completed','Cancelled');

DROP TYPE IF EXISTS bids_status_enum;
CREATE TYPE bids_status_enum AS ENUM ('Submitted','Accepted','Rejected');

DROP TYPE IF EXISTS transaction_type_enum;
CREATE TYPE transaction_type_enum AS ENUM ('InitialGrant','BidPlacement','AdminAdjustment');

-- tables
CREATE TABLE IF NOT EXISTS users (
  id               BIGSERIAL PRIMARY KEY,
  first_name       VARCHAR(32) NOT NULL,
  last_name        VARCHAR(32),
  email            CITEXT NOT NULL UNIQUE,
  password_hash    TEXT NOT NULL,
  user_role        user_role_enum NOT NULL,
  active           BOOLEAN NOT NULL DEFAULT FALSE, -- requires verification if tradesperson
  user_timestamps  TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS homeowner_profiles (
  user_id   BIGINT UNIQUE REFERENCES users(id),
  postcode  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tradesperson_profiles (
  user_id                     BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  business_name               TEXT,
  phone                       TEXT UNIQUE NOT NULL,
  skills                      TEXT,
  service_area                TEXT NOT NULL,
  verification_status         verification_status_enum NOT NULL DEFAULT 'Pending',
  token_balance               NUMERIC(10,0),
  qualification_document_url  TEXT NOT NULL UNIQUE,
  reference1_name             TEXT NOT NULL,
  reference1_phone            TEXT NOT NULL,
  reference1_relationship     TEXT NOT NULL,
  reference2_name             TEXT NOT NULL,
  reference2_phone            TEXT NOT NULL,
  reference2_relationship     TEXT NOT NULL,
  tp_profile_timestamps       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS jobs (
  id                  BIGSERIAL PRIMARY KEY,
  homeowner_user_id   BIGINT REFERENCES users(id),
  title               TEXT NOT NULL,
  description         TEXT NOT NULL,
  postcode            TEXT NOT NULL,
  preferred_timeframe TEXT,
  budget_min_max      TEXT,
  status              status_enum NOT NULL DEFAULT 'Open'
);

CREATE TABLE IF NOT EXISTS bids (
  id                   BIGSERIAL PRIMARY KEY,
  job_id               BIGINT REFERENCES jobs(id) ON DELETE CASCADE,
  tradesperson_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
  amount               NUMERIC(10,2) NOT NULL,
  message              TEXT,
  status               bids_status_enum NOT NULL,
  bids_timestamp       TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE (job_id, tradesperson_user_id)
);

CREATE TABLE IF NOT EXISTS admins (
  id    BIGSERIAL PRIMARY KEY,
  name  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS token_ledger (
  id                   BIGSERIAL PRIMARY KEY,
  tradesperson_user_id INTEGER UNIQUE REFERENCES tradesperson_profiles(user_id),
  change_amount        NUMERIC(10,2) NOT NULL,
  transaction_type     transaction_type_enum NOT NULL,
  related_job_id       INTEGER UNIQUE REFERENCES jobs(id),
  admin_user_id        INTEGER UNIQUE REFERENCES admins(id),
  token_ledger_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
  id                 BIGSERIAL PRIMARY KEY,
  related_job_id     INTEGER UNIQUE REFERENCES jobs(id),
  sender_user_id     INTEGER REFERENCES users(id),
  receiver_user_id   INTEGER REFERENCES users(id),
  content            TEXT,
  messages_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  read_status        TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ratings (
  id                    BIGSERIAL PRIMARY KEY,
  job_id                INTEGER UNIQUE REFERENCES jobs(id),
  reviewer_user_id      INTEGER UNIQUE REFERENCES homeowner_profiles(user_id),
  tradesperson_user_id  INTEGER UNIQUE REFERENCES tradesperson_profiles(user_id),
  inferred_score        INTEGER NOT NULL CHECK (inferred_score BETWEEN 1 AND 5),
  public_comment        TEXT,
  private_comment       TEXT,
  is_flagged            BOOLEAN NOT NULL DEFAULT FALSE,
  ratings_timestamp     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS review_questions (
  id          BIGSERIAL PRIMARY KEY,
  question_text TEXT NOT NULL,
  is_active   BOOLEAN NOT NULL
);

CREATE TABLE IF NOT EXISTS review_answers (
  id          BIGSERIAL PRIMARY KEY,
  rating_id   INTEGER UNIQUE REFERENCES ratings(id),
  question_id INTEGER UNIQUE REFERENCES review_questions(id),
  answer      BOOLEAN NOT NULL
);
