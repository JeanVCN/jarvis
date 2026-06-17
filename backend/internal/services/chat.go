package services

import (
	"context"
	"strings"

	"github.com/jean/jarvis/backend/internal/ai"
)

type ChatService struct {
	llm ai.LLM
}

func NewChatService(llm ai.LLM) *ChatService {
	return &ChatService{llm: llm}
}

func (s *ChatService) SendMessage(ctx context.Context, message string) (string, error) {
	return s.llm.Chat(ctx, strings.TrimSpace(message))
}

func (s *ChatService) StreamMessage(ctx context.Context, message string, onChunk func(chunk string) error) error {
	return s.llm.Stream(ctx, strings.TrimSpace(message), onChunk)
}
