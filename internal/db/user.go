package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"tradielynx/internal/models"
)

var (
	ErrUserNotFound   = errors.New("user not found")
	ErrBadPassword    = errors.New("bad password")
	ErrStatusNotFound = errors.New("tradesperson verification status not found")
)

type UserRepo struct{ DB *sql.DB } // data-access wrapper used by the auth layer.

// NormalizeEmail trims and lowercases before DB use.
// CITEXT makes comparisons case-insensitive, but normalizing helps UX consistency.
func NormalizeString(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

// InsertUser inserts a new user. On success it returns a null error
func InsertUser(ctx context.Context, db *sql.DB, user models.User) error {
	const q = `
		INSERT INTO users (first_name, last_name, email, password_hash, user_role, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id int64
	err := db.QueryRowContext(ctx, q,
		user.FirstName,
		user.LastName,
		NormalizeString(user.Email),
		user.HashedPassword,
		user.Role,
		user.IsActive,
	).Scan(&id)
	if err != nil {
		return err
	}
	return nil
}

// RetrieveUserByEmail loads the user row for login.
// It does NOT verify password—crypto is kept in the handler/service.
func RetrieveUserByEmail(ctx context.Context, db *sql.DB, email string) (models.User, error) {
	const q = `
		SELECT id, first_name, COALESCE(last_name,''), email, password_hash, user_role::text, active
		FROM users
		WHERE email = $1
		LIMIT 1
	`
	var (
		u       models.User
		roleStr string
	)
	err := db.QueryRowContext(ctx, q, NormalizeString(email)).
		Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.HashedPassword, &roleStr,
			&u.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, err
	}
	u.Role = models.Role(roleStr)
	return u, nil
}

// RetrieveUserByID returns a minimal user row for dashboard/etc.
func RetrieveUserByID(ctx context.Context, db *sql.DB, userID int64) (models.User, error) {
	const q = `
		SELECT id,
       		   first_name,
       		   COALESCE(last_name, ''),
       		   email,
       		   user_role::text,
       		   active
		FROM users
		WHERE id = $1
		LIMIT 1;
	`
	var (
		u       models.User
		roleStr string
	)
	err := db.QueryRowContext(ctx, q, userID).
		Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &roleStr, &u.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, err
	}
	u.Role = models.Role(roleStr)
	return u, nil
}

// LookupRole is used by middleware to refresh the role from DB.
func LookupRole(ctx context.Context, db *sql.DB, userID int64) (models.Role, error) {
	const q = `SELECT user_role::text FROM users WHERE id = $1 AND active = TRUE`
	var s string
	if err := db.QueryRowContext(ctx, q, userID).Scan(&s); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUserNotFound
		}
		return "", err
	}
	return models.Role(s), nil
}

// returns the user's role from the database, called by RequireAuth()
func (r *UserRepo) LookupUserRole(ctx context.Context, userID int64) (models.Role, error) {
	var s string
	// grabs user's role from DB, stores in 's'
	err := r.DB.QueryRowContext(ctx,
		`SELECT user_role::text FROM users WHERE id = $1`, userID,
	).Scan(&s)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUserNotFound
		}
		return "", err
	}

	switch models.Role(s) {
	case models.RoleHomeowner, models.RoleTradesperson, models.RoleAdmin:
		return models.Role(s), nil
	default:
		// Bad/unknown value in DB – treated as not found
		return "", ErrUserNotFound
	}
}

// Returns tradespeople who are still waiting for account verification.
func ListPendingTradespeople(ctx context.Context, db *sql.DB) ([]models.User, error) {
	const q = `
		SELECT id, first_name, COALESCE(last_name, ''), email, user_role::text, active
		FROM users
		WHERE user_role = $1 AND active = FALSE
		ORDER BY user_timestamps ASC
	`

	rows, err := db.QueryContext(ctx, q, models.RoleTradesperson)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]models.User, 0)

	for rows.Next() {
		var (
			u       models.User
			roleStr string
		)

		if err := rows.Scan(
			&u.ID,
			&u.FirstName,
			&u.LastName,
			&u.Email,
			&roleStr,
			&u.IsActive,
		); err != nil {
			return nil, err
		}

		u.Role = models.Role(roleStr)
		users = append(users, u)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// Activates a tradesperson after an admin approves the account.
func ActivateTradesperson(ctx context.Context, db *sql.DB, userID int64) error {
	result, err := db.ExecContext(ctx, `
		UPDATE users
		SET active = TRUE
		WHERE id = $1 AND user_role = $2 AND active = FALSE
	`, userID, models.RoleTradesperson)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}
