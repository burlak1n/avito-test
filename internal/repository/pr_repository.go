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
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var createdAt interface{}
	if pr.CreatedAt != nil {
		createdAt = pr.CreatedAt
	}

	query := `INSERT INTO pull_requests (pull_request_id, pull_request_name, author_id, status, created_at) VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.Exec(query, pr.PullRequestID, pr.PullRequestName, pr.AuthorID, pr.Status, createdAt)
	if err != nil {
		return err
	}

	if len(pr.AssignedReviewers) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, reviewerID := range pr.AssignedReviewers {
			_, err = stmt.Exec(pr.PullRequestID, reviewerID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *pullRequestRepository) GetByID(prID string) (*models.PullRequest, error) {
	query := `
		SELECT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at,
		       COALESCE(array_agg(prr.reviewer_id) FILTER (WHERE prr.reviewer_id IS NOT NULL), '{}') as reviewers
		FROM pull_requests pr
		LEFT JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE pr.pull_request_id = $1
		GROUP BY pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status, pr.created_at, pr.merged_at`

	var pr models.PullRequest
	var reviewers []string
	var createdAt, mergedAt sql.NullTime

	err := r.db.QueryRow(query, prID).Scan(
		&pr.PullRequestID,
		&pr.PullRequestName,
		&pr.AuthorID,
		&pr.Status,
		&createdAt,
		&mergedAt,
		&reviewers,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, err
	}

	if createdAt.Valid {
		pr.CreatedAt = &createdAt.Time
	}
	if mergedAt.Valid {
		pr.MergedAt = &mergedAt.Time
	}
	pr.AssignedReviewers = reviewers

	return &pr, nil
}

func (r *pullRequestRepository) UpdateStatus(prID string, status string) error {
	var query string
	if status == "MERGED" {
		query = `UPDATE pull_requests SET status = $1, merged_at = CURRENT_TIMESTAMP WHERE pull_request_id = $2`
	} else {
		query = `UPDATE pull_requests SET status = $1 WHERE pull_request_id = $2`
	}
	_, err := r.db.Exec(query, status, prID)
	return err
}

func (r *pullRequestRepository) UpdateReviewers(prID string, reviewers []string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM pr_reviewers WHERE pull_request_id = $1`, prID)
	if err != nil {
		return err
	}

	if len(reviewers) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO pr_reviewers (pull_request_id, reviewer_id) VALUES ($1, $2)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, reviewerID := range reviewers {
			_, err = stmt.Exec(prID, reviewerID)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *pullRequestRepository) GetByReviewerID(userID string) ([]*models.PullRequestShort, error) {
	query := `
		SELECT DISTINCT pr.pull_request_id, pr.pull_request_name, pr.author_id, pr.status
		FROM pull_requests pr
		JOIN pr_reviewers prr ON pr.pull_request_id = prr.pull_request_id
		WHERE prr.reviewer_id = $1
		ORDER BY pr.pull_request_id`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prs := make([]*models.PullRequestShort, 0)
	for rows.Next() {
		var pr models.PullRequestShort
		if err := rows.Scan(&pr.PullRequestID, &pr.PullRequestName, &pr.AuthorID, &pr.Status); err != nil {
			return nil, err
		}
		prs = append(prs, &pr)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return prs, nil
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
