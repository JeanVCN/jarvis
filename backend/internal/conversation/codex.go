package conversation

// Codex is opt-in: it uses an isolated CODEX_HOME and never inherits API keys.
// Tool requests use Jarvis's JSON envelope; execution stays in the Go journal.
import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Codex struct{ Model, Home, Binary string }

func NewCodex(model string) (Provider, error) {
	home := os.Getenv("JARVIS_CODEX_HOME")
	if os.Getenv("JARVIS_CODEX_ENABLED") != "true" || !filepath.IsAbs(home) {
		return nil, &Failure{Code: "codex_activation_required"}
	}
	bin := os.Getenv("JARVIS_CODEX_BIN")
	if bin == "" {
		bin = "codex"
	}
	return Codex{Model: model, Home: home, Binary: bin}, nil
}
func (p Codex) Stream(ctx context.Context, messages []Message, emit func(string) error) ([]Call, error) {
	work, e := os.MkdirTemp("", "jarvis-codex-")
	if e != nil {
		return nil, e
	}
	defer os.RemoveAll(work)
	cmd := exec.CommandContext(ctx, p.Binary, "app-server", "-c", "features.shell_tool=false", "-c", "features.unified_exec=false", "-c", "features.apply_patch_freeform=false", "-c", "features.apps=false", "-c", "features.multi_agent=false", "-c", `web_search="disabled"`)
	cmd.Dir = work
	for _, name := range []string{"PATH", "HOME", "USER", "LANG", "HTTP_PROXY", "HTTPS_PROXY", "ALL_PROXY", "NO_PROXY", "http_proxy", "https_proxy", "all_proxy", "no_proxy", "SSL_CERT_FILE", "SSL_CERT_DIR", "REQUESTS_CA_BUNDLE", "NODE_EXTRA_CA_CERTS", "CODEX_PROXY_CERT", "TMPDIR"} {
		if value, ok := os.LookupEnv(name); ok {
			cmd.Env = append(cmd.Env, name+"="+value)
		}
	}
	cmd.Env = append(cmd.Env, "CODEX_HOME="+p.Home)
	input, e := cmd.StdinPipe()
	if e != nil {
		return nil, e
	}
	output, e := cmd.StdoutPipe()
	if e != nil {
		return nil, e
	}
	cmd.Stderr = io.Discard
	if e = cmd.Start(); e != nil {
		return nil, e
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	var mu sync.Mutex
	send := func(v any) error { mu.Lock(); defer mu.Unlock(); return json.NewEncoder(input).Encode(v) }
	scan := bufio.NewScanner(output)
	scan.Buffer(make([]byte, 8192), 2*1024*1024)
	request := func(id int, method string, params any) (map[string]json.RawMessage, error) {
		if e := send(map[string]any{"id": id, "method": method, "params": params}); e != nil {
			return nil, e
		}
		for scan.Scan() {
			var v map[string]json.RawMessage
			if json.Unmarshal(scan.Bytes(), &v) != nil {
				continue
			}
			var n int
			_ = json.Unmarshal(v["id"], &n)
			if n != id {
				continue
			}
			if len(v["error"]) > 0 {
				return nil, codexFailure(v["error"])
			}
			var r map[string]json.RawMessage
			e := json.Unmarshal(v["result"], &r)
			return r, e
		}
		return nil, errors.New("codex_connection_closed")
	}
	if _, e = request(1, "initialize", map[string]any{"clientInfo": map[string]string{"name": "jarvis", "version": "0.1.0"}}); e != nil {
		return nil, e
	}
	if e = send(map[string]string{"method": "initialized"}); e != nil {
		return nil, e
	}
	account, e := request(2, "account/read", map[string]any{})
	if e != nil {
		return nil, e
	}
	var a struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(account["account"], &a)
	if a.Type != "chatgpt" {
		return nil, &Failure{Code: "chatgpt_login_required"}
	}
	params := map[string]any{"cwd": work, "ephemeral": true, "sandbox": "read-only", "approvalPolicy": "never", "baseInstructions": "Você é o Jarvis. Não use ferramentas nativas, arquivos, comandos ou rede. Responda em português, com texto curto. Quando precisar de uma ferramenta Jarvis, responda SOMENTE JSON: {\"jarvis_tool\":\"calculator\",\"input\":\"25 * 4\"}. Ferramentas permitidas: calculator e time. Resultados estarão no contexto; nunca invente execução."}
	if p.Model != "" {
		params["model"] = p.Model
	}
	started, e := request(3, "thread/start", params)
	if e != nil {
		return nil, e
	}
	var thread struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(started["thread"], &thread)
	if thread.ID == "" {
		return nil, errors.New("missing_thread")
	}
	contextJSON, _ := json.Marshal(messages)
	result, e := request(4, "turn/start", map[string]any{"threadId": thread.ID, "input": []any{map[string]any{"type": "text", "text": string(contextJSON), "text_elements": []any{}}}})
	if e != nil {
		return nil, e
	}
	var turn struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(result["turn"], &turn)
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = send(map[string]any{"id": 5, "method": "turn/interrupt", "params": map[string]string{"threadId": thread.ID, "turnId": turn.ID}})
		case <-done:
		}
	}()
	text := ""
	mode := ""
	emitted := 0
	for scan.Scan() {
		var v struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(scan.Bytes(), &v) != nil {
			continue
		}
		if len(v.ID) > 0 {
			_ = send(map[string]any{"id": v.ID, "error": map[string]any{"code": -32601, "message": "Jarvis denies native operations"}})
			return nil, &Failure{Code: "native_tool_denied"}
		}
		switch v.Method {
		case "item/agentMessage/delta":
			var d struct {
				Delta string `json:"delta"`
			}
			_ = json.Unmarshal(v.Params, &d)
			text += d.Delta
			if mode == "" && strings.TrimSpace(text) != "" {
				if strings.HasPrefix(strings.TrimSpace(text), "{") {
					mode = "json"
				} else {
					mode = "text"
				}
			}
			if mode == "text" {
				if e = emit(text[emitted:]); e != nil {
					return nil, e
				}
				emitted = len(text)
			}
		case "item/started":
			var d struct {
				Item struct {
					Type string `json:"type"`
				} `json:"item"`
			}
			_ = json.Unmarshal(v.Params, &d)
			switch d.Item.Type {
			case "commandExecution", "fileChange", "mcpToolCall", "webSearch":
				return nil, &Failure{Code: "native_tool_denied"}
			}
		case "turn/completed":
			var d struct {
				Turn struct {
					Status string `json:"status"`
				} `json:"turn"`
			}
			_ = json.Unmarshal(v.Params, &d)
			if d.Turn.Status != "completed" {
				return nil, codexFailure(v.Params)
			}
			if mode == "json" {
				var tool struct {
					Name  string `json:"jarvis_tool"`
					Input string `json:"input"`
				}
				if json.Unmarshal([]byte(text), &tool) == nil && tool.Name != "" {
					args, _ := json.Marshal(map[string]string{"input": tool.Input})
					return []Call{{ID: ID(), Type: "function", Function: Function{Name: tool.Name, Arguments: string(args)}}}, nil
				}
				if e = emit(text); e != nil {
					return nil, e
				}
			}
			return nil, nil
		case "error":
			return nil, codexFailure(v.Params)
		}
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, errors.New("codex_incomplete_stream")
}

func codexFailure(raw json.RawMessage) error {
	text := strings.ToLower(string(raw))
	code := "codex_provider_error"
	switch {
	case strings.Contains(text, "usagelimit"), strings.Contains(text, "usage limit"), strings.Contains(text, "rate_limit"), strings.Contains(text, "quota"), strings.Contains(text, "429"):
		code = "quota_exhausted"
	case strings.Contains(text, "unauthorized"), strings.Contains(text, "tokenexpired"), strings.Contains(text, "authentication"):
		code = "authentication_required"
	case strings.Contains(text, "modelnotfound"), strings.Contains(text, "model_not_found"):
		code = "model_unavailable"
	}
	return &Failure{Code: code}
}
