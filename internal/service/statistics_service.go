package service

import (
	"context"
	"log/slog"

	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/repository"
)

type StatisticsService struct {
	statsRepo repository.StatisticsRepository
	logger    *slog.Logger
}

func NewStatisticsService(statsRepo repository.StatisticsRepository, logger *slog.Logger) *StatisticsService {
	return &StatisticsService{
		statsRepo: statsRepo,
		logger:    logger,
	}
}

func (s *StatisticsService) GetStatistics(ctx context.Context) (*models.Statistics, error) {
	s.logger.DebugContext(ctx, "fetching statistics")

	stats, err := s.statsRepo.GetStatistics()
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get statistics", "error", err)
		return nil, err
	}

	s.logger.DebugContext(ctx, "statistics fetched successfully",
		"teams", stats.Teams.Total,
		"users", stats.Users.Total,
		"prs", stats.PullRequests.Total,
		"assignments", stats.ReviewAssignments.Total,
	)

	return stats, nil
}
