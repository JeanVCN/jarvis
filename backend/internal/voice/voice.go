package voice

import "context"

type AudioInput struct {
	Data     []byte
	Format   string
	Language string
}

type SpeechOutput struct {
	Audio       []byte
	ContentType string
}

type SpeechRecognizer interface {
	Recognize(ctx context.Context, input AudioInput) (string, error)
}

type VoiceProvider interface {
	Synthesize(ctx context.Context, text string) (SpeechOutput, error)
}

type ProviderStatus struct {
	STTAvailable bool   `json:"sttAvailable"`
	TTSAvailable bool   `json:"ttsAvailable"`
	STTCommand   string `json:"sttCommand"`
	TTSCommand   string `json:"ttsCommand"`
	Message      string `json:"message"`
}
