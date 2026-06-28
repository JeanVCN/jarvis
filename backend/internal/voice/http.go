package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

type HTTPSpeechRecognizer struct {
	transcribeURL string
	statusURL     string
	language      string
	client        *http.Client
}

type httpTranscriptResponse struct {
	Transcript string `json:"transcript"`
}

func NewHTTPSpeechRecognizer(transcribeURL string, statusURL string, language string) *HTTPSpeechRecognizer {
	return &HTTPSpeechRecognizer{
		transcribeURL: strings.TrimSpace(transcribeURL),
		statusURL:     strings.TrimSpace(statusURL),
		language:      strings.TrimSpace(language),
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
	}
}

func (r *HTTPSpeechRecognizer) Recognize(ctx context.Context, input AudioInput) (string, error) {
	if r.transcribeURL == "" {
		return "", ErrVoiceProviderUnavailable
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("audio", "speech."+audioExtension(input.Format))
	if err != nil {
		return "", fmt.Errorf("create stt multipart file: %w", err)
	}
	if _, err := part.Write(input.Data); err != nil {
		return "", fmt.Errorf("write stt multipart file: %w", err)
	}

	language := strings.TrimSpace(input.Language)
	if language == "" {
		language = r.language
	}
	if err := writer.WriteField("language", language); err != nil {
		return "", fmt.Errorf("write stt language field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close stt multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.transcribeURL, &body)
	if err != nil {
		return "", fmt.Errorf("create stt request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := r.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: stt server unavailable: %v", ErrVoiceProviderUnavailable, err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("read stt response: %w", err)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		if strings.Contains(string(data), "Invalid data found when processing input") {
			return "", ErrEmptyTranscript
		}
		return "", fmt.Errorf("%w: stt server returned %d: %s", ErrVoiceProviderUnavailable, res.StatusCode, strings.TrimSpace(string(data)))
	}

	var payload httpTranscriptResponse
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", fmt.Errorf("decode stt response: %w", err)
	}
	transcript := strings.TrimSpace(payload.Transcript)
	if transcript == "" {
		return "", ErrEmptyTranscript
	}
	return transcript, nil
}

func (r *HTTPSpeechRecognizer) Available() bool {
	if r.transcribeURL == "" {
		return false
	}
	statusURL := r.statusURL
	if statusURL == "" {
		statusURL = r.transcribeURL
	}

	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, statusURL, nil)
	if err != nil {
		return false
	}
	res, err := r.client.Do(req)
	if err != nil {
		return false
	}
	defer res.Body.Close()
	return res.StatusCode >= 200 && res.StatusCode < 300
}

func (r *HTTPSpeechRecognizer) Command() string {
	return r.transcribeURL
}
