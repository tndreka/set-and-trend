package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
)

type AnalyticsHandler struct {
	analyticsRepo *repositories.AnalyticsRepository
}

func NewAnalyticsHandler(analyticsRepo *repositories.AnalyticsRepository) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsRepo: analyticsRepo}
}

// GetSummary returns overall trading statistics
// GET /api/analytics/summary?user_id=uuid (optional)
func (h *AnalyticsHandler) GetSummary(c *gin.Context) {
	userID := c.Query("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	summary, err := h.analyticsRepo.GetTradeSummary(c.Request.Context(), userIDPtr)
	if err != nil {
		log.Error().Err(err).Msg("get analytics summary failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   summary,
	})
}

// GetByRule returns statistics grouped by rule
// GET /api/analytics/by-rule?user_id=uuid (optional)
func (h *AnalyticsHandler) GetByRule(c *gin.Context) {
	userID := c.Query("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	stats, err := h.analyticsRepo.GetStatsByRule(c.Request.Context(), userIDPtr)
	if err != nil {
		log.Error().Err(err).Msg("get analytics by rule failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics by rule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
		"count":  len(stats),
	})
}

// GetBySession returns statistics grouped by trading session
// GET /api/analytics/by-session?user_id=uuid (optional)
func (h *AnalyticsHandler) GetBySession(c *gin.Context) {
	userID := c.Query("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	stats, err := h.analyticsRepo.GetStatsBySession(c.Request.Context(), userIDPtr)
	if err != nil {
		log.Error().Err(err).Msg("get analytics by session failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics by session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
		"count":  len(stats),
	})
}

// GetByStrategy returns statistics grouped by strategy
// GET /api/analytics/by-strategy?user_id=uuid (optional)
func (h *AnalyticsHandler) GetByStrategy(c *gin.Context) {
	userID := c.Query("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	stats, err := h.analyticsRepo.GetStatsByStrategy(c.Request.Context(), userIDPtr)
	if err != nil {
		log.Error().Err(err).Msg("get analytics by strategy failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics by strategy"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
		"count":  len(stats),
	})
}

// GetByEmotion returns statistics grouped by emotion
// GET /api/analytics/by-emotion?user_id=uuid (optional)
func (h *AnalyticsHandler) GetByEmotion(c *gin.Context) {
	userID := c.Query("user_id")
	var userIDPtr *string
	if userID != "" {
		userIDPtr = &userID
	}

	stats, err := h.analyticsRepo.GetStatsByEmotion(c.Request.Context(), userIDPtr)
	if err != nil {
		log.Error().Err(err).Msg("get analytics by emotion failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get analytics by emotion"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   stats,
		"count":  len(stats),
	})
}
