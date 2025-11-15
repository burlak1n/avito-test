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
	userRepo repository.UserRepository
	prRepo   repository.PullRequestRepository
	db       interface {
		BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error)
	}
	logger *slog.Logger
}

func NewTeamService(teamRepo repository.TeamRepository, userRepo repository.UserRepository, prRepo repository.PullRequestRepository, db *sql.DB, logger *slog.Logger) *TeamService {
	return &TeamService{
		teamRepo: teamRepo,
		userRepo: userRepo,
		prRepo:   prRepo,
		db:       db,
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

func (s *TeamService) DeactivateTeamMembers(ctx context.Context, teamName string, userIDs []string) (map[string]interface{}, error) {
	s.logger.InfoContext(ctx, "deactivating team members", "team_name", teamName, "user_ids", userIDs)

	if len(userIDs) == 0 {
		return map[string]interface{}{
			"deactivated_users": []string{},
			"reassigned_prs":    0,
		}, nil
	}

	team, err := s.teamRepo.GetByName(teamName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrTeamNotFound
		}
		return nil, err
	}

	users, err := s.userRepo.GetUsersByIDs(userIDs)
	if err != nil {
		return nil, err
	}

	if len(users) != len(userIDs) {
		return nil, ErrUserNotFound
	}

	for _, u := range users {
		if u.TeamName != team.TeamName {
			return nil, ErrInvalidTeamMember
		}
	}

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	authorPRs, err := s.prRepo.GetOpenPRsByAuthors(userIDs)
	if err != nil {
		return nil, err
	}

	reviewerPRs, err := s.prRepo.GetOpenPRsByReviewers(userIDs)
	if err != nil {
		return nil, err
	}

	activeMembers, err := s.userRepo.GetActiveTeamMembers(teamName, "")
	if err != nil {
		return nil, err
	}

	activeMap := make(map[string]bool)
	for _, am := range activeMembers {
		activeMap[am.UserID] = true
	}
	for _, uid := range userIDs {
		delete(activeMap, uid)
	}

	var activeList []string
	for uid := range activeMap {
		activeList = append(activeList, uid)
	}

	reassignedCount := 0

	for _, pr := range authorPRs {
		if len(activeList) > 0 {
			newAuthor := activeList[reassignedCount%len(activeList)]
			if err := s.prRepo.ReassignAuthor(tx, pr.PullRequestID, newAuthor); err != nil {
				return nil, err
			}
			reassignedCount++
		}
	}

	for reviewerID, prs := range reviewerPRs {
		for _, pr := range prs {
			if err := s.prRepo.RemoveReviewer(tx, pr.PullRequestID, reviewerID); err != nil {
				return nil, err
			}

			updatedReviewers := make([]string, 0, len(pr.AssignedReviewers))
			for _, r := range pr.AssignedReviewers {
				if r != reviewerID {
					updatedReviewers = append(updatedReviewers, r)
				}
			}
			pr.AssignedReviewers = updatedReviewers

			if len(pr.AssignedReviewers) < 2 && len(activeList) > 0 {
				newReviewer := activeList[reassignedCount%len(activeList)]
				alreadyReviewer := false
				for _, r := range pr.AssignedReviewers {
					if r == newReviewer {
						alreadyReviewer = true
						break
					}
				}
				if !alreadyReviewer {
					if err := s.prRepo.AddReviewer(tx, pr.PullRequestID, newReviewer); err != nil {
						return nil, err
					}
					pr.AssignedReviewers = append(pr.AssignedReviewers, newReviewer)
					reassignedCount++
				}
			}
		}
	}

	if err := s.userRepo.DeactivateUsers(tx, userIDs); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	s.logger.InfoContext(ctx, "team members deactivated", "team_name", teamName, "count", len(userIDs), "reassigned", reassignedCount)

	return map[string]interface{}{
		"deactivated_users": userIDs,
		"reassigned_prs":    reassignedCount,
	}, nil
}
