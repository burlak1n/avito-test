package storage

import (
	"database/sql"

	"github.com/reviewer-service/internal/models"
)

type Storage interface {
	CreateTeam(team *models.Team) error
	GetTeam(teamName string) (*models.Team, error)
	UpdateUserActivity(userID string, isActive bool) (*models.User, error)
	GetUser(userID string) (*models.User, error)
	CreatePR(pr *models.PullRequest) error
	GetPR(prID string) (*models.PullRequest, error)
	UpdatePRStatus(prID string, status string) error
	UpdatePRReviewers(prID string, reviewers []string) error
	GetUserReviews(userID string) ([]*models.PullRequestShort, error)
	GetTeamActiveMembers(teamName string, excludeUserID string) ([]*models.User, error)
}

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(db *sql.DB) *PostgresStorage {
	return &PostgresStorage{db: db}
}

func (s *PostgresStorage) CreateTeam(team *models.Team) error {
	// TODO: implement
	return nil
}

func (s *PostgresStorage) GetTeam(teamName string) (*models.Team, error) {
	// TODO: implement
	return nil, nil
}

func (s *PostgresStorage) UpdateUserActivity(userID string, isActive bool) (*models.User, error) {
	// TODO: implement
	return nil, nil
}

func (s *PostgresStorage) GetUser(userID string) (*models.User, error) {
	// TODO: implement
	return nil, nil
}

func (s *PostgresStorage) CreatePR(pr *models.PullRequest) error {
	// TODO: implement
	return nil
}

func (s *PostgresStorage) GetPR(prID string) (*models.PullRequest, error) {
	// TODO: implement
	return nil, nil
}

func (s *PostgresStorage) UpdatePRStatus(prID string, status string) error {
	// TODO: implement
	return nil
}

func (s *PostgresStorage) UpdatePRReviewers(prID string, reviewers []string) error {
	// TODO: implement
	return nil
}

func (s *PostgresStorage) GetUserReviews(userID string) ([]*models.PullRequestShort, error) {
	// TODO: implement
	return nil, nil
}

func (s *PostgresStorage) GetTeamActiveMembers(teamName string, excludeUserID string) ([]*models.User, error) {
	// TODO: implement
	return nil, nil
}
