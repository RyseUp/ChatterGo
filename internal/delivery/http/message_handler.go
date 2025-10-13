package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/RyseUp/ChatterGo/internal/domain"
	"github.com/RyseUp/ChatterGo/internal/usecase"
	"github.com/RyseUp/ChatterGo/pkg/middleware"
)

type MessageHandler struct {
	messageUseCase usecase.MessageUseCase
}

func NewMessageHandler(messageUseCase usecase.MessageUseCase) *MessageHandler {
	return &MessageHandler{
		messageUseCase: messageUseCase,
	}
}

func (h *MessageHandler) SendMessage(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	message, err := h.messageUseCase.SendMessage(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, message)
}

func (h *MessageHandler) GetMessageHistory(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req domain.MessageHistoryRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	messages, err := h.messageUseCase.GetMessageHistory(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": messages})
}
