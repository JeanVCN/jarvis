package services

import (
	"context"
	"strings"

	"github.com/jean/jarvis/backend/internal/voice"
)

type VoiceService struct {
	recognizer voice.SpeechRecognizer
	provider   voice.VoiceProvider
	chat       *ChatService
}

type availabilityChecker interface {
	Available() bool
	Command() string
}

type VoiceConversationResult struct {
	Transcript       string
	Response         string
	Audio            []byte
	AudioContentType string
}

func NewVoiceService(recognizer voice.SpeechRecognizer, provider voice.VoiceProvider, chat *ChatService) *VoiceService {
	return &VoiceService{
		recognizer: recognizer,
		provider:   provider,
		chat:       chat,
	}
}

func (s *VoiceService) Converse(ctx context.Context, input voice.AudioInput) (VoiceConversationResult, error) {
	transcript, err := s.recognizer.Recognize(ctx, input)
	if err != nil {
		return VoiceConversationResult{}, err
	}

	response, err := s.chat.SendMessage(ctx, transcript)
	if err != nil {
		return VoiceConversationResult{}, err
	}

	speech, err := s.provider.Synthesize(ctx, response)
	if err != nil {
		return VoiceConversationResult{}, err
	}

	return VoiceConversationResult{
		Transcript:       strings.TrimSpace(transcript),
		Response:         strings.TrimSpace(response),
		Audio:            speech.Audio,
		AudioContentType: speech.ContentType,
	}, nil
}

func (s *VoiceService) Status() voice.ProviderStatus {
	sttAvailable, sttCommand := providerAvailability(s.recognizer)
	ttsAvailable, ttsCommand := providerAvailability(s.provider)

	message := "voice_ready"
	if !sttAvailable || !ttsAvailable {
		message = "voice_provider_unavailable"
	}

	return voice.ProviderStatus{
		STTAvailable: sttAvailable,
		TTSAvailable: ttsAvailable,
		STTCommand:   sttCommand,
		TTSCommand:   ttsCommand,
		Message:      message,
	}
}

func providerAvailability(provider any) (bool, string) {
	checker, ok := provider.(availabilityChecker)
	if !ok {
		return true, ""
	}

	return checker.Available(), checker.Command()
}
