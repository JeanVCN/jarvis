package voice

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestHTTPSpeechRecognizerRecognize(t *testing.T) {
	t.Parallel()

	recognizer := NewHTTPSpeechRecognizer("http://stt.local/transcribe", "http://stt.local/health", "pt")
	recognizer.client = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method == http.MethodGet {
			return jsonResponse(http.StatusOK, map[string]string{"status": "ok"}), nil
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm returned error: %v", err)
		}
		file, _, err := r.FormFile("audio")
		if err != nil {
			t.Fatalf("FormFile returned error: %v", err)
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll returned error: %v", err)
		}
		if !bytes.Equal(data, []byte("audio")) {
			t.Fatalf("unexpected audio: %q", string(data))
		}
		if r.FormValue("language") != "pt" {
			t.Fatalf("unexpected language: %s", r.FormValue("language"))
		}
		return jsonResponse(http.StatusOK, map[string]string{"transcript": "Jarvis abra o Spotify"}), nil
	})}

	transcript, err := recognizer.Recognize(context.Background(), AudioInput{
		Data:   []byte("audio"),
		Format: "audio/webm",
	})
	if err != nil {
		t.Fatalf("Recognize returned error: %v", err)
	}
	if transcript != "Jarvis abra o Spotify" {
		t.Fatalf("unexpected transcript: %s", transcript)
	}
	if !recognizer.Available() {
		t.Fatal("expected recognizer available")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, payload any) *http.Response {
	var body bytes.Buffer
	_ = json.NewEncoder(&body).Encode(payload)
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(&body),
		Header:     make(http.Header),
	}
}
