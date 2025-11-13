package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/reviewer-service/internal/service"
)

type UserHandler struct {
	service *service.UserService
	logger  *slog.Logger
}

func NewUserHandler(service *service.UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{
		service: service,
		logger:  logger,
	}
}

func (h *UserHandler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", "error", err)
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	user, err := h.service.SetUserActive(ctx, req.UserID, req.IsActive)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			// OpenAPI: 404 Not Found с кодом NOT_FOUND
			respondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		} else {
			// Ошибки БД или другие ошибки репозитория
			h.logger.ErrorContext(ctx, "failed to set user active", "error", err, "user_id", req.UserID)
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	// OpenAPI: 200 OK с { "user": {...} }
	respondJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

func (h *UserHandler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := r.URL.Query().Get("user_id")

	if userID == "" {
		h.logger.WarnContext(ctx, "user_id parameter missing")
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
		return
	}

	reviews, err := h.service.GetUserReviews(ctx, userID)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) {
			// OpenAPI: возвращает 200 даже если пользователя нет, с пустым списком
			// Но для консистентности можно возвращать 404
			respondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		} else {
			// Ошибки БД или другие ошибки репозитория
			h.logger.ErrorContext(ctx, "failed to get user reviews", "error", err, "user_id", userID)
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	// OpenAPI: 200 OK с { "user_id": "...", "pull_requests": [...] }
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id":       userID,
		"pull_requests": reviews,
	})
}
