package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type OllamaProvider struct {
	baseURL    string
	model      string
	httpClient *http.Client
}

func NewOllamaProvider(baseURL, model string) *OllamaProvider {
	return &OllamaProvider{
		baseURL:    strings.TrimRight(baseURL, "/"),
		model:      model,
		httpClient: &http.Client{},
	}
}

func (p *OllamaProvider) Chat(ctx context.Context, prompt string) (string, error) {
	endpoint, err := url.JoinPath(p.baseURL, "/api/generate")
	if err != nil {
		return "", fmt.Errorf("build ollama endpoint: %w", err)
	}

	payload := ollamaGenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create ollama request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var data ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("decode ollama response: %w", err)
	}

	answer := strings.TrimSpace(data.Response)
	if answer == "" {
		return "", fmt.Errorf("ollama returned empty response")
	}

	return answer, nil
}

func (p *OllamaProvider) Stream(ctx context.Context, prompt string, onChunk func(chunk string) error) error {
	endpoint, err := url.JoinPath(p.baseURL, "/api/generate")
	if err != nil {
		return fmt.Errorf("build ollama endpoint: %w", err)
	}

	payload := ollamaGenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: true,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode ollama request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create ollama stream request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call ollama stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("ollama stream returned status %d", resp.StatusCode)
	}

	decoder := json.NewDecoder(resp.Body)
	for {
		var data ollamaGenerateResponse
		if err := decoder.Decode(&data); err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("decode ollama stream response: %w", err)
		}
		if data.Response != "" {
			if err := onChunk(data.Response); err != nil {
				return err
			}
		}
		if data.Done {
			return nil
		}
	}

	return nil
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}
