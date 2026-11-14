package conversation

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type Message struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCalls  []Call `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}
type Function struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}
type Call struct {
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}
type Failure struct {
	Code       string `json:"code"`
	RetryAfter string `json:"retry_after,omitempty"`
}

func (e *Failure) Error() string { return e.Code }

type Provider interface {
	Stream(context.Context, []Message, func(string) error) ([]Call, error)
}
type Scripted struct{ Delay time.Duration }

func (p Scripted) Stream(ctx context.Context, messages []Message, emit func(string) error) ([]Call, error) {
	last := messages[len(messages)-1]
	text := "Olá! Estou pronto para conversar em português. Esta resposta reproduzível mede o caminho de voz, sem chamar um modelo real."
	if last.Role == "tool" {
		text = "O resultado da ferramenta é " + last.Content + "."
	} else if strings.Contains(strings.ToLower(last.Content), "ferramenta") {
		return []Call{{ID: "script-calculator", Type: "function", Function: Function{Name: "calculator", Arguments: `{"input":"25 * 4"}`}}}, nil
	} else if strings.Contains(strings.ToLower(last.Content), "longa") {
		text = strings.Repeat("Esta é uma resposta longa para testar interrupções. Você pode falar enquanto eu respondo. ", 8)
	}
	for _, word := range strings.Fields(text) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(p.Delay):
		}
		if err := emit(word + " "); err != nil {
			return nil, err
		}
	}
	return nil, nil
}

type HTTPProvider struct {
	URL, Key, Model string
	Ollama          bool
	Client          *http.Client
}

func definitions() []map[string]any {
	out := []map[string]any{}
	for _, name := range []string{"calculator", "time"} {
		out = append(out, map[string]any{"type": "function", "function": map[string]any{"name": name, "description": "Ferramenta segura do Jarvis: " + name, "parameters": map[string]any{"type": "object", "properties": map[string]any{"input": map[string]string{"type": "string"}}, "required": []string{"input"}, "additionalProperties": false}}})
	}
	return out
}
func (p HTTPProvider) Stream(ctx context.Context, messages []Message, emit func(string) error) ([]Call, error) {
	payload := map[string]any{"model": p.Model, "messages": messages, "stream": true, "tools": definitions()}
	if p.Ollama {
		if strings.HasPrefix(p.Model, "llama3.2") {
			delete(payload, "tools")
		}
		normalized := []map[string]any{}
		for _, m := range messages {
			raw, _ := json.Marshal(m)
			var v map[string]any
			_ = json.Unmarshal(raw, &v)
			if len(m.ToolCalls) > 0 {
				toolCalls := []map[string]any{}
				for _, c := range m.ToolCalls {
					var args any
					_ = json.Unmarshal([]byte(c.Function.Arguments), &args)
					toolCalls = append(toolCalls, map[string]any{"function": map[string]any{"name": c.Function.Name, "arguments": args}})
				}
				v["tool_calls"] = toolCalls
			}
			normalized = append(normalized, v)
		}
		payload["messages"] = normalized
		payload["options"] = map[string]any{"num_ctx": 2048, "num_thread": 2}
		payload["think"] = false
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", p.URL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if p.Key != "" {
		req.Header.Set("Authorization", "Bearer "+p.Key)
	}
	client := p.Client
	if client == nil {
		client = &http.Client{Timeout: 180 * time.Second}
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		code := "provider_unavailable"
		switch res.StatusCode {
		case 401, 403:
			code = "authentication_required"
		case 404:
			code = "model_unavailable"
		case 429:
			code = "quota_exhausted"
		}
		return nil, &Failure{Code: code, RetryAfter: res.Header.Get("Retry-After")}
	}
	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 4096), 1024*1024)
	calls := map[int]*Call{}
	order := []int{}
	completed := false
	for scanner.Scan() {
		line := scanner.Text()
		if !p.Ollama {
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			line = strings.TrimPrefix(line, "data: ")
			if line == "[DONE]" {
				completed = true
				break
			}
		}
		var frame struct {
			Error   json.RawMessage `json:"error"`
			Done    bool            `json:"done"`
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					Function struct {
						Name      string          `json:"name"`
						Arguments json.RawMessage `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
			Choices []struct {
				Finish *string `json:"finish_reason"`
				Delta  struct {
					Content string `json:"content"`
					Calls   []struct {
						Index    int      `json:"index"`
						ID       string   `json:"id"`
						Function Function `json:"function"`
					} `json:"tool_calls"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(line), &frame); err != nil {
			return nil, fmt.Errorf("invalid_provider_frame: %w", err)
		}
		if len(frame.Error) > 0 && string(frame.Error) != "null" {
			return nil, &Failure{Code: "provider_stream_error"}
		}
		if p.Ollama {
			if frame.Message.Content != "" {
				if err := emit(frame.Message.Content); err != nil {
					return nil, err
				}
			}
			for i, c := range frame.Message.ToolCalls {
				if calls[i] == nil {
					order = append(order, i)
				}
				calls[i] = &Call{ID: fmt.Sprintf("ollama-%d", i), Type: "function", Function: Function{Name: c.Function.Name, Arguments: string(c.Function.Arguments)}}
			}
			if frame.Done {
				completed = true
				break
			}
		} else {
			for _, c := range frame.Choices {
				if c.Delta.Content != "" {
					if err := emit(c.Delta.Content); err != nil {
						return nil, err
					}
				}
				for _, d := range c.Delta.Calls {
					if calls[d.Index] == nil {
						calls[d.Index] = &Call{Type: "function"}
						order = append(order, d.Index)
					}
					v := calls[d.Index]
					v.ID += d.ID
					v.Function.Name += d.Function.Name
					v.Function.Arguments += d.Function.Arguments
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if !completed {
		return nil, io.ErrUnexpectedEOF
	}
	result := []Call{}
	for _, i := range order {
		c := calls[i]
		if c.ID == "" || c.Function.Name == "" {
			return nil, errors.New("invalid_tool_call")
		}
		result = append(result, *c)
	}
	return result, nil
}
func Resolve(name, model string) (Provider, error) {
	switch name {
	case "scripted":
		return Scripted{Delay: 35 * time.Millisecond}, nil
	case "groq":
		if os.Getenv("JARVIS_GROQ_FREE_CONFIRMED") != "true" {
			return nil, &Failure{Code: "free_account_confirmation_required"}
		}
		key := os.Getenv("GROQ_API_KEY")
		if key == "" {
			return nil, &Failure{Code: "authentication_required"}
		}
		if model != "openai/gpt-oss-120b" && model != "qwen/qwen3.8-27b" {
			return nil, &Failure{Code: "model_not_allowed"}
		}
		return HTTPProvider{URL: "https://api.groq.com/openai/v1/chat/completions", Key: key, Model: model}, nil
	case "ollama":
		if model == "" {
			model = os.Getenv("OLLAMA_MODEL")
			if model == "" {
				model = "llama3.2"
			}
		}
		base := os.Getenv("OLLAMA_URL")
		if base == "" {
			base = "http://127.0.0.1:11434"
		}
		return HTTPProvider{URL: strings.TrimRight(base, "/") + "/api/chat", Model: model, Ollama: true}, nil
	case "codex":
		return NewCodex(model)
	default:
		return nil, &Failure{Code: "unknown_provider"}
	}
}
