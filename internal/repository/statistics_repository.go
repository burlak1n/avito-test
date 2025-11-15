package repository

import (
	"database/sql"

	"github.com/reviewer-service/internal/models"
)

type StatisticsRepository interface {
	GetStatistics() (*models.Statistics, error)
}

type statisticsRepository struct {
	db *sql.DB
}

func NewStatisticsRepository(db *sql.DB) StatisticsRepository {
	return &statisticsRepository{db: db}
}

func (r *statisticsRepository) GetStatistics() (*models.Statistics, error) {
	stats := &models.Statistics{}

	// Count teams
	var teamsCount int
	err := r.db.QueryRow("SELECT COUNT(*) FROM teams").Scan(&teamsCount)
	if err != nil {
		return nil, err
	}
	stats.Teams.Total = teamsCount

	// Count users
	var usersTotal, usersActive, usersInactive int
	err = r.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&usersTotal)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow("SELECT COUNT(*) FROM users WHERE is_active = true").Scan(&usersActive)
	if err != nil {
		return nil, err
	}
	usersInactive = usersTotal - usersActive
	stats.Users.Total = usersTotal
	stats.Users.Active = usersActive
	stats.Users.Inactive = usersInactive

	// Count pull requests
	var prTotal, prOpen, prMerged int
	err = r.db.QueryRow("SELECT COUNT(*) FROM pull_requests").Scan(&prTotal)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE status = 'OPEN'").Scan(&prOpen)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow("SELECT COUNT(*) FROM pull_requests WHERE status = 'MERGED'").Scan(&prMerged)
	if err != nil {
		return nil, err
	}
	stats.PullRequests.Total = prTotal
	stats.PullRequests.Open = prOpen
	stats.PullRequests.Merged = prMerged

	// Count review assignments
	var assignmentsTotal int
	err = r.db.QueryRow("SELECT COUNT(*) FROM pr_reviewers").Scan(&assignmentsTotal)
	if err != nil {
		return nil, err
	}
	stats.ReviewAssignments.Total = assignmentsTotal

	// Get assignments by reviewer
	rows, err := r.db.Query(`
		SELECT reviewer_id, COUNT(*) as count
		FROM pr_reviewers
		GROUP BY reviewer_id
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var byReviewer []models.ReviewerAssignment
	for rows.Next() {
		var assignment models.ReviewerAssignment
		if err := rows.Scan(&assignment.UserID, &assignment.Count); err != nil {
			return nil, err
		}
		byReviewer = append(byReviewer, assignment)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	stats.ReviewAssignments.ByReviewer = byReviewer

	return stats, nil
}



