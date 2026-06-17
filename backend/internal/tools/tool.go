package tools

import "context"

// Tool represents an external capability that Jarvis can execute through the registry.
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, input string) (string, error)
}

type Metadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
