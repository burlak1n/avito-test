package repository

import (
	"database/sql"

	"github.com/reviewer-service/internal/models"
)

type UserRepository interface {
	GetByID(userID string) (*models.User, error)
	UpdateActivity(userID string, isActive bool) (*models.User, error)
	GetActiveTeamMembers(teamName string, excludeUserID string) ([]*models.User, error)
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
