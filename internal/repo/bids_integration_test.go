//go:build integration

package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/lib/pq"

	"tradielynx/internal/models"
)

type awardFixture struct {
	ownerID, otherOwnerID int64
	jobID, firstBidID, secondBidID int64
	firstTradeID int64
}

func integrationDB(t *testing.T) *sql.DB {
	t.Helper()

	dsn := os.Getenv("TRADIELYNX_TEST_DATABASE_URL")
	if dsn == "" {
		t.Fatal("set TRADIELYNX_TEST_DATABASE_URL to a disposable PostgreSQL database")
	}

	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}

	schema := "tl_test_" + strconv.FormatInt(time.Now().UnixNano(), 36)
	if _, err := admin.Exec("CREATE SCHEMA " + pq.QuoteIdentifier(schema)); err != nil {
		admin.Close()
		t.Fatalf("create test schema: %v", err)
	}

	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema+",public")
	u.RawQuery = q.Encode()

	db, err := sql.Open("postgres", u.String())
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(5)

	t.Cleanup(func() {
		_ = db.Close()
		if _, err := admin.Exec("DROP SCHEMA " + pq.QuoteIdentifier(schema) + " CASCADE"); err != nil {
			t.Errorf("drop test schema: %v", err)
		}
		_ = admin.Close()
	})

	schemaSQL, err := os.ReadFile(filepath.Join("..", "db", "initial_db.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(string(schemaSQL)); err != nil {
		t.Fatalf("load application schema: %v", err)
	}
	return db
}

func insertedID(t *testing.T, db *sql.DB, query string, args ...any) int64 {
	t.Helper()
	var id int64
	if err := db.QueryRow(query, args...).Scan(&id); err != nil {
		t.Fatalf("insert fixture: %v", err)
	}
	return id
}

func testUser(t *testing.T, db *sql.DB, role, email string) int64 {
	t.Helper()
	return insertedID(t, db, `INSERT INTO users
		(first_name, email, password_hash, user_role, active)
		VALUES ('Test', $1, 'test-hash', $2, true) RETURNING id`, email, role)
}

func testJob(t *testing.T, db *sql.DB, owner int64) int64 {
	t.Helper()
	return insertedID(t, db, `INSERT INTO jobs
		(homeowner_user_id, title, description, postcode)
		VALUES ($1, 'Replace cabinet', 'Kitchen cabinet job', 'R3T 0A1')
		RETURNING id`, owner)
}

func testBid(t *testing.T, db *sql.DB, jobID, tradeID int64) int64 {
	t.Helper()
	return insertedID(t, db, `INSERT INTO bids
		(job_id, tradesperson_user_id, amount)
		VALUES ($1, $2, 150.00) RETURNING id`, jobID, tradeID)
}

func fixture(t *testing.T, db *sql.DB) awardFixture {
	t.Helper()
	owner := testUser(t, db, "Homeowner", "owner@example.test")
	other := testUser(t, db, "Homeowner", "other@example.test")
	trade1 := testUser(t, db, "Tradesperson", "trade1@example.test")
	trade2 := testUser(t, db, "Tradesperson", "trade2@example.test")
	job := testJob(t, db, owner)
	return awardFixture{ownerID: owner, otherOwnerID: other,
		jobID: job, firstBidID: testBid(t, db, job, trade1),
		secondBidID: testBid(t, db, job, trade2), firstTradeID: trade1}
}

func status(t *testing.T, db *sql.DB, table string, id int64) string {
	t.Helper()
	if table != "jobs" && table != "bids" {
		t.Fatal("unsupported status table")
	}
	var got string
	if err := db.QueryRow(fmt.Sprintf("SELECT status::text FROM %s WHERE id = $1", table), id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

func expectState(t *testing.T, db *sql.DB, f awardFixture, job, first, second string) {
	t.Helper()
	for _, check := range []struct {
		table string
		id int64
		want string
	}{
		{"jobs", f.jobID, job},
		{"bids", f.firstBidID, first},
		{"bids", f.secondBidID, second},
	} {
		got := status(t, db, check.table, check.id)
		if got != check.want {
			t.Errorf("%s id=%d: got %q, want %q", check.table, check.id, got, check.want)
		}
	}
}

func TestAwardBidIntegration(t *testing.T) {
	ctx := context.Background()

	t.Run("awards one bid and rejects the other", func(t *testing.T) {
		db := integrationDB(t)
		f := fixture(t, db)
		if err := AwardBid(ctx, db, f.ownerID, f.jobID, f.firstBidID); err != nil {
			t.Fatal(err)
		}
		expectState(t, db, f, "Awarded", "Accepted", "Rejected")
	})

	t.Run("wrong homeowner cannot award a bid", func(t *testing.T) {
		db := integrationDB(t)
		f := fixture(t, db)
		err := AwardBid(ctx, db, f.otherOwnerID, f.jobID, f.firstBidID)
		if !errors.Is(err, ErrJobNotFound) {
			t.Fatalf("got %v, want ErrJobNotFound", err)
		}
		expectState(t, db, f, "Open", "Submitted", "Submitted")
	})

	t.Run("cannot award a bid from a different job", func(t *testing.T) {
		db := integrationDB(t)
		f := fixture(t, db)
		differentJob := testJob(t, db, f.ownerID)
		differentBid := testBid(t, db, differentJob, f.firstTradeID)
		err := AwardBid(ctx, db, f.ownerID, f.jobID, differentBid)
		if !errors.Is(err, ErrBidNotFound) {
			t.Fatalf("got %v, want ErrBidNotFound", err)
		}
		expectState(t, db, f, "Open", "Submitted", "Submitted")
	})

	t.Run("cannot re-award a closed job", func(t *testing.T) {
		db := integrationDB(t)
		f := fixture(t, db)
		if err := AwardBid(ctx, db, f.ownerID, f.jobID, f.firstBidID); err != nil {
			t.Fatal(err)
		}
		err := AwardBid(ctx, db, f.ownerID, f.jobID, f.secondBidID)
		if !errors.Is(err, ErrJobNotOpen) {
			t.Fatalf("got %v, want ErrJobNotOpen", err)
		}
		expectState(t, db, f, "Awarded", "Accepted", "Rejected")
	})

	t.Run("rolls back every change on a job update failure", func(t *testing.T) {
		db := integrationDB(t)
		f := fixture(t, db)
		_, err := db.Exec(`CREATE FUNCTION prevent_award() RETURNS trigger
			LANGUAGE plpgsql AS $$
			BEGIN
				RAISE EXCEPTION 'simulated write failure';
			END;
			$$;
			CREATE TRIGGER fail_award BEFORE UPDATE ON jobs
			FOR EACH ROW EXECUTE FUNCTION prevent_award();`)
		if err != nil {
			t.Fatal(err)
		}
		if err := AwardBid(ctx, db, f.ownerID, f.jobID, f.firstBidID); err == nil {
			t.Fatal("expected a database error")
		}
		expectState(t, db, f, "Open", "Submitted", "Submitted")
	})

	t.Run("concurrent awards cannot produce two winners", func(t *testing.T) {
		db := integrationDB(t)
		f := fixture(t, db)
		begin := make(chan struct{})
		result := make(chan error, 2)
		for _, bidID := range []int64{f.firstBidID, f.secondBidID} {
			go func(bidID int64) {
				<-begin
				result <- AwardBid(ctx, db, f.ownerID, f.jobID, bidID)
			}(bidID)
		}
		close(begin)

		var successes, alreadyClosed int
		for i := 0; i < 2; i++ {
			switch err := <-result; {
			case err == nil:
				successes++
			case errors.Is(err, ErrJobNotOpen):
				alreadyClosed++
			default:
				t.Fatalf("unexpected concurrent award result: %v", err)
			}
		}
		if successes != 1 || alreadyClosed != 1 {
			t.Fatalf("got %d successes and %d already-closed responses; expected one each",
				successes, alreadyClosed)
		}

		first := status(t, db, "bids", f.firstBidID)
		second := status(t, db, "bids", f.secondBidID)
		if status(t, db, "jobs", f.jobID) != string(models.JobAwarded) ||
			!((first == "Accepted" && second == "Rejected") ||
				(first == "Rejected" && second == "Accepted")) {
			t.Fatalf("inconsistent final state: first=%s, second=%s", first, second)
		}
	})
}
