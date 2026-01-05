package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/services"
)

type TradeHandler struct {
	tradeService *services.TradeService
}

func NewTradeHandler(tradeService *services.TradeService) *TradeHandler {
	return &TradeHandler{tradeService: tradeService}
}

type CreateTradeRequest struct {
	AccountID      string  `json:"account_id" binding:"required,uuid"`
	CandleID       string  `json:"candle_id" binding:"required,uuid"`
	Bias           string  `json:"bias" binding:"required,oneof=long short"`
	PlannedEntry   float64 `json:"planned_entry" binding:"required,gt=0"`
	PlannedSL      float64 `json:"planned_sl" binding:"required,gt=0"`
	PlannedTP      float64 `json:"planned_tp" binding:"required,gt=0"`
	PlannedRiskPct float64 `json:"planned_risk_pct" binding:"required,gt=0,lte=100"`
	ReasonForTrade string  `json:"reason_for_trade" binding:"required,min=10"`
}

func (h *TradeHandler) CreateTrade(c *gin.Context) {
	var req CreateTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid trade request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	candleID, err := uuid.Parse(req.CandleID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid candle_id"})
		return
	}

	trade, err := h.tradeService.CreateTrade(c.Request.Context(), services.CreateTradeInput{
		AccountID:      accountID,
		CandleID:       candleID,
		Bias:           req.Bias,
		PlannedEntry:   req.PlannedEntry,
		PlannedSL:      req.PlannedSL,
		PlannedTP:      req.PlannedTP,
		PlannedRiskPct: req.PlannedRiskPct,
		ReasonForTrade: req.ReasonForTrade,
	})
	if err != nil {
		log.Error().Err(err).Msg("create trade failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Str("trade_id", trade.ID.String()).
		Str("bias", trade.Bias).
		Str("rr", trade.PlannedRR).
		Msg("trade created")

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": trade})
}
