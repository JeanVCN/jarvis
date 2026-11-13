package handlers

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jean/jarvis/backend/internal/services"
	"github.com/jean/jarvis/backend/internal/voice"
)

func TestVoiceHandlerConverse(t *testing.T) {
	t.Parallel()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("audio", "speech.webm")
	if err != nil {
		t.Fatalf("CreateFormFile returned error: %v", err)
	}
	if _, err := part.Write([]byte("audio")); err != nil {
		t.Fatalf("write multipart returned error: %v", err)
	}
	if err := writer.WriteField("language", "pt"); err != nil {
		t.Fatalf("WriteField returned error: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer close returned error: %v", err)
	}

	handler := NewVoiceHandler(services.NewVoiceService(
		handlerFakeRecognizer{},
		handlerFakeVoiceProvider{},
		services.NewChatService(&handlerFakeLLM{}, newHandlerVoiceToolService(t)),
	))
	router := gin.New()
	router.POST("/api/voice/conversation", handler.Converse)

	req := httptest.NewRequest(http.MethodPost, "/api/voice/conversation", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte(`"transcript":"Ola Jarvis"`)) {
		t.Fatalf("expected transcript in response: %s", res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte(`"audioBase64":"YXVkaW8="`)) {
		t.Fatalf("expected audio in response: %s", res.Body.String())
	}
}

func TestVoiceHandlerStatus(t *testing.T) {
	t.Parallel()

	handler := NewVoiceHandler(services.NewVoiceService(
		handlerFakeRecognizer{},
		handlerFakeVoiceProvider{},
		services.NewChatService(&handlerFakeLLM{}, newHandlerVoiceToolService(t)),
	))
	router := gin.New()
	router.GET("/api/voice/status", handler.Status)

	req := httptest.NewRequest(http.MethodGet, "/api/voice/status", nil)
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", res.Code, res.Body.String())
	}
	if !bytes.Contains(res.Body.Bytes(), []byte(`"message":"voice_ready"`)) {
		t.Fatalf("expected voice_ready status: %s", res.Body.String())
	}
}

type handlerFakeRecognizer struct{}

func (r handlerFakeRecognizer) Recognize(ctx context.Context, input voice.AudioInput) (string, error) {
	return "Ola Jarvis", nil
}

func (r handlerFakeRecognizer) Available() bool {
	return true
}

func (r handlerFakeRecognizer) Command() string {
	return "fake-stt"
}

type handlerFakeVoiceProvider struct{}

func (p handlerFakeVoiceProvider) Synthesize(ctx context.Context, text string) (voice.SpeechOutput, error) {
	return voice.SpeechOutput{Audio: []byte("audio"), ContentType: "audio/wav"}, nil
}

func (p handlerFakeVoiceProvider) Available() bool {
	return true
}

func (p handlerFakeVoiceProvider) Command() string {
	return "fake-tts"
}

type handlerFakeLLM struct{}

func (l *handlerFakeLLM) Chat(ctx context.Context, prompt string) (string, error) {
	return "Resposta falada", nil
}

func (l *handlerFakeLLM) Stream(ctx context.Context, prompt string, onChunk func(chunk string) error) error {
	return onChunk("Resposta falada")
}

func newHandlerVoiceToolService(t *testing.T) *services.ToolService {
	t.Helper()
	return newToolTestService(t)
}
