package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/repository"
)

type UserService struct {
	userRepo repository.UserRepository
	prRepo   repository.PullRequestRepository
	logger   *slog.Logger
}

func NewUserService(userRepo repository.UserRepository, prRepo repository.PullRequestRepository, logger *slog.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		prRepo:   prRepo,
		logger:   logger,
	}
}

func (s *UserService) SetUserActive(ctx context.Context, userID string, isActive bool) (*models.User, error) {
	s.logger.InfoContext(ctx, "updating user activity", "user_id", userID, "is_active", isActive)

	_, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.ErrorContext(ctx, "user not found", "error", err, "user_id", userID)
			return nil, ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get user", "error", err, "user_id", userID)
		return nil, err
	}

	updatedUser, err := s.userRepo.UpdateActivity(userID, isActive)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to update user activity", "error", err, "user_id", userID)
		return nil, err
	}

	s.logger.InfoContext(ctx, "user activity updated", "user_id", userID, "is_active", isActive)
	return updatedUser, nil
}

func (s *UserService) GetUserReviews(ctx context.Context, userID string) ([]*models.PullRequestShort, error) {
	s.logger.DebugContext(ctx, "fetching user reviews", "user_id", userID)

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.logger.ErrorContext(ctx, "user not found", "error", err, "user_id", userID)
			return nil, ErrUserNotFound
		}
		s.logger.ErrorContext(ctx, "failed to get user", "error", err, "user_id", userID)
		return nil, err
	}

	reviews, err := s.prRepo.GetByReviewerID(user.UserID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch reviews", "error", err, "user_id", userID)
		return nil, err
	}

	s.logger.DebugContext(ctx, "reviews fetched", "user_id", userID, "count", len(reviews))
	return reviews, nil
}
