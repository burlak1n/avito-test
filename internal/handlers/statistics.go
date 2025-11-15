package handlers

import (
	"log/slog"
	"net/http"

	"github.com/reviewer-service/internal/service"
)

type StatisticsHandler struct {
	service *service.StatisticsService
	logger  *slog.Logger
}

func NewStatisticsHandler(service *service.StatisticsService, logger *slog.Logger) *StatisticsHandler {
	return &StatisticsHandler{
		service: service,
		logger:  logger,
	}
}

func (h *StatisticsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.service.GetStatistics(ctx)
	if err != nil {
		h.logger.ErrorContext(ctx, "failed to get statistics", "error", err)
		respondError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get statistics")
		return
	}

	respondJSON(w, http.StatusOK, stats)
}



