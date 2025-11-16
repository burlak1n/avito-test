package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/reviewer-service/internal/models"
)

type mockPRRepository struct {
	prs map[string]*models.PullRequest
}

func (m *mockPRRepository) Create(pr *models.PullRequest) error {
	if _, exists := m.prs[pr.PullRequestID]; exists {
		return errors.New("PR already exists")
	}
	m.prs[pr.PullRequestID] = pr
	return nil
}

func (m *mockPRRepository) GetByID(prID string) (*models.PullRequest, error) {
	pr, exists := m.prs[prID]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return pr, nil
}

func (m *mockPRRepository) UpdateStatus(prID string, status string) error {
	pr, exists := m.prs[prID]
	if !exists {
		return sql.ErrNoRows
	}
	now := time.Now()
	pr.Status = status
	pr.MergedAt = &now
	return nil
}

func (m *mockPRRepository) UpdateReviewers(prID string, reviewers []string) error {
	pr, exists := m.prs[prID]
	if !exists {
		return sql.ErrNoRows
	}
	pr.AssignedReviewers = reviewers
	return nil
}

func (m *mockPRRepository) GetByReviewerID(userID string) ([]*models.PullRequestShort, error) {
	return nil, nil
}

func (m *mockPRRepository) GetOpenPRsByAuthors(userIDs []string) ([]*models.PullRequest, error) {
	return nil, nil
}

func (m *mockPRRepository) GetOpenPRsByReviewers(userIDs []string) (map[string][]*models.PullRequest, error) {
	return nil, nil
}

func (m *mockPRRepository) ReassignAuthor(tx *sql.Tx, prID, newAuthorID string) error {
	return nil
}

func (m *mockPRRepository) RemoveReviewer(tx *sql.Tx, prID, reviewerID string) error {
	return nil
}

func (m *mockPRRepository) AddReviewer(tx *sql.Tx, prID, reviewerID string) error {
	return nil
}

type mockUserRepository struct {
	users map[string]*models.User
}

func (m *mockUserRepository) GetByID(userID string) (*models.User, error) {
	user, exists := m.users[userID]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return user, nil
}

func (m *mockUserRepository) UpdateActivity(userID string, isActive bool) (*models.User, error) {
	user, exists := m.users[userID]
	if !exists {
		return nil, sql.ErrNoRows
	}
	user.IsActive = isActive
	return user, nil
}

func (m *mockUserRepository) GetActiveTeamMembers(teamName string, excludeUserID string) ([]*models.User, error) {
	var members []*models.User
	for _, user := range m.users {
		if user.TeamName == teamName && user.IsActive && user.UserID != excludeUserID {
			members = append(members, user)
		}
	}
	return members, nil
}

func (m *mockUserRepository) DeactivateUsers(tx *sql.Tx, userIDs []string) error {
	return nil
}

func (m *mockUserRepository) GetUsersByIDs(userIDs []string) ([]*models.User, error) {
	return nil, nil
}

func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestPullRequestService_CreatePR(t *testing.T) {
	tests := []struct {
		name          string
		prID          string
		prName        string
		authorID      string
		setupMocks    func() (*mockPRRepository, *mockUserRepository)
		expectedError error
		validate      func(t *testing.T, pr *models.PullRequest)
	}{
		{
			name:     "successful creation",
			prID:     "pr-1",
			prName:   "Test PR",
			authorID: "user-1",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				prRepo := &mockPRRepository{prs: make(map[string]*models.PullRequest)}
				userRepo := &mockUserRepository{
					users: map[string]*models.User{
						"user-1": {UserID: "user-1", Username: "author", TeamName: "team-1", IsActive: true},
						"user-2": {UserID: "user-2", Username: "reviewer1", TeamName: "team-1", IsActive: true},
						"user-3": {UserID: "user-3", Username: "reviewer2", TeamName: "team-1", IsActive: true},
					},
				}
				return prRepo, userRepo
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *models.PullRequest) {
				if pr == nil {
					t.Fatal("expected PR to be created")
				}
				if pr.PullRequestID != "pr-1" {
					t.Errorf("expected PR ID 'pr-1', got '%s'", pr.PullRequestID)
				}
				if pr.Status != "OPEN" {
					t.Errorf("expected status 'OPEN', got '%s'", pr.Status)
				}
				if len(pr.AssignedReviewers) != 2 {
					t.Errorf("expected 2 reviewers, got %d", len(pr.AssignedReviewers))
				}
			},
		},
		{
			name:     "PR already exists",
			prID:     "pr-existing",
			prName:   "Existing PR",
			authorID: "user-1",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				prRepo := &mockPRRepository{
					prs: map[string]*models.PullRequest{
						"pr-existing": {PullRequestID: "pr-existing"},
					},
				}
				userRepo := &mockUserRepository{
					users: map[string]*models.User{
						"user-1": {UserID: "user-1", Username: "author", TeamName: "team-1", IsActive: true},
					},
				}
				return prRepo, userRepo
			},
			expectedError: ErrPRExists,
		},
		{
			name:     "author not found",
			prID:     "pr-1",
			prName:   "Test PR",
			authorID: "user-not-found",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				prRepo := &mockPRRepository{prs: make(map[string]*models.PullRequest)}
				userRepo := &mockUserRepository{users: make(map[string]*models.User)}
				return prRepo, userRepo
			},
			expectedError: ErrAuthorNotFound,
		},
		{
			name:     "no active reviewers",
			prID:     "pr-1",
			prName:   "Test PR",
			authorID: "user-1",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				prRepo := &mockPRRepository{prs: make(map[string]*models.PullRequest)}
				userRepo := &mockUserRepository{
					users: map[string]*models.User{
						"user-1": {UserID: "user-1", Username: "author", TeamName: "team-1", IsActive: true},
					},
				}
				return prRepo, userRepo
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *models.PullRequest) {
				if len(pr.AssignedReviewers) != 0 {
					t.Errorf("expected 0 reviewers, got %d", len(pr.AssignedReviewers))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prRepo, userRepo := tt.setupMocks()
			service := NewPullRequestService(prRepo, userRepo, setupTestLogger())

			pr, err := service.CreatePR(context.Background(), tt.prID, tt.prName, tt.authorID)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, pr)
			}
		})
	}
}

func TestPullRequestService_MergePR(t *testing.T) {
	tests := []struct {
		name          string
		prID          string
		setupMocks    func() *mockPRRepository
		expectedError error
		validate      func(t *testing.T, pr *models.PullRequest)
	}{
		{
			name: "successful merge",
			prID: "pr-1",
			setupMocks: func() *mockPRRepository {
				now := time.Now()
				return &mockPRRepository{
					prs: map[string]*models.PullRequest{
						"pr-1": {
							PullRequestID: "pr-1",
							Status:        "OPEN",
							CreatedAt:     &now,
						},
					},
				}
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *models.PullRequest) {
				if pr.Status != "MERGED" {
					t.Errorf("expected status 'MERGED', got '%s'", pr.Status)
				}
				if pr.MergedAt == nil {
					t.Error("expected MergedAt to be set")
				}
			},
		},
		{
			name: "PR not found",
			prID: "pr-not-found",
			setupMocks: func() *mockPRRepository {
				return &mockPRRepository{prs: make(map[string]*models.PullRequest)}
			},
			expectedError: ErrPRNotFound,
		},
		{
			name: "already merged (idempotent)",
			prID: "pr-merged",
			setupMocks: func() *mockPRRepository {
				now := time.Now()
				return &mockPRRepository{
					prs: map[string]*models.PullRequest{
						"pr-merged": {
							PullRequestID: "pr-merged",
							Status:        "MERGED",
							MergedAt:      &now,
						},
					},
				}
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *models.PullRequest) {
				if pr.Status != "MERGED" {
					t.Errorf("expected status 'MERGED', got '%s'", pr.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prRepo := tt.setupMocks()
			userRepo := &mockUserRepository{users: make(map[string]*models.User)}
			service := NewPullRequestService(prRepo, userRepo, setupTestLogger())

			pr, err := service.MergePR(context.Background(), tt.prID)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, pr)
			}
		})
	}
}

func TestPullRequestService_ReassignReviewer(t *testing.T) {
	tests := []struct {
		name          string
		prID          string
		oldUserID     string
		setupMocks    func() (*mockPRRepository, *mockUserRepository)
		expectedError error
		validate      func(t *testing.T, pr *models.PullRequest, newUserID string)
	}{
		{
			name:      "successful reassignment",
			prID:      "pr-1",
			oldUserID: "user-1",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				now := time.Now()
				prRepo := &mockPRRepository{
					prs: map[string]*models.PullRequest{
						"pr-1": {
							PullRequestID:     "pr-1",
							Status:            "OPEN",
							AssignedReviewers: []string{"user-1", "user-2"},
							CreatedAt:         &now,
						},
					},
				}
				userRepo := &mockUserRepository{
					users: map[string]*models.User{
						"user-1": {UserID: "user-1", Username: "old", TeamName: "team-1", IsActive: true},
						"user-2": {UserID: "user-2", Username: "reviewer2", TeamName: "team-1", IsActive: true},
						"user-3": {UserID: "user-3", Username: "new", TeamName: "team-1", IsActive: true},
					},
				}
				return prRepo, userRepo
			},
			expectedError: nil,
			validate: func(t *testing.T, pr *models.PullRequest, newUserID string) {
				if newUserID == "" {
					t.Error("expected new user ID to be set")
				}
				if newUserID == "user-1" {
					t.Error("new user should be different from old user")
				}
				found := false
				for _, reviewer := range pr.AssignedReviewers {
					if reviewer == newUserID {
						found = true
					}
					if reviewer == "user-1" {
						t.Error("old reviewer should not be in the list")
					}
				}
				if !found {
					t.Error("new reviewer should be in the list")
				}
			},
		},
		{
			name:      "PR not found",
			prID:      "pr-not-found",
			oldUserID: "user-1",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				return &mockPRRepository{prs: make(map[string]*models.PullRequest)},
					&mockUserRepository{users: make(map[string]*models.User)}
			},
			expectedError: ErrPRNotFound,
		},
		{
			name:      "PR already merged",
			prID:      "pr-merged",
			oldUserID: "user-1",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				now := time.Now()
				return &mockPRRepository{
					prs: map[string]*models.PullRequest{
						"pr-merged": {
							PullRequestID:     "pr-merged",
							Status:            "MERGED",
							AssignedReviewers: []string{"user-1"},
							MergedAt:          &now,
						},
					},
				}, &mockUserRepository{users: make(map[string]*models.User)}
			},
			expectedError: ErrPRMerged,
		},
		{
			name:      "reviewer not assigned",
			prID:      "pr-1",
			oldUserID: "user-not-assigned",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				now := time.Now()
				return &mockPRRepository{
					prs: map[string]*models.PullRequest{
						"pr-1": {
							PullRequestID:     "pr-1",
							Status:            "OPEN",
							AssignedReviewers: []string{"user-1", "user-2"},
							CreatedAt:         &now,
						},
					},
				}, &mockUserRepository{users: make(map[string]*models.User)}
			},
			expectedError: ErrNotAssigned,
		},
		{
			name:      "no replacement candidate",
			prID:      "pr-1",
			oldUserID: "user-1",
			setupMocks: func() (*mockPRRepository, *mockUserRepository) {
				now := time.Now()
				prRepo := &mockPRRepository{
					prs: map[string]*models.PullRequest{
						"pr-1": {
							PullRequestID:     "pr-1",
							Status:            "OPEN",
							AssignedReviewers: []string{"user-1", "user-2"},
							CreatedAt:         &now,
						},
					},
				}
				userRepo := &mockUserRepository{
					users: map[string]*models.User{
						"user-1": {UserID: "user-1", Username: "old", TeamName: "team-1", IsActive: true},
						"user-2": {UserID: "user-2", Username: "reviewer2", TeamName: "team-1", IsActive: true},
					},
				}
				return prRepo, userRepo
			},
			expectedError: ErrNoCandidate,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prRepo, userRepo := tt.setupMocks()
			service := NewPullRequestService(prRepo, userRepo, setupTestLogger())

			pr, newUserID, err := service.ReassignReviewer(context.Background(), tt.prID, tt.oldUserID)

			if tt.expectedError != nil {
				if err == nil {
					t.Errorf("expected error %v, got nil", tt.expectedError)
				} else if !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, pr, newUserID)
			}
		})
	}
}

