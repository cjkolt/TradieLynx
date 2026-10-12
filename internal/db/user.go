package db

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"tradielynx/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepo struct{ DB *sql.DB }

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func InsertUser(ctx context.Context, db *sql.DB, user models.User) error {
	const q = `
		INSERT INTO users (first_name, last_name, email, password_hash, user_role, active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := db.ExecContext(ctx, q,
		user.FirstName,
		user.LastName,
		NormalizeEmail(user.Email),
		user.HashedPassword,
		user.Role,
		user.IsActive,
	)
	return err
}

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

	err := db.QueryRowContext(ctx, q, NormalizeEmail(email)).
		Scan(&u.ID, &u.FirstName, &u.LastName, &u.Email, &u.HashedPassword, &roleStr, &u.IsActive)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, ErrUserNotFound
		}
		return models.User{}, err
	}

	u.Role = models.Role(roleStr)
	return u, nil
}

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

func (r *UserRepo) LookupUserAuth(ctx context.Context, userID int64) (models.Role, bool, error) {
	var (
		roleStr string
		active  bool
	)

	err := r.DB.QueryRowContext(ctx,
		`SELECT user_role::text, active FROM users WHERE id = $1`, userID,
	).Scan(&roleStr, &active)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", false, ErrUserNotFound
		}
		return "", false, err
	}

	role, ok := models.ParseRole(roleStr)
	if !ok {
		return "", false, ErrUserNotFound
	}

	return role, active, nil
}

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
