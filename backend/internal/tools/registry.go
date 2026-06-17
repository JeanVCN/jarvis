package tools

import (
	"context"
	"errors"
	"fmt"
	"sort"
)

var (
	ErrToolAlreadyRegistered = errors.New("tool already registered")
	ErrToolNotFound          = errors.New("tool not found")
	ErrInvalidTool           = errors.New("invalid tool")
)

type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(tool Tool) error {
	if tool == nil || tool.Name() == "" {
		return ErrInvalidTool
	}

	name := tool.Name()
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("%w: %s", ErrToolAlreadyRegistered, name)
	}

	r.tools[name] = tool
	return nil
}

func (r *Registry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

func (r *Registry) List() []Metadata {
	items := make([]Metadata, 0, len(r.tools))
	for _, tool := range r.tools {
		items = append(items, Metadata{
			Name:        tool.Name(),
			Description: tool.Description(),
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items
}

func (r *Registry) Execute(ctx context.Context, name string, input string) (string, error) {
	tool, ok := r.Get(name)
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrToolNotFound, name)
	}

	return tool.Execute(ctx, input)
}
