package tools

import (
	"context"
	"testing"
)

func TestRegistryExecute(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register(NewCalculatorTool()); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	result, err := registry.Execute(context.Background(), "calculator", "2 + 3 * 4")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "14" {
		t.Fatalf("expected 14, got %q", result)
	}
}

func TestRegistryListIsSorted(t *testing.T) {
	t.Parallel()

	registry := NewRegistry()
	if err := registry.Register(NewTimeTool()); err != nil {
		t.Fatalf("Register time returned error: %v", err)
	}
	if err := registry.Register(NewCalculatorTool()); err != nil {
		t.Fatalf("Register calculator returned error: %v", err)
	}

	tools := registry.List()
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}
	if tools[0].Name != "calculator" || tools[1].Name != "time" {
		t.Fatalf("expected sorted tools, got %#v", tools)
	}
}
