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
	// TODO: implement
	return nil, nil
}

func (r *userRepository) UpdateActivity(userID string, isActive bool) (*models.User, error) {
	// TODO: implement
	return nil, nil
}

func (r *userRepository) GetActiveTeamMembers(teamName string, excludeUserID string) ([]*models.User, error) {
	// TODO: implement
	return nil, nil
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
