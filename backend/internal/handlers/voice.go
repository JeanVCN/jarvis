package handlers

import (
	"context"
	"encoding/base64"
	"errors"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/services"
	"github.com/jean/jarvis/backend/internal/voice"
)

const maxVoiceUploadBytes = 15 << 20

type VoiceHandler struct {
	service *services.VoiceService
}

func NewVoiceHandler(service *services.VoiceService) *VoiceHandler {
	return &VoiceHandler{service: service}
}

func (h *VoiceHandler) Status(c *gin.Context) {
	c.JSON(http.StatusOK, h.service.Status())
}

func (h *VoiceHandler) Converse(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(maxVoiceUploadBytes); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_multipart"})
		return
	}

	file, header, err := c.Request.FormFile("audio")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_required"})
		return
	}
	defer file.Close()

	if header.Size > maxVoiceUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_too_large"})
		return
	}

	audio, err := io.ReadAll(io.LimitReader(file, maxVoiceUploadBytes+1))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_read_failed"})
		return
	}
	if len(audio) > maxVoiceUploadBytes {
		c.JSON(http.StatusBadRequest, gin.H{"error": "audio_too_large"})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Minute)
	defer cancel()

	start := time.Now()
	log.Printf("voice conversation received: audio_bytes=%d content_type=%s", len(audio), header.Header.Get("Content-Type"))

	result, err := h.service.Converse(ctx, voice.AudioInput{
		Data:         audio,
		Format:       header.Header.Get("Content-Type"),
		Language:     c.PostForm("language"),
		WakeRequired: c.PostForm("wakeRequired") == "true",
	})
	if err != nil {
		log.Printf("voice conversation failed after %s: %v", time.Since(start), err)
		if errors.Is(err, voice.ErrVoiceProviderUnavailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "voice_provider_unavailable"})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": "voice_conversation_failed"})
		return
	}

	log.Printf("voice conversation completed in %s: transcript_length=%d response_length=%d audio_bytes=%d", time.Since(start), len(result.Transcript), len(result.Response), len(result.Audio))
	c.JSON(http.StatusOK, voiceConversationResponse{
		Transcript:       result.Transcript,
		Response:         result.Response,
		AudioContentType: result.AudioContentType,
		AudioBase64:      base64.StdEncoding.EncodeToString(result.Audio),
		Ignored:          result.Ignored,
		WakeWordDetected: result.WakeWordDetected,
	})
}

type voiceConversationResponse struct {
	Transcript       string `json:"transcript"`
	Response         string `json:"response"`
	AudioContentType string `json:"audioContentType"`
	AudioBase64      string `json:"audioBase64"`
	Ignored          bool   `json:"ignored"`
	WakeWordDetected bool   `json:"wakeWordDetected"`
}
