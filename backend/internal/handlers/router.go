package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/database"
	"github.com/jean/jarvis/backend/internal/middleware"
	"github.com/jean/jarvis/backend/internal/services"
)

func NewRouter(db *database.Postgres, chatService *services.ChatService, toolService *services.ToolService, voiceService *services.VoiceService, corsAllowedOrigin string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.CORS(corsAllowedOrigin))

	healthHandler := NewHealthHandler(db)
	router.GET("/health", healthHandler.Check)

	chatHandler := NewChatHandler(chatService)
	router.POST("/api/chat", chatHandler.Chat)
	router.POST("/api/chat/stream", chatHandler.Stream)

	toolHandler := NewToolHandler(toolService)
	router.GET("/api/tools", toolHandler.List)
	router.POST("/api/tools/execute", toolHandler.Execute)

	voiceHandler := NewVoiceHandler(voiceService)
	router.GET("/api/voice/status", voiceHandler.Status)
	router.POST("/api/voice/conversation", voiceHandler.Converse)

	return router
}
