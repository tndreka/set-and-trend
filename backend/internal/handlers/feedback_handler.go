package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
)

type FeedbackHandler struct {
	feedbackRepo *repositories.FeedbackRepository
	tradeRepo    *repositories.TradeRepository
}

func NewFeedbackHandler(
	feedbackRepo *repositories.FeedbackRepository,
	tradeRepo *repositories.TradeRepository,
) *FeedbackHandler {
	return &FeedbackHandler{
		feedbackRepo: feedbackRepo,
		tradeRepo:    tradeRepo,
	}
}

// Valid emotion types
var validEmotions = map[string]bool{
	"calm":    true,
	"anxious": true,
	"fomo":    true,
	"revenge": true,
	"other":   true,
}

type CreateFeedbackRequest struct {
	FollowedPlan   bool    `json:"followed_plan"`
	EmotionBefore  string  `json:"emotion_before" binding:"required"`
	EmotionDuring  string  `json:"emotion_during" binding:"required"`
	EmotionAfter   string  `json:"emotion_after" binding:"required"`
	BiggestMistake *string `json:"biggest_mistake"`
	ScreenshotURL  *string `json:"screenshot_url"`
}

func (h *FeedbackHandler) validateEmotions(before, during, after string) error {
	if !validEmotions[before] {
		return errors.New("invalid emotion_before: must be calm, anxious, fomo, revenge, or other")
	}
	if !validEmotions[during] {
		return errors.New("invalid emotion_during: must be calm, anxious, fomo, revenge, or other")
	}
	if !validEmotions[after] {
		return errors.New("invalid emotion_after: must be calm, anxious, fomo, revenge, or other")
	}
	return nil
}

// CreateFeedback creates feedback for a trade
// POST /api/trades/:id/feedback
func (h *FeedbackHandler) CreateFeedback(c *gin.Context) {
	tradeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade ID"})
		return
	}

	var req CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate emotions
	if err := h.validateEmotions(req.EmotionBefore, req.EmotionDuring, req.EmotionAfter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify trade exists
	_, err = h.tradeRepo.GetTradeByID(c.Request.Context(), tradeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "trade not found"})
			return
		}
		log.Error().Err(err).Msg("failed to get trade")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to verify trade"})
		return
	}

	// Check if feedback already exists
	existing, err := h.feedbackRepo.GetFeedbackByTradeID(c.Request.Context(), tradeID)
	if err == nil && existing != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "feedback already exists for this trade, use PUT to update"})
		return
	}

	feedback, err := h.feedbackRepo.CreateFeedback(c.Request.Context(), repositories.FeedbackCreateParams{
		TradeID:        tradeID,
		FollowedPlan:   req.FollowedPlan,
		EmotionBefore:  req.EmotionBefore,
		EmotionDuring:  req.EmotionDuring,
		EmotionAfter:   req.EmotionAfter,
		BiggestMistake: req.BiggestMistake,
		ScreenshotURL:  req.ScreenshotURL,
	})
	if err != nil {
		log.Error().Err(err).Msg("create feedback failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create feedback"})
		return
	}

	log.Info().
		Str("trade_id", tradeID.String()).
		Bool("followed_plan", feedback.FollowedPlan).
		Msg("trade feedback created")

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": feedback})
}

// GetFeedback retrieves feedback for a trade
// GET /api/trades/:id/feedback
func (h *FeedbackHandler) GetFeedback(c *gin.Context) {
	tradeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade ID"})
		return
	}

	feedback, err := h.feedbackRepo.GetFeedbackByTradeID(c.Request.Context(), tradeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found for this trade"})
			return
		}
		log.Error().Err(err).Msg("get feedback failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get feedback"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": feedback})
}

// UpdateFeedback updates feedback for a trade
// PUT /api/trades/:id/feedback
func (h *FeedbackHandler) UpdateFeedback(c *gin.Context) {
	tradeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade ID"})
		return
	}

	var req CreateFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate emotions
	if err := h.validateEmotions(req.EmotionBefore, req.EmotionDuring, req.EmotionAfter); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if feedback exists
	_, err = h.feedbackRepo.GetFeedbackByTradeID(c.Request.Context(), tradeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found for this trade"})
			return
		}
		log.Error().Err(err).Msg("get feedback failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get feedback"})
		return
	}

	feedback, err := h.feedbackRepo.UpdateFeedback(c.Request.Context(), repositories.FeedbackCreateParams{
		TradeID:        tradeID,
		FollowedPlan:   req.FollowedPlan,
		EmotionBefore:  req.EmotionBefore,
		EmotionDuring:  req.EmotionDuring,
		EmotionAfter:   req.EmotionAfter,
		BiggestMistake: req.BiggestMistake,
		ScreenshotURL:  req.ScreenshotURL,
	})
	if err != nil {
		log.Error().Err(err).Msg("update feedback failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update feedback"})
		return
	}

	log.Info().
		Str("trade_id", tradeID.String()).
		Msg("trade feedback updated")

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": feedback})
}

// DeleteFeedback deletes feedback for a trade
// DELETE /api/trades/:id/feedback
func (h *FeedbackHandler) DeleteFeedback(c *gin.Context) {
	tradeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade ID"})
		return
	}

	// Check if feedback exists
	_, err = h.feedbackRepo.GetFeedbackByTradeID(c.Request.Context(), tradeID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "feedback not found for this trade"})
			return
		}
		log.Error().Err(err).Msg("get feedback failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get feedback"})
		return
	}

	err = h.feedbackRepo.DeleteFeedback(c.Request.Context(), tradeID)
	if err != nil {
		log.Error().Err(err).Msg("delete feedback failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete feedback"})
		return
	}

	log.Info().
		Str("trade_id", tradeID.String()).
		Msg("trade feedback deleted")

	c.JSON(http.StatusOK, gin.H{"status": "success", "message": "feedback deleted"})
}
