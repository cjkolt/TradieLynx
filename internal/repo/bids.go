package repo

import (
	"context"
	"database/sql"
	"errors"

	"tradielynx/internal/models"
)

var (
	ErrJobNotOpen  = errors.New("job is not open")
	ErrBidNotFound = errors.New("bid not found")
)

func ListBidsForJob(ctx context.Context, db *sql.DB, jobID, ownerID int64) ([]models.Bid, error) {
	const q = `
		SELECT
			b.id,
			b.job_id,
			b.tradesperson_user_id,
			TRIM(u.first_name || ' ' || COALESCE(u.last_name, '')),
			b.amount,
			COALESCE(b.message, ''),
			b.status::text,
			b.bids_timestamp
		FROM bids b
		JOIN jobs j ON j.id = b.job_id
		JOIN users u ON u.id = b.tradesperson_user_id
		WHERE b.job_id = $1
		  AND j.homeowner_user_id = $2
		ORDER BY b.bids_timestamp ASC
	`

	rows, err := db.QueryContext(ctx, q, jobID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bids := make([]models.Bid, 0)

	for rows.Next() {
		var (
			bid       models.Bid
			statusStr string
		)

		if err := rows.Scan(
			&bid.ID,
			&bid.JobID,
			&bid.TradespersonID,
			&bid.TradespersonName,
			&bid.Amount,
			&bid.Message,
			&statusStr,
			&bid.CreatedAt,
		); err != nil {
			return nil, err
		}

		bid.Status = models.BidStatus(statusStr)
		bids = append(bids, bid)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bids, nil
}

func AwardBid(ctx context.Context, db *sql.DB, ownerID, jobID, bidID int64) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var jobStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status::text
		FROM jobs
		WHERE id = $1 AND homeowner_user_id = $2
		FOR UPDATE
	`, jobID, ownerID).Scan(&jobStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrJobNotFound
		}
		return err
	}
	if models.JobStatus(jobStatus) != models.JobOpen {
		return ErrJobNotOpen
	}

	var bidStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT status::text
		FROM bids
		WHERE id = $1 AND job_id = $2
		FOR UPDATE
	`, bidID, jobID).Scan(&bidStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrBidNotFound
		}
		return err
	}
	if models.BidStatus(bidStatus) != models.BidSubmitted {
		return ErrBidNotFound
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE bids
		SET status = $1
		WHERE id = $2
	`, models.BidAccepted, bidID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE bids
		SET status = $1
		WHERE job_id = $2
		  AND id <> $3
		  AND status = $4
	`, models.BidRejected, jobID, bidID, models.BidSubmitted); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET status = $1
		WHERE id = $2
	`, models.JobAwarded, jobID); err != nil {
		return err
	}

	return tx.Commit()
}
