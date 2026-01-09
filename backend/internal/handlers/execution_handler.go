package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"set-and-trend/backend/internal/repositories"
	"set-and-trend/backend/internal/services"
)

type ExecutionHandler struct {
	executionService *services.ExecutionService
	execRepo         *repositories.ExecutionRepository
}

func NewExecutionHandler(
	executionService *services. ExecutionService,
	execRepo *repositories.ExecutionRepository,
) *ExecutionHandler {
	return &ExecutionHandler{
		executionService: executionService,
		execRepo:         execRepo,
	}
}

type ExecuteTradeRequest struct {
	ActualEntry float64 `json:"actual_entry" binding:"required,gt=0"`
	Reason      *string `json:"reason"`
}

func (h *ExecutionHandler) ExecuteTrade(c *gin.Context) {
	tradeID, err := uuid.Parse(c. Param("id"))
	if err != nil {
		c. JSON(http.StatusBadRequest, gin.H{"error": "invalid trade ID"})
		return
	}

	var req ExecuteTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.executionService.ExecuteTrade(c.Request.Context(), services.ExecuteTradeInput{
		TradeID:     tradeID,
		ActualEntry: req.ActualEntry,
		ExecutedAt:  time.Now().UTC(),
		Reason:      req.Reason,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	state, _ := h.executionService.GetTradeState(c. Request.Context(), tradeID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"trade_id": tradeID,
		"state":    state,
		"message": "trade executed successfully",
	})
}

type CloseTradeRequest struct {
	ClosePrice float64 `json:"close_price" binding:"required,gt=0"`
	Reason     *string `json:"reason"`
}

func (h *ExecutionHandler) CloseTrade(c *gin.Context) {
	tradeID, err := uuid.Parse(c. Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade ID"})
		return
	}

	var req CloseTradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h.executionService. CloseTrade(c.Request.Context(), services.CloseTradeInput{
		TradeID:    tradeID,
		ClosePrice: req.ClosePrice,
		ExecutedAt:  time.Now().UTC(),
		Reason:     req.Reason,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err. Error()})
		return
	}

	state, _ := h.executionService.GetTradeState(c.Request.Context(), tradeID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"trade_id": tradeID,
		"state":    state,
		"message":  "trade closed successfully",
	})
}

type CancelTradeRequest struct {
	Reason string `json:"reason" binding:"required,min=5"`
}

func (h *ExecutionHandler) CancelTrade(c *gin.Context) {
	tradeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin. H{"error": "invalid trade ID"})
		return
	}

	var req CancelTradeRequest
	if err := c. ShouldBindJSON(&req); err != nil {
		c. JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err = h. executionService.CancelTrade(c.Request.Context(), services.CancelTradeInput{
		TradeID:    tradeID,
		ExecutedAt: time.Now().UTC(),
		Reason:     req. Reason,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":  err.Error()})
		return
	}

	state, _ := h.executionService.GetTradeState(c.Request. Context(), tradeID)

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"trade_id": tradeID,
		"state":    state,
		"message": "trade cancelled successfully",
	})
}

func (h *ExecutionHandler) GetTradeState(c *gin. Context) {
	tradeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error":  "invalid trade ID"})
		return
	}

	state, err := h.executionService. GetTradeState(c.Request.Context(), tradeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get trade state"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"trade_id":  tradeID,
		"state":   state,
	})
}

func (h *ExecutionHandler) GetTradeExecutions(c *gin.Context) {
	tradeID, err := uuid.Parse(c. Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trade ID"})
		return
	}

	executions, err := h.execRepo.GetExecutionsByTradeID(c. Request.Context(), tradeID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get executions"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":     "success",
		"trade_id":   tradeID,
		"executions": executions,
		"count":      len(executions),
	})
}
