package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/database"
	"github.com/jean/jarvis/backend/internal/middleware"
	"github.com/jean/jarvis/backend/internal/services"
	"strings"
)

func NewRouter(db *database.Postgres, chatService *services.ChatService, toolService *services.ToolService, voiceService *services.VoiceService, corsAllowedOrigin string) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.Use(middleware.CORS(corsAllowedOrigin))
	router.Use(func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin != "" && c.Request.Method != "GET" && c.Request.Method != "OPTIONS" {
			allowed := false
			for _, item := range strings.Split(corsAllowedOrigin, ",") {
				if strings.TrimSpace(item) == origin {
					allowed = true
				}
			}
			if !allowed {
				c.AbortWithStatusJSON(403, gin.H{"error": "origin_denied"})
				return
			}
		}
		c.Next()
	})

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
