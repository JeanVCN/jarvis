package services

import (
	"context"
	"strings"
	"testing"

	"github.com/jean/jarvis/backend/internal/tools"
)

func TestChatServiceReturnsNormalLLMResponse(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{responses: []string{"Ola! Como posso ajudar?"}}
	service := NewChatService(llm, newChatTestToolService(t))

	response, err := service.SendMessage(context.Background(), "Ola Jarvis")
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if response != "Ola! Como posso ajudar?" {
		t.Fatalf("unexpected response: %s", response)
	}
	if len(llm.prompts) != 1 {
		t.Fatalf("expected 1 llm call, got %d", len(llm.prompts))
	}
	if llm.prompts[0] != "Ola Jarvis" {
		t.Fatalf("expected direct prompt, got %q", llm.prompts[0])
	}
}

func TestChatServiceExecutesCalculatorToolCall(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{}
	service := NewChatService(llm, newChatTestToolService(t))

	response, err := service.SendMessage(context.Background(), "Quanto e 25 vezes 4?")
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if response != "O resultado e 100." {
		t.Fatalf("unexpected response: %s", response)
	}
	if len(llm.prompts) != 0 {
		t.Fatalf("expected no llm calls, got %d", len(llm.prompts))
	}
}

func TestChatServiceExecutesTimeToolCall(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{}
	service := NewChatService(llm, newChatTestToolService(t))

	response, err := service.SendMessage(context.Background(), "Que horas sao?")
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if !strings.HasPrefix(response, "Agora: ") {
		t.Fatalf("unexpected response: %s", response)
	}
	if len(llm.prompts) != 0 {
		t.Fatalf("expected no llm calls, got %d", len(llm.prompts))
	}
}

func TestChatServiceExecutesSystemToolCall(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{}
	service := NewChatService(llm, newChatTestToolService(t))

	response, err := service.SendMessage(context.Background(), "Abra o VSCode")
	if err != nil {
		t.Fatalf("SendMessage returned error: %v", err)
	}
	if response != "Pronto, abri o VSCode." {
		t.Fatalf("unexpected response: %s", response)
	}
	if len(llm.prompts) != 0 {
		t.Fatalf("expected no llm calls, got %d", len(llm.prompts))
	}
}

func TestChatServiceDetectsSpotifyVoiceCommands(t *testing.T) {
	t.Parallel()

	service := NewChatService(&fakeLLM{}, newChatTestToolService(t))
	cases := []string{
		"abrir o Spotify",
		"abre spotfy",
		"coloque o spotifai",
		"abra scottiff",
		"spotify",
	}

	for _, input := range cases {
		call, ok := service.detectToolCall(input)
		if !ok {
			t.Fatalf("expected tool call for %q", input)
		}
		if call.Tool != "system" || call.Input != "open_spotify" {
			t.Fatalf("unexpected tool call for %q: %#v", input, call)
		}
	}
}

func TestChatServiceDoesNotOpenSpotifyForQuestions(t *testing.T) {
	t.Parallel()

	service := NewChatService(&fakeLLM{}, newChatTestToolService(t))

	call, ok := service.detectToolCall("O que e Spotify?")
	if ok {
		t.Fatalf("unexpected tool call: %#v", call)
	}
}

func TestChatServiceStreamsNormalMessagesDirectly(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{streamResponses: []string{"Ola", " Jean"}}
	service := NewChatService(llm, newChatTestToolService(t))

	var chunks []string
	err := service.StreamMessage(context.Background(), "Ola Jarvis", func(chunk string) error {
		chunks = append(chunks, chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamMessage returned error: %v", err)
	}
	if strings.Join(chunks, "") != "Ola Jean" {
		t.Fatalf("unexpected stream chunks: %#v", chunks)
	}
	if len(llm.prompts) != 0 {
		t.Fatalf("expected no blocking chat calls, got %d", len(llm.prompts))
	}
	if len(llm.streamPrompts) != 1 || llm.streamPrompts[0] != "Ola Jarvis" {
		t.Fatalf("expected direct stream prompt, got %#v", llm.streamPrompts)
	}
}

func TestChatServiceDetectsCalculatorExpression(t *testing.T) {
	t.Parallel()

	service := NewChatService(&fakeLLM{}, newChatTestToolService(t))

	call, ok := service.detectToolCall("Calcule (10 + 5) * 2")
	if !ok {
		t.Fatal("expected tool call")
	}
	if call.Tool != "calculator" || call.Input != "(10 + 5) * 2" {
		t.Fatalf("unexpected tool call: %#v", call)
	}
}

func newChatTestToolService(t *testing.T) *ToolService {
	t.Helper()

	registry := tools.NewRegistry()
	if err := registry.Register(tools.NewCalculatorTool()); err != nil {
		t.Fatalf("register calculator: %v", err)
	}
	if err := registry.Register(newFakeTool("system", "Executa acoes controladas do sistema operacional.")); err != nil {
		t.Fatalf("register system: %v", err)
	}
	if err := registry.Register(tools.NewTimeTool()); err != nil {
		t.Fatalf("register time: %v", err)
	}
	return NewToolService(registry)
}

type fakeLLM struct {
	responses       []string
	streamResponses []string
	prompts         []string
	streamPrompts   []string
}

func (l *fakeLLM) Chat(ctx context.Context, prompt string) (string, error) {
	l.prompts = append(l.prompts, prompt)
	if len(l.responses) == 0 {
		return "", nil
	}
	response := l.responses[0]
	l.responses = l.responses[1:]
	return response, nil
}

func (l *fakeLLM) Stream(ctx context.Context, prompt string, onChunk func(chunk string) error) error {
	l.streamPrompts = append(l.streamPrompts, prompt)
	for len(l.streamResponses) > 0 {
		chunk := l.streamResponses[0]
		l.streamResponses = l.streamResponses[1:]
		if err := onChunk(chunk); err != nil {
			return err
		}
	}
	return nil
}

type fakeTool struct {
	name        string
	description string
}

func newFakeTool(name string, description string) fakeTool {
	return fakeTool{name: name, description: description}
}

func (t fakeTool) Name() string {
	return t.name
}

func (t fakeTool) Description() string {
	return t.description
}

func (t fakeTool) Execute(ctx context.Context, input string) (string, error) {
	return "Acao " + strings.TrimSpace(input) + " executada com sucesso.", nil
}
