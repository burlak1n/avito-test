package handlers

import (
	"encoding/json"
	"net/http"
	"github.com/reviewer-service/internal/models"
	"github.com/reviewer-service/internal/storage"
)

type Handler struct {
	storage storage.Storage
}

func NewHandler(storage storage.Storage) *Handler {
	return &Handler{storage: storage}
}

func (h *Handler) AddTeam(w http.ResponseWriter, r *http.Request) {
	var team models.Team
	if err := json.NewDecoder(r.Body).Decode(&team); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	if err := h.storage.CreateTeam(&team); err != nil {
		respondError(w, http.StatusBadRequest, "TEAM_EXISTS", "team_name already exists")
		return
	}
	
	respondJSON(w, http.StatusCreated, map[string]interface{}{"team": team})
}

func (h *Handler) GetTeam(w http.ResponseWriter, r *http.Request) {
	teamName := r.URL.Query().Get("team_name")
	if teamName == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "team_name is required")
		return
	}
	
	team, err := h.storage.GetTeam(teamName)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "Team not found")
		return
	}
	
	respondJSON(w, http.StatusOK, team)
}

func (h *Handler) SetUserActive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserID   string `json:"user_id"`
		IsActive bool   `json:"is_active"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	user, err := h.storage.UpdateUserActivity(req.UserID, req.IsActive)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{"user": user})
}

func (h *Handler) CreatePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID   string `json:"pull_request_id"`
		PullRequestName string `json:"pull_request_name"`
		AuthorID        string `json:"author_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	// TODO: implement PR creation logic with reviewer assignment
	respondJSON(w, http.StatusCreated, map[string]interface{}{})
}

func (h *Handler) MergePR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	// TODO: implement merge logic
	respondJSON(w, http.StatusOK, map[string]interface{}{})
}

func (h *Handler) ReassignReviewer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PullRequestID string `json:"pull_request_id"`
		OldUserID     string `json:"old_user_id"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}
	
	// TODO: implement reassign logic
	respondJSON(w, http.StatusOK, map[string]interface{}{})
}

func (h *Handler) GetUserReviews(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "user_id is required")
		return
	}
	
	reviews, err := h.storage.GetUserReviews(userID)
	if err != nil {
		respondError(w, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}
	
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"user_id": userID,
		"pull_requests": reviews,
	})
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

