package repo

import (
	"context"
	"database/sql"

	"tradielynx/internal/models"
)

func ListOpenJobsFiltered(ctx context.Context, db *sql.DB, filter models.JobFilter, limit, offset int) ([]models.Job, error) {
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
		  AND (
			$2 = ''
			OR title ILIKE '%' || $2 || '%'
			OR description ILIKE '%' || $2 || '%'
		  )
		  AND ($3 = '' OR REPLACE(postcode, ' ', '') ILIKE REPLACE($3, ' ', '') || '%')
		ORDER BY id DESC
		LIMIT $4 OFFSET $5
	`

	rows, err := db.QueryContext(ctx, q,
		models.JobOpen,
		filter.Query,
		filter.PostCode,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]models.Job, 0, limit)

	for rows.Next() {
		var (
			job       models.Job
			timeframe sql.NullString
			budget    sql.NullString
			statusStr string
		)

		if err := rows.Scan(
			&job.ID,
			&job.HomeownerID,
			&job.Title,
			&job.Description,
			&job.PostCode,
			&timeframe,
			&budget,
			&statusStr,
		); err != nil {
			return nil, err
		}

		if timeframe.Valid {
			job.PreferredTimeframe = timeframe.String
		}
		if budget.Valid {
			job.BudgetMinMax = budget.String
		}
		job.Status = models.JobStatus(statusStr)

		jobs = append(jobs, job)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return jobs, nil
}
