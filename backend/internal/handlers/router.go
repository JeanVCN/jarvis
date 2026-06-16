package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/database"
)

func NewRouter(db *database.Postgres) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	healthHandler := NewHealthHandler(db)
	router.GET("/health", healthHandler.Check)

	return router
}
