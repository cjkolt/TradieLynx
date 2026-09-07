package db

import (
	"context"
	"database/sql"

	"tradielynx/internal/models"
)

func InsertJob(ctx context.Context, db *sql.DB, job models.Job) error {
	const q = `
		INSERT INTO jobs (homeowner_user_id, title, description, postcode, preferred_timeframe, budget_min_max, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var id int64
	err := db.QueryRowContext(ctx, q,
		job.HomeownerID,
		job.Title,
		job.Description,
		job.PostCode,
		job.PreferredTimeframe,
		job.BudgetMinMax,
		job.Status,
	).Scan(&id)
	if err != nil {
		return err
	}

	return nil
}

func InsertBid(ctx context.Context, db *sql.DB, bid models.Bid) (int64, error) {
	const q = `
		INSERT INTO bids (job_id, tradesperson_user_id, amount, message, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id int64
	err := db.QueryRowContext(ctx, q,
		bid.JobID,
		bid.TradespersonID,
		bid.Amount,
		bid.Message,
		bid.Status,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func RetrieveBidByID(ctx context.Context, dbConn *sql.DB, bidID int64) (models.Bid, error) {
	const q = `
		SELECT id, job_id, tradesperson_user_id, amount, COALESCE(message, ''), status::text, bids_timestamp
		FROM bids
		WHERE id = $1
	`
	var (
		b       models.Bid
		statusS string
	)
	err := dbConn.QueryRowContext(ctx, q, bidID).Scan(
		&b.ID,
		&b.JobID,
		&b.TradespersonID,
		&b.Amount,
		&b.Message,
		&statusS,
		&b.CreatedAt,
	)
	if err != nil {
		return models.Bid{}, err
	}
	b.Status = models.BidStatus(statusS)
	return b, nil
}
