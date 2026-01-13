package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/services"
)

type SignalHandler struct {
	signalService *services.SignalScannerService
}

func NewSignalHandler(signalService *services.SignalScannerService) *SignalHandler {
	return &SignalHandler{signalService: signalService}
}

// GetLatestSignal returns the current market signal
func (h *SignalHandler) GetLatestSignal(c *gin.Context) {
	signal, err := h.signalService.ScanLatest(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to scan for signals")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to scan for signals"})
		return
	}

	// Determine action based on signal
	action := "WAIT"
	if signal.Bias == "long" && len(signal.RulesPassed) > 0 && signal.Confidence >= 0.8 {
		action = "BUY"
	} else if signal.Bias == "short" && len(signal.RulesPassed) > 0 && signal.Confidence >= 0.8 {
		action = "SELL"
	}

	log.Info().
		Str("bias", signal.Bias).
		Float64("confidence", signal.Confidence).
		Int("rules_passed", len(signal.RulesPassed)).
		Str("action", action).
		Msg("signal scanned")

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"signal": signal,
			"action": action,
			"message": getActionMessage(action, signal),
		},
	})
}

func getActionMessage(action string, signal *services.Signal) string {
	switch action {
	case "BUY":
		return "🟢 Strong bullish signal detected! Consider entering a LONG position."
	case "SELL":
		return "🔴 Strong bearish signal detected! Consider entering a SHORT position."
	default:
		return "⚪ No clear signal. Wait for better setup."
	}
}
