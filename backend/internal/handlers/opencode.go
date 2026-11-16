package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/opencode"
)

func RegisterOpenCode(router *gin.Engine, client *opencode.Client) {
	router.GET("/api/opencode/status", func(c *gin.Context) {
		result, err := client.Status(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "opencode_unavailable"})
			return
		}
		c.JSON(http.StatusOK, result)
	})
	router.GET("/api/opencode/projects", func(c *gin.Context) {
		result, err := client.Projects(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "opencode_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"projects": result})
	})
	router.GET("/api/opencode/projects/:project/sessions", func(c *gin.Context) {
		result, err := client.Sessions(c.Request.Context(), c.Param("project"))
		if err != nil {
			if err.Error() == "invalid_project_id" {
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_project_id"})
				return
			}
			c.JSON(http.StatusBadGateway, gin.H{"error": "opencode_unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"sessions": result})
	})
}
