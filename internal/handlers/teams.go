package handlers

import (
	"encoding/json"
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
		// OpenAPI: 400 Bad Request с кодом TEAM_EXISTS
		respondError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		return
	}

	// OpenAPI: 201 Created с { "team": {...} }
	respondJSON(w, http.StatusCreated, map[string]interface{}{"team": team})
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
		// OpenAPI: 404 Not Found с кодом NOT_FOUND
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Team not found")
		return
	}

	// OpenAPI: 200 OK, возвращаем Team напрямую
	respondJSON(w, http.StatusOK, team)
}
