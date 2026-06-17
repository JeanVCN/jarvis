package services

import (
	"context"
	"strings"

	"github.com/jean/jarvis/backend/internal/tools"
)

type ToolService struct {
	registry *tools.Registry
}

func NewToolService(registry *tools.Registry) *ToolService {
	return &ToolService{registry: registry}
}

func (s *ToolService) ListTools() []tools.Metadata {
	return s.registry.List()
}

func (s *ToolService) ExecuteTool(ctx context.Context, name string, input string) (string, error) {
	return s.registry.Execute(ctx, strings.TrimSpace(name), input)
}
