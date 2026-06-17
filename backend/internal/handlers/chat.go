package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/services"
)

const maxChatMessageLength = 8000

type ChatHandler struct {
	service *services.ChatService
}

func NewChatHandler(service *services.ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

func (h *ChatHandler) Chat(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_required"})
		return
	}
	if len(message) > maxChatMessageLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_too_long"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 60*time.Second)
	defer cancel()

	start := time.Now()
	log.Printf("chat request received: message_length=%d", len(message))

	response, err := h.service.SendMessage(ctx, message)
	if err != nil {
		log.Printf("chat request failed after %s: %v", time.Since(start), err)
		c.JSON(http.StatusBadGateway, gin.H{"error": "llm_unavailable"})
		return
	}

	log.Printf("chat response generated in %s: response_length=%d", time.Since(start), len(response))
	c.JSON(http.StatusOK, chatResponse{Response: response})
}

func (h *ChatHandler) Stream(c *gin.Context) {
	var req chatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_required"})
		return
	}
	if len(message) > maxChatMessageLength {
		c.JSON(http.StatusBadRequest, gin.H{"error": "message_too_long"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	start := time.Now()
	log.Printf("chat stream request received: message_length=%d", len(message))

	var responseLength int
	err := h.service.StreamMessage(ctx, message, func(chunk string) error {
		responseLength += len(chunk)
		return writeSSE(c, "delta", streamDelta{Delta: chunk})
	})
	if err != nil {
		log.Printf("chat stream failed after %s: %v", time.Since(start), err)
		_ = writeSSE(c, "error", streamError{Error: "llm_unavailable"})
		return
	}

	log.Printf("chat stream completed in %s: response_length=%d", time.Since(start), responseLength)
	_ = writeSSE(c, "done", streamDone{Done: true})
}

func writeSSE(c *gin.Context, event string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}

	c.Writer.Flush()
	return nil
}

type chatRequest struct {
	Message string `json:"message"`
}

type chatResponse struct {
	Response string `json:"response"`
}

type streamDelta struct {
	Delta string `json:"delta"`
}

type streamDone struct {
	Done bool `json:"done"`
}

type streamError struct {
	Error string `json:"error"`
}
