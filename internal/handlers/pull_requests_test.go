package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/service"
)

func setupTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

type mockPRService struct {
	createPRFunc          func(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error)
	mergePRFunc           func(ctx context.Context, prID string) (*models.PullRequest, error)
	reassignReviewerFunc  func(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error)
}

func (m *mockPRService) CreatePR(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error) {
	if m.createPRFunc != nil {
		return m.createPRFunc(ctx, prID, prName, authorID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPRService) MergePR(ctx context.Context, prID string) (*models.PullRequest, error) {
	if m.mergePRFunc != nil {
		return m.mergePRFunc(ctx, prID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockPRService) ReassignReviewer(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error) {
	if m.reassignReviewerFunc != nil {
		return m.reassignReviewerFunc(ctx, prID, oldUserID)
	}
	return nil, "", errors.New("not implemented")
}

func TestPullRequestHandler_CreatePR(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockService    *mockPRService
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful creation",
			requestBody: map[string]string{
				"pull_request_id":   "pr-1",
				"pull_request_name": "Test PR",
				"author_id":         "user-1",
			},
			mockService: &mockPRService{
				createPRFunc: func(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error) {
					return &models.PullRequest{
						PullRequestID:   prID,
						PullRequestName: prName,
						AuthorID:        authorID,
						Status:          "OPEN",
					}, nil
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "PR already exists",
			requestBody: map[string]string{
				"pull_request_id":   "pr-existing",
				"pull_request_name": "Existing PR",
				"author_id":         "user-1",
			},
			mockService: &mockPRService{
				createPRFunc: func(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error) {
					return nil, service.ErrPRExists
				},
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "PR_EXISTS",
		},
		{
			name: "author not found",
			requestBody: map[string]string{
				"pull_request_id":   "pr-1",
				"pull_request_name": "Test PR",
				"author_id":         "user-not-found",
			},
			mockService: &mockPRService{
				createPRFunc: func(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error) {
					return nil, service.ErrAuthorNotFound
				},
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name: "internal server error (database/connection)",
			requestBody: map[string]string{
				"pull_request_id":   "pr-1",
				"pull_request_name": "Test PR",
				"author_id":         "user-1",
			},
			mockService: &mockPRService{
				createPRFunc: func(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error) {
					return nil, errors.New("database connection failed")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "INTERNAL_ERROR",
		},
		{
			name:        "invalid request body",
			requestBody: "invalid json",
			mockService: &mockPRService{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/pullRequest/create", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := &PullRequestHandler{
				service: tt.mockService,
				logger:  setupTestLogger(),
			}

			handler.CreatePR(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response models.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if response.Error.Code != tt.expectedError {
					t.Errorf("expected error code %s, got %s", tt.expectedError, response.Error.Code)
				}
			}
		})
	}
}

func TestPullRequestHandler_MergePR(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockService    *mockPRService
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful merge",
			requestBody: map[string]string{
				"pull_request_id": "pr-1",
			},
			mockService: &mockPRService{
				mergePRFunc: func(ctx context.Context, prID string) (*models.PullRequest, error) {
					return &models.PullRequest{
						PullRequestID: prID,
						Status:        "MERGED",
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "PR not found",
			requestBody: map[string]string{
				"pull_request_id": "pr-not-found",
			},
			mockService: &mockPRService{
				mergePRFunc: func(ctx context.Context, prID string) (*models.PullRequest, error) {
					return nil, service.ErrPRNotFound
				},
			},
			expectedStatus: http.StatusNotFound,
			expectedError:  "NOT_FOUND",
		},
		{
			name: "internal server error (database/connection)",
			requestBody: map[string]string{
				"pull_request_id": "pr-1",
			},
			mockService: &mockPRService{
				mergePRFunc: func(ctx context.Context, prID string) (*models.PullRequest, error) {
					return nil, errors.New("database connection failed")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedError:  "INTERNAL_ERROR",
		},
		{
			name:        "invalid request body",
			requestBody: "invalid json",
			mockService: &mockPRService{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/pullRequest/merge", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := &PullRequestHandler{
				service: tt.mockService,
				logger:  setupTestLogger(),
			}

			handler.MergePR(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response models.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if response.Error.Code != tt.expectedError {
					t.Errorf("expected error code %s, got %s", tt.expectedError, response.Error.Code)
				}
			}
		})
	}
}

func TestPullRequestHandler_ReassignReviewer(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		mockService    *mockPRService
		expectedStatus int
		expectedError  string
	}{
		{
			name: "successful reassignment",
			requestBody: map[string]string{
				"pull_request_id": "pr-1",
				"old_user_id":     "user-1",
			},
			mockService: &mockPRService{
				reassignReviewerFunc: func(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error) {
					return &models.PullRequest{
						PullRequestID: prID,
						Status:        "OPEN",
					}, "user-2", nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "PR merged",
			requestBody: map[string]string{
				"pull_request_id": "pr-merged",
				"old_user_id":     "user-1",
			},
			mockService: &mockPRService{
				reassignReviewerFunc: func(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error) {
					return nil, "", service.ErrPRMerged
				},
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "PR_MERGED",
		},
		{
			name: "reviewer not assigned",
			requestBody: map[string]string{
				"pull_request_id": "pr-1",
				"old_user_id":     "user-not-assigned",
			},
			mockService: &mockPRService{
				reassignReviewerFunc: func(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error) {
					return nil, "", service.ErrNotAssigned
				},
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "NOT_ASSIGNED",
		},
		{
			name: "no candidate",
			requestBody: map[string]string{
				"pull_request_id": "pr-1",
				"old_user_id":     "user-1",
			},
			mockService: &mockPRService{
				reassignReviewerFunc: func(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error) {
					return nil, "", service.ErrNoCandidate
				},
			},
			expectedStatus: http.StatusConflict,
			expectedError:  "NO_CANDIDATE",
		},
		{
			name:        "invalid request body",
			requestBody: "invalid json",
			mockService: &mockPRService{},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_REQUEST",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var bodyBytes []byte
			var err error
			if str, ok := tt.requestBody.(string); ok {
				bodyBytes = []byte(str)
			} else {
				bodyBytes, err = json.Marshal(tt.requestBody)
				if err != nil {
					t.Fatalf("failed to marshal request body: %v", err)
				}
			}

			req := httptest.NewRequest("POST", "/pullRequest/reassign", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler := &PullRequestHandler{
				service: tt.mockService,
				logger:  setupTestLogger(),
			}

			handler.ReassignReviewer(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedError != "" {
				var response models.ErrorResponse
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("failed to unmarshal response: %v", err)
				}
				if response.Error.Code != tt.expectedError {
					t.Errorf("expected error code %s, got %s", tt.expectedError, response.Error.Code)
				}
			}
		})
	}
}

