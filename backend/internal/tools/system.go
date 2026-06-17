package tools

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

var ErrSystemActionNotAllowed = errors.New("system action not allowed")

type SystemAction struct {
	Name        string
	Description string
	Command     string
	Args        []string
}

type SystemTool struct {
	actions map[string]SystemAction
	runner  commandRunner
}

type commandRunner interface {
	Run(ctx context.Context, command string, args ...string) error
}

type execCommandRunner struct{}

func NewSystemTool(actions []SystemAction) *SystemTool {
	actionMap := make(map[string]SystemAction, len(actions))
	for _, action := range actions {
		if action.Name == "" || action.Command == "" {
			continue
		}
		actionMap[action.Name] = action
	}

	return &SystemTool{
		actions: actionMap,
		runner:  execCommandRunner{},
	}
}

func NewDefaultSystemTool() *SystemTool {
	return NewSystemTool(DefaultSystemActions())
}

func DefaultSystemActions() []SystemAction {
	return []SystemAction{
		{
			Name:        "open_browser",
			Description: "Abre o navegador padrao",
			Command:     "xdg-open",
			Args:        []string{"about:blank"},
		},
		{
			Name:        "open_vscode",
			Description: "Abre o VSCode",
			Command:     "code",
		},
		{
			Name:        "open_terminal",
			Description: "Abre o terminal padrao",
			Command:     "x-terminal-emulator",
		},
		{
			Name:        "open_spotify",
			Description: "Abre o Spotify se estiver instalado",
			Command:     "spotify",
		},
		{
			Name:        "lock_screen",
			Description: "Bloqueia a sessao atual",
			Command:     "loginctl",
			Args:        []string{"lock-session"},
		},
	}
}

func (t *SystemTool) Name() string {
	return "system"
}

func (t *SystemTool) Description() string {
	return fmt.Sprintf("Executa acoes controladas do sistema operacional. Acoes permitidas: %s", strings.Join(t.actionNames(), ", "))
}

func (t *SystemTool) Execute(ctx context.Context, input string) (string, error) {
	actionName := strings.TrimSpace(input)
	action, ok := t.actions[actionName]
	if !ok {
		return "", fmt.Errorf("%w: %s", ErrSystemActionNotAllowed, actionName)
	}

	if err := t.runner.Run(ctx, action.Command, action.Args...); err != nil {
		return "", fmt.Errorf("execute system action %s: %w", actionName, err)
	}

	return fmt.Sprintf("Acao %s executada com sucesso.", actionName), nil
}

func (t *SystemTool) actionNames() []string {
	names := make([]string, 0, len(t.actions))
	for name := range t.actions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r execCommandRunner) Run(ctx context.Context, command string, args ...string) error {
	cmd := exec.CommandContext(ctx, command, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}
