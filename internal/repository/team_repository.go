package repository

import (
	"database/sql"

	"github.com/reviewer-service/internal/models"
)

type TeamRepository interface {
	Create(team *models.Team) error
	GetByName(teamName string) (*models.Team, error)
}

type teamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) TeamRepository {
	return &teamRepository{db: db}
}

func (r *teamRepository) Create(team *models.Team) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `INSERT INTO teams (team_name) VALUES ($1)`
	_, err = tx.Exec(query, team.TeamName)
	if err != nil {
		return err
	}

	if len(team.Members) > 0 {
		stmt, err := tx.Prepare(`INSERT INTO users (user_id, username, team_name, is_active) VALUES ($1, $2, $3, $4)`)
		if err != nil {
			return err
		}
		defer stmt.Close()

		for _, member := range team.Members {
			_, err = stmt.Exec(member.UserID, member.Username, team.TeamName, member.IsActive)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func (r *teamRepository) GetByName(teamName string) (*models.Team, error) {
	team := &models.Team{
		TeamName: teamName,
		Members:  []models.TeamMember{},
	}

	var exists bool
	err := r.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM teams WHERE team_name = $1)`, teamName).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, sql.ErrNoRows
	}

	query := `SELECT user_id, username, is_active FROM users WHERE team_name = $1 ORDER BY user_id`
	rows, err := r.db.Query(query, teamName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var member models.TeamMember
		if err := rows.Scan(&member.UserID, &member.Username, &member.IsActive); err != nil {
			return nil, err
		}
		team.Members = append(team.Members, member)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return team, nil
}
