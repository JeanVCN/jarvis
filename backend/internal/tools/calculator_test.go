package tools

import (
	"context"
	"testing"
)

func TestCalculatorToolExecute(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"25 * 4":       "100",
		"(10 + 5) * 2": "30",
		"10 / 4":       "2.5",
		"-2 + 8":       "6",
	}

	tool := NewCalculatorTool()
	for input, expected := range tests {
		input := input
		expected := expected
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			result, err := tool.Execute(context.Background(), input)
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if result != expected {
				t.Fatalf("expected %q, got %q", expected, result)
			}
		})
	}
}

func TestCalculatorToolExecuteRejectsInvalidExpression(t *testing.T) {
	t.Parallel()

	tool := NewCalculatorTool()
	if _, err := tool.Execute(context.Background(), "10 / 0"); err == nil {
		t.Fatal("expected division by zero error")
	}
}
