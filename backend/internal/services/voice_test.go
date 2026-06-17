package services

import (
	"context"
	"testing"

	"github.com/jean/jarvis/backend/internal/voice"
)

func TestVoiceServiceConverse(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{responses: []string{"Resposta em texto"}}
	service := NewVoiceService(
		fakeRecognizer{transcript: "Ola Jarvis"},
		fakeVoiceProvider{audio: []byte("audio")},
		NewChatService(llm, newChatTestToolService(t)),
	)

	result, err := service.Converse(context.Background(), voice.AudioInput{
		Data:   []byte("input"),
		Format: "audio/webm",
	})
	if err != nil {
		t.Fatalf("Converse returned error: %v", err)
	}
	if result.Transcript != "Ola Jarvis" {
		t.Fatalf("unexpected transcript: %s", result.Transcript)
	}
	if result.Response != "Resposta em texto" {
		t.Fatalf("unexpected response: %s", result.Response)
	}
	if string(result.Audio) != "audio" {
		t.Fatalf("unexpected audio: %s", string(result.Audio))
	}
}

func TestVoiceServiceStatus(t *testing.T) {
	t.Parallel()

	service := NewVoiceService(
		fakeStatusRecognizer{available: false, command: "faster-whisper"},
		fakeStatusVoiceProvider{available: true, command: "piper"},
		NewChatService(&fakeLLM{}, newChatTestToolService(t)),
	)

	status := service.Status()
	if status.STTAvailable {
		t.Fatal("expected STT unavailable")
	}
	if !status.TTSAvailable {
		t.Fatal("expected TTS available")
	}
	if status.Message != "voice_provider_unavailable" {
		t.Fatalf("unexpected message: %s", status.Message)
	}
}

type fakeRecognizer struct {
	transcript string
}

func (r fakeRecognizer) Recognize(ctx context.Context, input voice.AudioInput) (string, error) {
	return r.transcript, nil
}

type fakeVoiceProvider struct {
	audio []byte
}

func (p fakeVoiceProvider) Synthesize(ctx context.Context, text string) (voice.SpeechOutput, error) {
	return voice.SpeechOutput{Audio: p.audio, ContentType: "audio/wav"}, nil
}

type fakeStatusRecognizer struct {
	available bool
	command   string
}

func (r fakeStatusRecognizer) Recognize(ctx context.Context, input voice.AudioInput) (string, error) {
	return "transcript", nil
}

func (r fakeStatusRecognizer) Available() bool {
	return r.available
}

func (r fakeStatusRecognizer) Command() string {
	return r.command
}

type fakeStatusVoiceProvider struct {
	available bool
	command   string
}

func (p fakeStatusVoiceProvider) Synthesize(ctx context.Context, text string) (voice.SpeechOutput, error) {
	return voice.SpeechOutput{Audio: []byte("audio"), ContentType: "audio/wav"}, nil
}

func (p fakeStatusVoiceProvider) Available() bool {
	return p.available
}

func (p fakeStatusVoiceProvider) Command() string {
	return p.command
}
