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
	// TODO: implement
	return nil
}

func (r *teamRepository) GetByName(teamName string) (*models.Team, error) {
	// TODO: implement
	return nil, nil
}
