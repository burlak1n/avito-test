package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/repository"
)

type TeamService struct {
	teamRepo repository.TeamRepository
	logger   *slog.Logger
}

func NewTeamService(teamRepo repository.TeamRepository, logger *slog.Logger) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		logger:   logger,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, team *models.Team) error {
	s.logger.InfoContext(ctx, "creating team", "team_name", team.TeamName)

	existing, err := s.teamRepo.GetByName(team.TeamName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		s.logger.ErrorContext(ctx, "failed to check team existence", "error", err, "team_name", team.TeamName)
		return err
	}
	if existing != nil {
		s.logger.WarnContext(ctx, "team already exists", "team_name", team.TeamName)
		return ErrTeamExists
	}

	if err := s.teamRepo.Create(team); err != nil {
		s.logger.ErrorContext(ctx, "failed to create team", "error", err, "team_name", team.TeamName)
		return err
	}

	s.logger.InfoContext(ctx, "team created successfully", "team_name", team.TeamName)
	return nil
}

func (s *TeamService) GetTeam(ctx context.Context, teamName string) (*models.Team, error) {
	s.logger.DebugContext(ctx, "fetching team", "team_name", teamName)

	team, err := s.teamRepo.GetByName(teamName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.ErrorContext(ctx, "team not found", "error", err, "team_name", teamName)
			return nil, ErrTeamNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get team", "error", err, "team_name", teamName)
		return nil, err
	}

	return team, nil
}
