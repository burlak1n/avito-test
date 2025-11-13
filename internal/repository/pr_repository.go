package repository

import (
	"database/sql"

	"github.com/reviewer-service/internal/models"
)

type PullRequestRepository interface {
	Create(pr *models.PullRequest) error
	GetByID(prID string) (*models.PullRequest, error)
	UpdateStatus(prID string, status string) error
	UpdateReviewers(prID string, reviewers []string) error
	GetByReviewerID(userID string) ([]*models.PullRequestShort, error)
}

type pullRequestRepository struct {
	db *sql.DB
}

func NewPullRequestRepository(db *sql.DB) PullRequestRepository {
	return &pullRequestRepository{db: db}
}

func (r *pullRequestRepository) Create(pr *models.PullRequest) error {
	// TODO: implement
	return nil
}

func (r *pullRequestRepository) GetByID(prID string) (*models.PullRequest, error) {
	// TODO: implement
	return nil, nil
}

func (r *pullRequestRepository) UpdateStatus(prID string, status string) error {
	// TODO: implement
	return nil
}

func (r *pullRequestRepository) UpdateReviewers(prID string, reviewers []string) error {
	// TODO: implement
	return nil
}

func (r *pullRequestRepository) GetByReviewerID(userID string) ([]*models.PullRequestShort, error) {
	// TODO: implement
	return nil, nil
}
