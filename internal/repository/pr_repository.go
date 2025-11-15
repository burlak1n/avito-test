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
	GetOpenPRsByAuthors(userIDs []string) ([]*models.PullRequest, error)
	GetOpenPRsByReviewers(userIDs []string) (map[string][]*models.PullRequest, error)
	ReassignAuthor(tx *sql.Tx, prID, newAuthorID string) error
	RemoveReviewer(tx *sql.Tx, prID, reviewerID string) error
	AddReviewer(tx *sql.Tx, prID, reviewerID string) error
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

func (r *pullRequestRepository) GetOpenPRsByAuthors(userIDs []string) ([]*models.PullRequest, error) {
	if len(userIDs) == 0 {
		return []*models.PullRequest{}, nil
	}

	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status,
		       COALESCE(array_agg(prr.reviewer_id) FILTER (WHERE prr.reviewer_id IS NOT NULL), '{}') as reviewers
		FROM pull_requests pr
		LEFT JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE pr.author_id = ANY($1) AND pr.status = 'OPEN'
		GROUP BY pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status`

	rows, err := r.db.Query(query, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prs := make([]*models.PullRequest, 0)
	for rows.Next() {
		pr := &models.PullRequest{}
		var reviewers []string
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &reviewers); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = reviewers
		prs = append(prs, pr)
	}
	return prs, rows.Err()
}

func (r *pullRequestRepository) GetOpenPRsByReviewers(userIDs []string) (map[string][]*models.PullRequest, error) {
	if len(userIDs) == 0 {
		return make(map[string][]*models.PullRequest), nil
	}

	query := `
		SELECT prr.reviewer_id, pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status,
		       COALESCE(array_agg(prr2.reviewer_id) FILTER (WHERE prr2.reviewer_id IS NOT NULL), '{}') as reviewers
		FROM pr_reviewers prr
		JOIN pull_requests pr ON prr.pull_request_id = pr.pull_request_id
		LEFT JOIN pr_reviewers prr2 ON pr.pull_request_id = prr2.pull_request_id
		WHERE prr.reviewer_id = ANY($1) AND pr.status = 'OPEN'
		GROUP BY prr.reviewer_id, pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status`

	rows, err := r.db.Query(query, userIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]*models.PullRequest)
	for rows.Next() {
		var reviewerID string
		pr := &models.PullRequest{}
		var reviewers []string
		if err := rows.Scan(&reviewerID, &pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status, &reviewers); err != nil {
			return nil, err
		}
		pr.AssignedReviewers = reviewers
		result[reviewerID] = append(result[reviewerID], pr)
	}
	return result, rows.Err()
}

func (r *pullRequestRepository) ReassignAuthor(tx *sql.Tx, prID, newAuthorID string) error {
	query := `UPDATE pull_requests SET author_id = $1 WHERE pull_request_id = $2`
	_, err := tx.Exec(query, newAuthorID, prID)
	return err
}

func (r *pullRequestRepository) RemoveReviewer(tx *sql.Tx, prID, reviewerID string) error {
	query := `DELETE FROM pr_reviewers WHERE pull_request_id = $1 AND reviewer_id = $2`
	_, err := tx.Exec(query, prID, reviewerID)
	return err
}

func (r *pullRequestRepository) AddReviewer(tx *sql.Tx, prID, reviewerID string) error {
	query := `INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`
	_, err := tx.Exec(query, prID, reviewerID)
	return err
}
