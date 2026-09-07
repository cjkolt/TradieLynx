package repo

import (
	"context"
	"database/sql"
	"errors"

	"tradielynx/internal/models"
)

var ErrJobNotFound = errors.New("job not found")

// ListOpenJobs returns open jobs for contractor browsing.
// Uses LIMIT/OFFSET for pagination. If you don't want pagination yet,
// pass a sane limit (e.g. 50) and offset 0.
func ListOpenJobs(ctx context.Context, db *sql.DB, limit, offset int) ([]models.Job, error) {
	// Safety defaults to avoid accidental "SELECT * from jobs forever"
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	const q = `
		SELECT
			id,
			homeowner_user_id,
			title,
			description,
			postcode,
			preferred_timeframe,
			budget_min_max,
			status::text
		FROM jobs
		WHERE status = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := db.QueryContext(ctx, q, models.JobOpen, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0, limit)

	for rows.Next() {
		var (
			j       models.Job
			tf      sql.NullString
			budget  sql.NullString
			statusS string
		)

		if err := rows.Scan(
			&j.ID,
			&j.HomeownerID,
			&j.Title,
			&j.Description,
			&j.PostCode,
			&tf,
			&budget,
			&statusS,
		); err != nil {
			return nil, err
		}

		if tf.Valid {
			j.PreferredTimeframe = tf.String
		}
		if budget.Valid {
			j.BudgetMinMax = budget.String
		}
		j.Status = models.JobStatus(statusS)

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

// GetJobByID returns a single job by ID.
// Useful for /jobs/{id} detail pages.
func GetJobByID(ctx context.Context, db *sql.DB, id int64) (models.Job, error) {
	const q = `
		SELECT
			id,
			homeowner_user_id,
			title,
			description,
			postcode,
			preferred_timeframe,
			budget_min_max,
			status::text
		FROM jobs
		WHERE id = $1
		LIMIT 1
	`

	var (
		j       models.Job
		tf      sql.NullString
		budget  sql.NullString
		statusS string
	)

	err := db.QueryRowContext(ctx, q, id).Scan(&j.ID,
		&j.HomeownerID,
		&j.Title,
		&j.Description,
		&j.PostCode,
		&tf,
		&budget,
		&statusS,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Job{}, ErrJobNotFound
		}
		return models.Job{}, err
	}

	if tf.Valid {
		j.PreferredTimeframe = tf.String
	}
	if budget.Valid {
		j.BudgetMinMax = budget.String
	}
	j.Status = models.JobStatus(statusS)

	return j, nil
}

// GetJobForOwner is optional but recommended.
// Prevents homeowners from reading jobs they don't own.
// Useful for edit/delete views later.
func GetJobForOwner(ctx context.Context, db *sql.DB, id, ownerID int64) (models.Job, error) {
	const q = `
		SELECT
			id,
			homeowner_user_id,
			title,
			description,
			postcode,
			preferred_timeframe,
			budget_min_max,
			status::text
		FROM jobs
		WHERE id = $1 AND homeowner_user_id = $2
		LIMIT 1
	`

	var (
		j       models.Job
		tf      sql.NullString
		budget  sql.NullString
		statusS string
	)

	err := db.QueryRowContext(ctx, q, id, ownerID).Scan(
		&j.ID,
		&j.HomeownerID,
		&j.Title,
		&j.Description,
		&j.PostCode,
		&tf,
		&budget,
		&statusS,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.Job{}, ErrJobNotFound
		}
		return models.Job{}, err
	}

	if tf.Valid {
		j.PreferredTimeframe = tf.String
	}
	if budget.Valid {
		j.BudgetMinMax = budget.String
	}
	j.Status = models.JobStatus(statusS)

	return j, nil
}

// ListJobsByOwner returns jobs posted by one homeowner.
// Used by the My Jobs page.
func ListJobsByOwner(ctx context.Context, db *sql.DB, ownerID int64) ([]models.Job, error) {
	const q = `
		SELECT
			id,
			homeowner_user_id,
			title,
			description,
			postcode,
			preferred_timeframe,
			budget_min_max,
			status::text
		FROM jobs
		WHERE homeowner_user_id = $1
		ORDER BY id DESC
	`

	rows, err := db.QueryContext(ctx, q, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0)

	for rows.Next() {
		var (
			j       models.Job
			tf      sql.NullString
			budget  sql.NullString
			statusS string
		)

		if err := rows.Scan(
			&j.ID,
			&j.HomeownerID,
			&j.Title,
			&j.Description,
			&j.PostCode,
			&tf,
			&budget,
			&statusS,
		); err != nil {
			return nil, err
		}

		if tf.Valid {
			j.PreferredTimeframe = tf.String
		}
		if budget.Valid {
			j.BudgetMinMax = budget.String
		}
		j.Status = models.JobStatus(statusS)

		jobs = append(jobs, j)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}

// ListBidsByTradesperson returns bids submitted by one tradesperson.
// The job title is joined in so My Bids can link back to the job.
func ListBidsByTradesperson(ctx context.Context, db *sql.DB, tradespersonID int64) ([]models.Bid, error) {
	const q = `
		SELECT
			b.id,
			b.job_id,
			b.tradesperson_user_id,
			j.title,
			b.amount,
			COALESCE(b.message, ''),
			b.status::text,
			b.bids_timestamp
		FROM bids b
		JOIN jobs j ON j.id = b.job_id
		WHERE b.tradesperson_user_id = $1
		ORDER BY b.bids_timestamp DESC
	`

	rows, err := db.QueryContext(ctx, q, tradespersonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	bids := make([]models.Bid, 0)

	for rows.Next() {
		var (
			b       models.Bid
			statusS string
		)

		if err := rows.Scan(
			&b.ID,
			&b.JobID,
			&b.TradespersonID,
			&b.JobTitle,
			&b.Amount,
			&b.Message,
			&statusS,
			&b.CreatedAt,
		); err != nil {
			return nil, err
		}

		b.Status = models.BidStatus(statusS)
		bids = append(bids, b)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return bids, nil
}
