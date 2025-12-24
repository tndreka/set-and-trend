package handlers

import (
	"net/http"
	
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"set-and-trend/backend/internal/repositories"
)

type UserHandler struct {
	userRepo *repositories.UserRepository
}

func NewUserHandler(userRepo *repositories.UserRepository) *UserHandler {
	return &UserHandler{userRepo: userRepo}
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	id := uuid.New()
	
	user, err := h.userRepo.CreateUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": user})
}
