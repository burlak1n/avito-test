package repository

import (
	"database/sql"

	"github.com/reviewer-service/internal/models"
)

type UserRepository interface {
	GetByID(userID string) (*models.User, error)
	UpdateActivity(userID string, isActive bool) (*models.User, error)
	GetActiveTeamMembers(teamName string, excludeUserID string) ([]*models.User, error)
	DeactivateUsers(tx *sql.Tx, userIDs []string) error
	GetUsersByIDs(userIDs []string) ([]*models.User, error)
}

type userRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) GetByID(userID string) (*models.User, error) {
	query := `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = $1`
	var user models.User
	err := r.db.QueryRow(query, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) UpdateActivity(userID string, isActive bool) (*models.User, error) {
	query := `UPDATE users SET is_active = $1, updated_at = CURRENT_TIMESTAMP WHERE user_id = $2 RETURNING user_id, username, team_name, is_active`
	var user models.User
	err := r.db.QueryRow(query, isActive, userID).Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetActiveTeamMembers(teamName string, excludeUserID string) ([]*models.User, error) {
	var query string
	var args []interface{}

	if excludeUserID != "" {
		query = `SELECT user_id, username, team_name, is_active FROM users WHERE team_name = $1 AND is_active = true AND user_id != $2 ORDER BY user_id`
		args = []interface{}{teamName, excludeUserID}
	} else {
		query = `SELECT user_id, username, team_name, is_active FROM users WHERE team_name = $1 AND is_active = true ORDER BY user_id`
		args = []interface{}{teamName}
	}

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*models.User, 0)
	for rows.Next() {
		var user models.User
		if err := rows.Scan(&user.UserID, &user.Username, &user.TeamName, &user.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *userRepository) DeactivateUsers(tx *sql.Tx, userIDs []string) error {
	if len(userIDs) == 0 {
		return nil
	}

	query := `UPDATE users SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE user_id = ANY($1)`
	_, err := tx.Exec(query, userIDs)
	return err
}

func (r *userRepository) GetUsersByIDs(userIDs []string) ([]*models.User, error) {
	if len(userIDs) == 0 {
		return []*models.User{}, nil
	}

	query := `SELECT user_id, username, team_name, is_active FROM users WHERE user_id = ANY($1)`
	rows, err := r.db.Query(query, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*models.User, 0)
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.UserID, &u.Username, &u.TeamName, &u.IsActive); err != nil {
			return nil, err
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}
