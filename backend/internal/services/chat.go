package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/jean/jarvis/backend/internal/ai"
)

type ChatService struct {
	llm   ai.LLM
	tools *ToolService
}

func NewChatService(llm ai.LLM, tools *ToolService) *ChatService {
	return &ChatService{llm: llm, tools: tools}
}

func (s *ChatService) SendMessage(ctx context.Context, message string) (string, error) {
	message = strings.TrimSpace(message)
	if call, ok := s.detectToolCall(message); ok {
		return s.executeToolCall(ctx, call)
	}

	return s.llm.Chat(ctx, message)
}

func (s *ChatService) StreamMessage(ctx context.Context, message string, onChunk func(chunk string) error) error {
	message = strings.TrimSpace(message)
	if call, ok := s.detectToolCall(message); ok {
		response, err := s.executeToolCall(ctx, call)
		if err != nil {
			return err
		}
		return onChunk(response)
	}

	return s.llm.Stream(ctx, message, onChunk)
}

func (s *ChatService) executeToolCall(ctx context.Context, call toolCall) (string, error) {
	result, err := s.tools.ExecuteTool(ctx, call.Tool, call.Input)
	if err != nil {
		return s.toolErrorResponse(call, err), nil
	}
	return s.toolSuccessResponse(call, result), nil
}

func (s *ChatService) toolSuccessResponse(call toolCall, result string) string {
	switch call.Tool {
	case "system":
		return systemActionSuccessResponse(call.Input)
	case "calculator":
		return fmt.Sprintf("O resultado e %s.", strings.TrimSpace(result))
	case "time":
		return fmt.Sprintf("Agora: %s.", strings.TrimSpace(result))
	default:
		return strings.TrimSpace(result)
	}
}

func (s *ChatService) toolErrorResponse(call toolCall, err error) string {
	switch call.Tool {
	case "system":
		return fmt.Sprintf("Nao consegui executar essa acao: %v", err)
	default:
		return fmt.Sprintf("Nao consegui usar a ferramenta %s: %v", call.Tool, err)
	}
}

func systemActionSuccessResponse(input string) string {
	switch input {
	case "open_spotify":
		return "Pronto, abri o Spotify."
	case "open_vscode":
		return "Pronto, abri o VSCode."
	case "open_browser":
		return "Pronto, abri o navegador."
	case "open_terminal":
		return "Pronto, abri o terminal."
	case "lock_screen":
		return "Pronto, bloqueei a tela."
	default:
		return "Pronto, acao executada."
	}
}

func (s *ChatService) detectToolCall(message string) (toolCall, bool) {
	normalized := normalizePortuguese(message)
	if normalized == "" || s.tools == nil {
		return toolCall{}, false
	}

	if input, ok := detectSystemAction(normalized); ok && s.hasTool("system") {
		log.Printf("chat tool detected: tool=system input=%s message=%q", input, message)
		return toolCall{Tool: "system", Input: input}, true
	}

	if shouldUseTimeTool(normalized) && s.hasTool("time") {
		log.Printf("chat tool detected: tool=time input= message=%q", message)
		return toolCall{Tool: "time"}, true
	}

	if input, ok := extractCalculatorInput(normalized); ok && s.hasTool("calculator") {
		log.Printf("chat tool detected: tool=calculator input=%q message=%q", input, message)
		return toolCall{Tool: "calculator", Input: input}, true
	}

	return toolCall{}, false
}

func (s *ChatService) hasTool(name string) bool {
	for _, tool := range s.tools.ListTools() {
		if tool.Name == name {
			return true
		}
	}
	return false
}

func shouldUseTimeTool(message string) bool {
	timeIntents := []string{
		"que horas",
		"qual horario",
		"hora atual",
		"horas sao",
		"data atual",
		"qual a data",
		"que dia e",
		"dia de hoje",
	}
	return containsAny(message, timeIntents)
}

func detectSystemAction(message string) (string, bool) {
	hasActionIntent := containsAny(message, []string{
		"abra",
		"abrir",
		"abre",
		"abrir o",
		"abrir a",
		"inicie",
		"iniciar",
		"executa",
		"execute",
		"rode",
		"rodar",
		"chama",
		"chamar",
		"coloque",
		"colocar",
		"bloqueie",
		"bloquear",
	})
	if !hasActionIntent && !looksLikeShortSystemCommand(message) {
		return "", false
	}

	switch {
	case containsAny(message, []string{"vscode", "vs code", "visual studio code", "code"}):
		return "open_vscode", true
	case containsAny(message, []string{"navegador", "browser", "chrome", "firefox"}):
		return "open_browser", true
	case strings.Contains(message, "terminal"):
		return "open_terminal", true
	case containsAny(message, []string{"spotify", "spotfy", "spotifi", "spotifai", "spoti fi", "spot fire", "spotif"}):
		return "open_spotify", true
	case containsAny(message, []string{"bloqueia a tela", "bloqueie a tela", "bloquear a tela", "lock screen"}):
		return "lock_screen", true
	default:
		return "", false
	}
}

func looksLikeShortSystemCommand(message string) bool {
	if len(strings.Fields(message)) > 5 {
		return false
	}
	if containsAny(message, []string{"?", "o que", "quem", "qual", "porque", "por que", "quando", "onde"}) {
		return false
	}
	return containsAny(message, []string{
		"vscode",
		"vs code",
		"visual studio code",
		"navegador",
		"browser",
		"chrome",
		"firefox",
		"terminal",
		"spotify",
		"spotfy",
		"spotifi",
		"spotifai",
		"spoti fi",
		"spot fire",
		"spotif",
	})
}

func extractCalculatorInput(message string) (string, bool) {
	replacer := strings.NewReplacer(
		"multiplicado por", "*",
		"dividido por", "/",
		"vezes", "*",
		"mais", "+",
		"menos", "-",
		"dividido", "/",
		"quanto e", " ",
		"calcule", " ",
		"calcula", " ",
	)
	message = replacer.Replace(message)
	message = strings.ReplaceAll(message, ",", ".")

	var best string
	var current strings.Builder
	for _, r := range message {
		if isCalculatorRune(r) {
			current.WriteRune(r)
			continue
		}
		best = longerCalculatorCandidate(best, current.String())
		current.Reset()
	}
	best = longerCalculatorCandidate(best, current.String())
	best = strings.TrimSpace(best)
	if !looksLikeCalculation(best) {
		return "", false
	}
	return best, true
}

func longerCalculatorCandidate(current string, candidate string) string {
	candidate = strings.TrimSpace(candidate)
	if len(candidate) > len(current) {
		return candidate
	}
	return current
}

func looksLikeCalculation(input string) bool {
	if input == "" {
		return false
	}

	hasDigit := false
	hasOperator := false
	for _, r := range input {
		switch {
		case r >= '0' && r <= '9':
			hasDigit = true
		case r == '+' || r == '-' || r == '*' || r == '/':
			hasOperator = true
		}
	}
	return hasDigit && hasOperator
}

func isCalculatorRune(r rune) bool {
	return (r >= '0' && r <= '9') || r == ' ' || r == '\t' || r == '.' || r == '+' || r == '-' || r == '*' || r == '/' || r == '(' || r == ')'
}

func normalizePortuguese(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u",
		"ç", "c",
	)
	return replacer.Replace(value)
}

func containsAny(value string, options []string) bool {
	for _, option := range options {
		if strings.Contains(value, option) {
			return true
		}
	}
	return false
}

type toolCall struct {
	Tool  string
	Input string
}
