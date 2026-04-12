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

type TradeHandler struct {
	tradeService *services.TradeService
	tradeRepo    *repositories.TradeRepository
	candleRepo   *repositories.CandleRepository
}

func NewTradeHandler(
	tradeService *services.TradeService,
	tradeRepo *repositories.TradeRepository,
	candleRepo *repositories.CandleRepository,
) *TradeHandler {
	return &TradeHandler{
		tradeService: tradeService,
		tradeRepo:    tradeRepo,
		candleRepo:   candleRepo,
	}
}

type CreateTradeRequest struct {
	AccountID      string  `json:"account_id" binding:"required,uuid"`
	CandleID       string  `json:"candle_id" binding:"required,uuid"`
	Direction      string  `json:"direction" binding:"required,oneof=LONG SHORT long short"`
	PlannedEntry   float64 `json:"planned_entry" binding:"required,gt=0"`
	PlannedSL      float64 `json:"planned_sl" binding:"required,gt=0"`
	PlannedTP      float64 `json:"planned_tp" binding:"required,gt=0"`
	PlannedRiskPct float64 `json:"planned_risk_pct" binding:"required,gt=0,lte=100"`
	Symbol         string  `json:"symbol" binding:"required"`
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

	trade, err := h.tradeService.CreateTrade(
		c.Request.Context(),
		services.CreateTradeInput{
			AccountID:      accountID,
			CandleID:       candleID,
			Direction:      req.Direction,
			PlannedEntry:   req.PlannedEntry,
			PlannedSL:      req.PlannedSL,
			PlannedTP:      req.PlannedTP,
			PlannedRiskPct: req.PlannedRiskPct,
			Symbol:         req.Symbol,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("create trade failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Str("trade_id", trade.ID.String()).
		Str("direction", trade.Direction).
		Msg("trade created")

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": trade})
}

// CreateSimpleTradeRequest is the conversational/Telegram path. The user logs
// a trade against a strategy without picking a specific candle — the handler
// auto-resolves the most recent H4 candle for that symbol.
type CreateSimpleTradeRequest struct {
	AccountID      string  `json:"account_id" binding:"required,uuid"`
	Symbol         string  `json:"symbol" binding:"required"`
	Direction      string  `json:"direction" binding:"required,oneof=LONG SHORT long short"`
	PlannedEntry   float64 `json:"planned_entry" binding:"required,gt=0"`
	PlannedSL      float64 `json:"planned_sl" binding:"required,gt=0"`
	PlannedTP      float64 `json:"planned_tp" binding:"required,gt=0"`
	PlannedRiskPct float64 `json:"planned_risk_pct" binding:"required,gt=0,lte=100"`
}

func (h *TradeHandler) CreateSimpleTrade(c *gin.Context) {
	var req CreateSimpleTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	accountID, err := uuid.Parse(req.AccountID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid account_id"})
		return
	}

	candle, err := h.candleRepo.GetLatestH4Candle(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("resolve latest H4 candle failed")
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "no recent H4 candle available"})
		return
	}

	trade, err := h.tradeService.CreateSimpleTrade(
		c.Request.Context(),
		services.CreateSimpleTradeInput{
			AccountID:      accountID,
			CandleID:       candle.ID,
			Symbol:         req.Symbol,
			Timeframe:      "H4",
			Direction:      req.Direction,
			PlannedEntry:   req.PlannedEntry,
			PlannedSL:      req.PlannedSL,
			PlannedTP:      req.PlannedTP,
			PlannedRiskPct: req.PlannedRiskPct,
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("create simple trade failed")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Info().
		Str("trade_id", trade.ID.String()).
		Str("strategy_account", accountID.String()).
		Msg("simple trade created")

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": trade})
}

// ListTrades returns the most recent N trades for a user. Defaults to 50 if
// `limit` is omitted. user_id is required (matches the analytics handler
// pattern of explicit query params rather than JWT-context resolution).
func (h *TradeHandler) ListTrades(c *gin.Context) {
	userIDStr := c.Query("user_id")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_id required"})
		return
	}
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	limit := int32(50)
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}

	trades, err := h.tradeRepo.GetTradesByUserID(c.Request.Context(), userID, limit)
	if err != nil {
		log.Error().Err(err).Msg("list trades failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": trades})
}
