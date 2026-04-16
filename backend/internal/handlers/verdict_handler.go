package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
)

type VerdictHandler struct {
	verdictRepo *repositories.VerdictRepository
	signalRepo  *repositories.SignalRepository
}

func NewVerdictHandler(
	verdictRepo *repositories.VerdictRepository,
	signalRepo *repositories.SignalRepository,
) *VerdictHandler {
	return &VerdictHandler{verdictRepo: verdictRepo, signalRepo: signalRepo}
}

type createVerdictRequest struct {
	AgentID    string          `json:"agent_id" binding:"required"`
	Verdict    string          `json:"verdict" binding:"required"`
	Confidence *float64        `json:"confidence,omitempty"`
	Reason     *string         `json:"reason,omitempty"`
	RawOutput  json.RawMessage `json:"raw_output,omitempty"`
}

// POST /api/signals/:id/verdicts — specialist agents write their row here.
func (h *VerdictHandler) CreateVerdict(c *gin.Context) {
	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal id"})
		return
	}
	var req createVerdictRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	v, err := h.verdictRepo.CreateVerdict(c.Request.Context(), repositories.CreateVerdictParams{
		SignalID:   signalID,
		AgentID:    req.AgentID,
		Verdict:    req.Verdict,
		Confidence: req.Confidence,
		Reason:     req.Reason,
		RawOutput:  req.RawOutput,
	})
	if err != nil {
		log.Error().Err(err).Msg("create verdict failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": v})
}

// GET /api/signals/:id/verdicts — composer reads all verdicts for a signal.
func (h *VerdictHandler) ListVerdicts(c *gin.Context) {
	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal id"})
		return
	}
	out, err := h.verdictRepo.ListBySignal(c.Request.Context(), signalID)
	if err != nil {
		log.Error().Err(err).Msg("list verdicts failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": out})
}

// GET /api/signals/unprocessed?older_than_sec=30&limit=50 — composer's poll target.
func (h *VerdictHandler) ListUnprocessedSignals(c *gin.Context) {
	olderThan := 30
	if v := c.Query("older_than_sec"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 && n <= 3600 {
			olderThan = n
		}
	}
	limit := int32(50)
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 500 {
			limit = int32(n)
		}
	}
	out, err := h.signalRepo.ListUnprocessed(c.Request.Context(), olderThan, limit)
	if err != nil {
		log.Error().Err(err).Msg("list unprocessed failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success", "data": out})
}

type markComposedRequest struct {
	AISummary string `json:"ai_summary" binding:"required"`
}

// POST /api/signals/:id/composed — composer stamps signal after Telegram push.
func (h *VerdictHandler) MarkComposed(c *gin.Context) {
	signalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid signal id"})
		return
	}
	var req markComposedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.signalRepo.MarkComposed(c.Request.Context(), signalID, req.AISummary); err != nil {
		log.Error().Err(err).Msg("mark composed failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
