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

func TestVoiceServiceIgnoresWakeRequiredAudioWithoutWakeWord(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{responses: []string{"nao deveria chamar"}}
	provider := &countingVoiceProvider{}
	service := NewVoiceService(
		fakeRecognizer{transcript: "abra o Spotify"},
		provider,
		NewChatService(llm, newChatTestToolService(t)),
	)

	result, err := service.Converse(context.Background(), voice.AudioInput{
		Data:         []byte("input"),
		Format:       "audio/webm",
		WakeRequired: true,
	})
	if err != nil {
		t.Fatalf("Converse returned error: %v", err)
	}
	if !result.Ignored {
		t.Fatal("expected ignored result")
	}
	if result.WakeWordDetected {
		t.Fatal("did not expect wake word")
	}
	if len(llm.prompts) != 0 {
		t.Fatalf("expected no llm calls, got %d", len(llm.prompts))
	}
	if provider.calls != 0 {
		t.Fatalf("expected no tts calls, got %d", provider.calls)
	}
}

func TestVoiceServiceIgnoresEmptyTranscriptWhenWakeRequired(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{responses: []string{"nao deveria chamar"}}
	provider := &countingVoiceProvider{}
	service := NewVoiceService(
		emptyTranscriptRecognizer{},
		provider,
		NewChatService(llm, newChatTestToolService(t)),
	)

	result, err := service.Converse(context.Background(), voice.AudioInput{
		Data:         []byte("input"),
		Format:       "audio/webm",
		WakeRequired: true,
	})
	if err != nil {
		t.Fatalf("Converse returned error: %v", err)
	}
	if !result.Ignored {
		t.Fatal("expected ignored result")
	}
	if len(llm.prompts) != 0 {
		t.Fatalf("expected no llm calls, got %d", len(llm.prompts))
	}
	if provider.calls != 0 {
		t.Fatalf("expected no tts calls, got %d", provider.calls)
	}
}

func TestVoiceServiceStripsWakeWordBeforeChat(t *testing.T) {
	t.Parallel()

	llm := &fakeLLM{responses: []string{"Pronto"}}
	service := NewVoiceService(
		fakeRecognizer{transcript: "Jarvis diga ola"},
		fakeVoiceProvider{audio: []byte("audio")},
		NewChatService(llm, newChatTestToolService(t)),
	)

	result, err := service.Converse(context.Background(), voice.AudioInput{
		Data:         []byte("input"),
		Format:       "audio/webm",
		WakeRequired: true,
	})
	if err != nil {
		t.Fatalf("Converse returned error: %v", err)
	}
	if result.Ignored {
		t.Fatal("did not expect ignored result")
	}
	if !result.WakeWordDetected {
		t.Fatal("expected wake word")
	}
	if len(llm.prompts) != 1 || llm.prompts[0] != "diga ola" {
		t.Fatalf("unexpected llm prompts: %#v", llm.prompts)
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

type countingVoiceProvider struct {
	calls int
}

func (p *countingVoiceProvider) Synthesize(ctx context.Context, text string) (voice.SpeechOutput, error) {
	p.calls++
	return voice.SpeechOutput{Audio: []byte("audio"), ContentType: "audio/wav"}, nil
}

type emptyTranscriptRecognizer struct{}

func (r emptyTranscriptRecognizer) Recognize(ctx context.Context, input voice.AudioInput) (string, error) {
	return "", voice.ErrEmptyTranscript
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
