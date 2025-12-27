package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
)

type CandleHandler struct {
	candleRepo *repositories.CandleRepository
}

func NewCandleHandler(candleRepo *repositories.CandleRepository) *CandleHandler {
	return &CandleHandler{candleRepo: candleRepo}
}

type CreateCandleRequest struct {
	TimestampUTC string  `json:"timestamp_utc" binding:"required"`
	Open         string  `json:"open" binding:"required"`
	High         string  `json:"high" binding:"required"`
	Low          string  `json:"low" binding:"required"`
	Close        string  `json:"close" binding:"required"`
	Volume       *int64  `json:"volume"`
}

func (h *CandleHandler) CreateCandle(c *gin.Context) {
	var req CreateCandleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid candle request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, req.TimestampUTC)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timestamp format, use RFC3339"})
		return
	}

	candle, err := h.candleRepo.CreateCandle(c.Request.Context(), repositories.CandleCreateParams{
		ID:           uuid.New(),
		TimestampUTC: timestamp,
		Open:         req.Open,
		High:         req.High,
		Low:          req.Low,
		Close:        req.Close,
		Volume:       req.Volume,
	})
	if err != nil {
		log.Error().Err(err).Msg("create candle failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Info().
		Str("candle_id", candle.ID.String()).
		Time("timestamp", candle.TimestampUTC).
		Msg("candle created")

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": candle})
}

func (h *CandleHandler) GetLatestCandles(c *gin.Context) {
	candles, err := h.candleRepo.GetLatestCandles(c.Request.Context(), 20)
	if err != nil {
		log.Error().Err(err).Msg("get candles failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "success", "data": candles})
}
