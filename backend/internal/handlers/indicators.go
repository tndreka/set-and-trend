package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
)

type IndicatorHandler struct {
	indicatorRepo *repositories.IndicatorRepository
	candleRepo    *repositories.CandleRepository
}

func NewIndicatorHandler(
	indicatorRepo *repositories.IndicatorRepository,
	candleRepo *repositories.CandleRepository,
) *IndicatorHandler {
	return &IndicatorHandler{
		indicatorRepo: indicatorRepo,
		candleRepo:    candleRepo,
	}
}

type ComputeIndicatorRequest struct {
	CandleID string `json:"candle_id" binding:"required,uuid"`
}

func (h *IndicatorHandler) ComputeIndicator(c *gin.Context) {
	var req ComputeIndicatorRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid indicator request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	candleID, err := uuid.Parse(req.CandleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid candle_id"})
		return
	}

	// Get the candle
	candles, err := h.candleRepo.GetLatestCandles(c.Request.Context(), 1)
	if err != nil || len(candles) == 0 {
		log.Error().Err(err).Msg("candle not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "candle not found"})
		return
	}

	candle := candles[0]

	// Parse OHLC
	open, _ := strconv.ParseFloat(candle.Open, 64)
	high, _ := strconv.ParseFloat(candle.High, 64)
	low, _ := strconv.ParseFloat(candle.Low, 64)
	close, _ := strconv.ParseFloat(candle.Close, 64)

	// Compute indicators
	indicators := services.ComputeBasicIndicators(services.Candle{
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: candle.Volume,
	})

	// Store indicators
	indicator, err := h.indicatorRepo.CreateIndicator(c.Request.Context(), repositories.IndicatorCreateParams{
		ID:                 uuid.New(),
		CandleID:           candleID,
		EMA20:              indicators.EMA20,
		EMA50:              indicators.EMA50,
		EMA200:             indicators.EMA200,
		RangeSize:          indicators.RangeSize,
		BodySize:           indicators.BodySize,
		UpperWick:          indicators.UpperWick,
		LowerWick:          indicators.LowerWick,
		MidPrice:           indicators.MidPrice,
		LastSwingHighPrice: indicators.LastSwingHighPrice,
		LastSwingLowPrice:  indicators.LastSwingLowPrice,
	})
	if err != nil {
		log.Error().Err(err).Msg("create indicator failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Info().
		Str("indicator_id", indicator.ID.String()).
		Str("candle_id", candleID.String()).
		Msg("indicator computed")

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": indicator})
}
