package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/service"
)

type PRService interface {
	CreatePR(ctx context.Context, prID, prName, authorID string) (*models.PullRequest, error)
	MergePR(ctx context.Context, prID string) (*models.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID, oldUserID string) (*models.PullRequest, string, error)
}

type PullRequestHandler struct {
	service PRService
	logger  *slog.Logger
}

func NewPullRequestHandler(service *service.PullRequestService, logger *slog.Logger) *PullRequestHandler {
	return &PullRequestHandler{
		service: service,
		logger:  logger,
	}
}

func (h *PullRequestHandler) CreatePR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", "error", err)
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pr, err := h.service.CreatePR(ctx, req.PullRequestID, req.PullRequestName, req.AuthorID)
	if err != nil {
		// OpenAPI:
		// - 404 Not Found: автор/команда не найдены
		// - 409 Conflict с кодом PR_EXISTS: PR уже существует
		if errors.Is(err, service.ErrPRExists) {
			respondError(w, http.StatusConflict, "PR_EXISTS", "PR id already exists")
		} else if errors.Is(err, service.ErrAuthorNotFound) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Author or team not found")
		} else {
			h.logger.ErrorContext(ctx, "internal server error", "error", err)
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	// OpenAPI: 201 Created с { "pr": {...} }
	respondJSON(w, http.StatusCreated, map[string]interface{}{"pr": pr})
}

func (h *PullRequestHandler) MergePR(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", "error", err)
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pr, err := h.service.MergePR(ctx, req.PullRequestID)
	if err != nil {
		// OpenAPI: 404 Not Found с кодом NOT_FOUND
		if errors.Is(err, service.ErrPRNotFound) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "PR not found")
		} else {
			h.logger.ErrorContext(ctx, "internal server error", "error", err)
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	// OpenAPI: 200 OK с { "pr": {...} } (идемпотентно)
	respondJSON(w, http.StatusOK, map[string]interface{}{"pr": pr})
}

func (h *PullRequestHandler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"` // OpenAPI требует old_user_id
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", "error", err)
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	pr, replacedBy, err := h.service.ReassignReviewer(ctx, req.PullRequestID, req.OldUserID)
	if err != nil {
		// OpenAPI:
		// - 404 Not Found: PR или пользователь не найден
		// - 409 Conflict с кодами: PR_MERGED, NOT_ASSIGNED, NO_CANDIDATE

		if errors.Is(err, service.ErrPRMerged) {
			// OpenAPI: 409 Conflict с кодом PR_MERGED
			respondError(w, http.StatusConflict, "PR_MERGED", "cannot reassign on merged PR")
		} else if errors.Is(err, service.ErrNotAssigned) {
			// OpenAPI: 409 Conflict с кодом NOT_ASSIGNED
			respondError(w, http.StatusConflict, "NOT_ASSIGNED", "reviewer is not assigned to this PR")
		} else if errors.Is(err, service.ErrNoCandidate) {
			// OpenAPI: 409 Conflict с кодом NO_CANDIDATE
			respondError(w, http.StatusConflict, "NO_CANDIDATE", "no active replacement candidate in team")
		} else {
			// OpenAPI: 404 Not Found для "PR не найден" или "пользователь не найден"
			respondError(w, http.StatusNotFound, "NOT_FOUND", "PR or user not found")
		}
		return
	}

	// OpenAPI: 200 OK с { "pr": {...}, "replaced_by": "..." }
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"pr":          pr,
		"replaced_by": replacedBy,
	})
}
