package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/service"
)

type TeamHandler struct {
	service *service.TeamService
	logger  *slog.Logger
}

func NewTeamHandler(service *service.TeamService, logger *slog.Logger) *TeamHandler {
	return &TeamHandler{
		service: service,
		logger:  logger,
	}
}

func (h *TeamHandler) AddTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var team models.Team

	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", "error", err)
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if err := h.service.CreateTeam(ctx, &team); err != nil {
		if errors.Is(err, service.ErrTeamExists) {
			// OpenAPI: 400 Bad Request с кодом TEAM_EXISTS
			respondError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		} else {
			// Ошибки БД или другие ошибки репозитория
			h.logger.ErrorContext(ctx, "failed to create team", "error", err, "team_name", team.TeamName)
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	created, err := h.service.GetTeam(ctx, team.TeamName)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get created team", "error", err, "team_name", team.TeamName)
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		return
	}

	// OpenAPI: 201 Created с { "team": {...} }
	respondJSON(w, http.StatusCreated, map[string]interface{}{"team": created})
}

func (h *TeamHandler) GetTeam(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	teamName := r.URL.Query().Get("team_name")

	if teamName == "" {
		h.logger.WarnContext(ctx, "team_name parameter missing")
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "team_name is required")
		return
	}

	team, err := h.service.GetTeam(ctx, teamName)
	if err != nil {
		if errors.Is(err, service.ErrTeamNotFound) {
			// OpenAPI: 404 Not Found с кодом NOT_FOUND
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Team not found")
		} else {
			// Ошибки БД или другие ошибки репозитория
			h.logger.ErrorContext(ctx, "failed to get team", "error", err, "team_name", teamName)
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	// OpenAPI: 200 OK, возвращаем Team напрямую
	respondJSON(w, http.StatusOK, team)
}

func (h *TeamHandler) DeactivateTeamMembers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req struct {
		TeamName string   `json:"team_name"`
		UserIDs  []string `json:"user_ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.WarnContext(ctx, "invalid request body", "error", err)
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	result, err := h.service.DeactivateTeamMembers(ctx, req.TeamName, req.UserIDs)
	if err != nil {
		if errors.Is(err, service.ErrTeamNotFound) || errors.Is(err, service.ErrUserNotFound) {
			respondError(w, http.StatusNotFound, "NOT_FOUND", "Team or user not found")
		} else if errors.Is(err, service.ErrInvalidTeamMember) {
			respondError(w, http.StatusBadRequest, "INVALID_TEAM_MEMBER", "One or more users are not members of the specified team")
		} else {
			h.logger.ErrorContext(ctx, "failed to deactivate team members", "error", err, "team_name", req.TeamName)
			respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
		}
		return
	}

	respondJSON(w, http.StatusOK, result)
}
