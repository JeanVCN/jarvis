package tools

import (
	"context"
	"time"
)

type TimeTool struct {
	now func() time.Time
}

func NewTimeTool() *TimeTool {
	return &TimeTool{now: time.Now}
}

func (t *TimeTool) Name() string {
	return "time"
}

func (t *TimeTool) Description() string {
	return "Retorna data e hora atual"
}

func (t *TimeTool) Execute(ctx context.Context, input string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return t.now().Format("2006-01-02 15:04:05"), nil
	}
}
