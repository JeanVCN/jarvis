package ai

import "context"

type LLM interface {
	Chat(ctx context.Context, prompt string) (string, error)
	Stream(ctx context.Context, prompt string, onChunk func(chunk string) error) error
}
