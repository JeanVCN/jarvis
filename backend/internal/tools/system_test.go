package tools

import (
	"context"
	"testing"
)

func TestSystemToolExecuteAllowedAction(t *testing.T) {
	t.Parallel()

	runner := &fakeCommandRunner{}
	tool := NewSystemTool([]SystemAction{
		{
			Name:    "open_vscode",
			Command: "code",
		},
	})
	tool.runner = runner

	result, err := tool.Execute(context.Background(), "open_vscode")
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result != "Acao open_vscode executada com sucesso." {
		t.Fatalf("unexpected result: %s", result)
	}
	if runner.command != "code" {
		t.Fatalf("expected code command, got %q", runner.command)
	}
}

func TestSystemToolRejectsUnknownAction(t *testing.T) {
	t.Parallel()

	tool := NewSystemTool([]SystemAction{
		{
			Name:    "open_vscode",
			Command: "code",
		},
	})
	tool.runner = &fakeCommandRunner{}

	if _, err := tool.Execute(context.Background(), "rm -rf /"); err == nil {
		t.Fatal("expected unknown action error")
	}
}

type fakeCommandRunner struct {
	command string
	args    []string
}

func (r *fakeCommandRunner) Run(ctx context.Context, command string, args ...string) error {
	r.command = command
	r.args = args
	return nil
}
