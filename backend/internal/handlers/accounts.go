package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"set-and-trend/backend/internal/repositories"
)

type AccountHandler struct {
	accountRepo *repositories.AccountRepository
	userRepo    *repositories.UserRepository
}

func NewAccountHandler(accountRepo *repositories.AccountRepository, userRepo *repositories.UserRepository) *AccountHandler {
	return &AccountHandler{
		accountRepo: accountRepo,
		userRepo:    userRepo,
	}
}

type CreateAccountRequest struct {
	UserID                 string  `json:"user_id" binding:"required,uuid"`
	Type                   string  `json:"type" binding:"required,oneof=demo live"`
	BrokerName             string  `json:"broker_name" binding:"required,min=1,max=50"`
	Currency               string  `json:"currency" binding:"required,len=3,uppercase"`
	Balance                string  `json:"balance" binding:"required"`
	Leverage               int     `json:"leverage" binding:"required,gt=0,lte=1000"`
	MaxRiskPerTradePct     float64 `json:"max_risk_per_trade_pct" binding:"required,gte=0,lte=100"`
	MaxDailyRiskPct        float64 `json:"max_daily_risk_pct" binding:"required,gte=0,lte=100"`
	Timezone               string  `json:"timezone" binding:"required,max=50"`
	PreferredSession       string  `json:"preferred_session" binding:"required,oneof=london new_york asian custom"`
}

func (h *AccountHandler) CreateAccount(c *gin.Context) {
	var req CreateAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Warn().Err(err).Msg("invalid account request")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	// User existence check
	_, err = h.userRepo.GetUser(c.Request.Context(), userID)
	if err != nil {
		log.Warn().Str("user_id", userID.String()).Msg("user not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	// Timezone validation
	if _, err := time.LoadLocation(req.Timezone); err != nil {
		log.Warn().Str("timezone", req.Timezone).Msg("invalid timezone")
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid timezone (use IANA format)"})
		return
	}

	// ✅ Match repository params exactly
	account, err := h.accountRepo.CreateAccount(c.Request.Context(), repositories.AccountCreateParams{
		ID:                     uuid.New(),
		UserID:                 userID,
		Type:                   req.Type,
		BrokerName:             req.BrokerName,
		Currency:               req.Currency,
		Balance:                req.Balance,
		Leverage:               int32(req.Leverage),
		MaxRiskPerTradePct:     req.MaxRiskPerTradePct,
		MaxDailyRiskPct:        req.MaxDailyRiskPct,
		Timezone:               req.Timezone,
		PreferredSession:       req.PreferredSession,
	})
	if err != nil {
		log.Error().Err(err).Msg("create account failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.Info().
		Str("account_id", account.ID.String()).
		Str("user_id", account.UserID.String()).
		Msg("account created")

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": account})
}
