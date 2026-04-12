package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
)

type StrategyHandler struct {
	strategyRepo *repositories.StrategyRepository
	signalRepo   *repositories.SignalRepository
}

func NewStrategyHandler(
	strategyRepo *repositories.StrategyRepository,
	signalRepo *repositories.SignalRepository,
) *StrategyHandler {
	return &StrategyHandler{
		strategyRepo: strategyRepo,
		signalRepo:   signalRepo,
	}
}

// GET /api/strategies?enabled_only=true
func (h *StrategyHandler) ListStrategies(c *gin.Context) {
	enabledOnly := c.Query("enabled_only") == "true"
	out, err := h.strategyRepo.ListStrategies(c.Request.Context(), enabledOnly)
	if err != nil {
		log.Error().Err(err).Msg("list strategies failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": out})
}

// GET /api/strategies/:id/signals?limit=N
func (h *StrategyHandler) GetStrategySignals(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid strategy id"})
		return
	}
	limit := int32(50)
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}
	signals, err := h.signalRepo.ListByStrategy(c.Request.Context(), id, limit)
	if err != nil {
		log.Error().Err(err).Msg("list strategy signals failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": signals})
}

// GET /api/signals/active — unnotified signals (the setup-watcher poll target).
func (h *StrategyHandler) GetActiveSignals(c *gin.Context) {
	limit := int32(100)
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}
	signals, err := h.signalRepo.ListUnnotified(c.Request.Context(), limit)
	if err != nil {
		log.Error().Err(err).Msg("list unnotified signals failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": signals})
}

// POST /api/signals/:id/notified — called by setup-watcher after pushing to Telegram.
func (h *StrategyHandler) MarkSignalNotified(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal id"})
		return
	}
	if err := h.signalRepo.MarkNotified(c.Request.Context(), id); err != nil {
		log.Error().Err(err).Msg("mark signal notified failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
