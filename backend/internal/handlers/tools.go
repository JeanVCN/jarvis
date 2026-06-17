package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/services"
	"github.com/jean/jarvis/backend/internal/tools"
)

const maxToolInputLength = 4000

type ToolHandler struct {
	service *services.ToolService
}

func NewToolHandler(service *services.ToolService) *ToolHandler {
	return &ToolHandler{service: service}
}

func (h *ToolHandler) List(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.ListTools())
}

func (h *ToolHandler) Execute(c *gin.Context) {
	var req toolExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	toolName := strings.TrimSpace(req.Tool)
	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tool_required"})
		return
	}
	if len(req.Input) > maxToolInputLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "input_too_long"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	start := time.Now()
	log.Printf("tool execution requested: tool=%s input_length=%d", toolName, len(req.Input))

	result, err := h.service.ExecuteTool(ctx, toolName, req.Input)
	if err != nil {
		log.Printf("tool execution failed after %s: tool=%s error=%v", time.Since(start), toolName, err)
		if errors.Is(err, tools.ErrToolNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "tool_not_found"})
			return
		}
		if errors.Is(err, tools.ErrInvalidExpression) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_tool_input"})
			return
		}
		if errors.Is(err, tools.ErrSystemActionNotAllowed) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_tool_input"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "tool_execution_failed"})
		return
	}

	log.Printf("tool execution completed in %s: tool=%s result_length=%d", time.Since(start), toolName, len(result))
	c.JSON(http.StatusOK, toolExecuteResponse{Result: result})
}

type toolExecuteRequest struct {
	Tool  string `json:"tool"`
	Input string `json:"input"`
}

type toolExecuteResponse struct {
	Result string `json:"result"`
}
