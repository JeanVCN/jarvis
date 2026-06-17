package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/database"
	"github.com/jean/jarvis/backend/internal/middleware"
	"github.com/jean/jarvis/backend/internal/services"
)

func NewRouter(db *database.Postgres, chatService *services.ChatService, corsAllowedOrigin string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.CORS(corsAllowedOrigin))

	healthHandler := NewHealthHandler(db)
	router.GET("/health", healthHandler.Check)

	chatHandler := NewChatHandler(chatService)
	router.POST("/api/chat", chatHandler.Chat)
	router.POST("/api/chat/stream", chatHandler.Stream)

	return router
}
