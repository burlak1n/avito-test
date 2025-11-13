package service

import (
	"context"
	"log/slog"
	"math/rand"
	"time"

	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/repository"
)

type PullRequestService struct {
	prRepo   repository.PullRequestRepository
	userRepo repository.UserRepository
	logger   *slog.Logger
}

func NewPullRequestService(prRepo repository.PullRequestRepository, userRepo repository.UserRepository, logger *slog.Logger) *PullRequestService {
	return &PullRequestService{
		prRepo:   prRepo,
		userRepo: userRepo,
		logger:   logger,
	}
}

func (s *PullRequestService) CreatePR(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error) {
	s.logger.InfoContext(ctx, "creating PR", "pr_id", prID, "author_id", authorID)

	existing, _ := s.prRepo.GetByID(prID)
	if existing != nil {
		s.logger.WarnContext(ctx, "PR already exists", "pr_id", prID)
		return nil, ErrPRExists
	}

	author, err := s.userRepo.GetByID(authorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "author not found", "error", err, "author_id", authorID)
		return nil, ErrAuthorNotFound
	}

	candidates, err := s.userRepo.GetActiveTeamMembers(author.TeamName, authorID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get team members", "error", err, "team_name", author.TeamName)
		return nil, err
	}

	reviewers := selectRandomReviewers(candidates, 2)
	s.logger.InfoContext(ctx, "reviewers selected", "pr_id", prID, "reviewers", reviewers, "candidates_count", len(candidates))

	now := time.Now()
	pr := &models.PullRequest{
		PullRequestID:     prID,
		PullRequestName:   prName,
		AuthorID:          authorID,
		Status:            "OPEN",
		AssignedReviewers: reviewers,
		CreatedAt:         &now,
	}

	if err := s.prRepo.Create(pr); err != nil {
		s.logger.ErrorContext(ctx, "failed to create PR", "error", err, "pr_id", prID)
		return nil, err
	}

	s.logger.InfoContext(ctx, "PR created successfully", "pr_id", prID, "reviewers_count", len(reviewers))
	return pr, nil
}

func (s *PullRequestService) MergePR(ctx context.Context, prID string) (*models.PullRequest, error) {
	s.logger.InfoContext(ctx, "merging PR", "pr_id", prID)

	pr, err := s.prRepo.GetByID(prID)
	if err != nil {
		s.logger.ErrorContext(ctx, "PR not found", "error", err, "pr_id", prID)
		return nil, ErrPRNotFound
	}

	if pr.Status == "MERGED" {
		s.logger.InfoContext(ctx, "PR already merged (idempotent)", "pr_id", prID)
		return pr, nil
	}

	if err := s.prRepo.UpdateStatus(prID, "MERGED"); err != nil {
		s.logger.ErrorContext(ctx, "failed to merge PR", "error", err, "pr_id", prID)
		return nil, err
	}

	s.logger.InfoContext(ctx, "PR merged successfully", "pr_id", prID)
	return s.prRepo.GetByID(prID)
}

func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error) {
	s.logger.InfoContext(ctx, "reassigning reviewer", "pr_id", prID, "old_user_id", oldUserID)

	pr, err := s.prRepo.GetByID(prID)
	if err != nil {
		s.logger.ErrorContext(ctx, "PR not found", "error", err, "pr_id", prID)
		return nil, "", ErrPRNotFound
	}

	if pr.Status == "MERGED" {
		s.logger.WarnContext(ctx, "cannot reassign on merged PR", "pr_id", prID)
		return nil, "", ErrPRMerged
	}

	found := false
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID == oldUserID {
			found = true
			break
		}
	}
	if !found {
		s.logger.WarnContext(ctx, "reviewer not assigned to PR", "pr_id", prID, "user_id", oldUserID)
		return nil, "", ErrNotAssigned
	}

	oldUser, err := s.userRepo.GetByID(oldUserID)
	if err != nil {
		s.logger.ErrorContext(ctx, "old reviewer not found", "error", err, "user_id", oldUserID)
		return nil, "", ErrUserNotFound
	}

	candidates, err := s.userRepo.GetActiveTeamMembers(oldUser.TeamName, oldUserID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to get team members", "error", err, "team_name", oldUser.TeamName)
		return nil, "", err
	}

	filteredCandidates := make([]*models.User, 0)
	for _, candidate := range candidates {
		isAlreadyAssigned := false
		for _, reviewerID := range pr.AssignedReviewers {
			if candidate.UserID == reviewerID {
				isAlreadyAssigned = true
				break
			}
		}
		if !isAlreadyAssigned {
			filteredCandidates = append(filteredCandidates, candidate)
		}
	}

	if len(filteredCandidates) == 0 {
		s.logger.WarnContext(ctx, "no replacement candidates available", "pr_id", prID, "team_name", oldUser.TeamName)
		return nil, "", ErrNoCandidate
	}

	newReviewers := make([]string, 0, len(pr.AssignedReviewers))
	for _, reviewerID := range pr.AssignedReviewers {
		if reviewerID != oldUserID {
			newReviewers = append(newReviewers, reviewerID)
		}
	}

	newReviewer := filteredCandidates[rand.Intn(len(filteredCandidates))]
	newReviewers = append(newReviewers, newReviewer.UserID)

	if err := s.prRepo.UpdateReviewers(prID, newReviewers); err != nil {
		s.logger.ErrorContext(ctx, "failed to update reviewers", "error", err, "pr_id", prID)
		return nil, "", err
	}

	updatedPR, err := s.prRepo.GetByID(prID)
	if err != nil {
		s.logger.ErrorContext(ctx, "failed to fetch updated PR", "error", err, "pr_id", prID)
		return nil, "", err
	}

	s.logger.InfoContext(ctx, "reviewer reassigned successfully", "pr_id", prID, "old_user_id", oldUserID, "new_user_id", newReviewer.UserID)
	return updatedPR, newReviewer.UserID, nil
}

func selectRandomReviewers(candidates []*models.User, maxCount int) []string {
	if len(candidates) == 0 {
		return []string{}
	}

	count := maxCount
	if len(candidates) < count {
		count = len(candidates)
	}

	shuffled := make([]*models.User, len(candidates))
	copy(shuffled, candidates)
	rand.Shuffle(len(shuffled), func(i, j int) {
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	})

	reviewers := make([]string, count)
	for i := 0; i < count; i++ {
		reviewers[i] = shuffled[i].UserID
	}

	return reviewers
}
