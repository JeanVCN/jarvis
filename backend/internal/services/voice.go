package services

import (
	"context"
	"errors"
	"strings"

	"github.com/jean/jarvis/backend/internal/voice"
)

type VoiceService struct {
	recognizer voice.SpeechRecognizer
	provider   voice.VoiceProvider
	chat       *ChatService
	wakeWords  []string
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
	Ignored          bool
	WakeWordDetected bool
}

func NewVoiceService(recognizer voice.SpeechRecognizer, provider voice.VoiceProvider, chat *ChatService) *VoiceService {
	return &VoiceService{
		recognizer: recognizer,
		provider:   provider,
		chat:       chat,
		wakeWords:  []string{"jarvis"},
	}
}

func (s *VoiceService) Converse(ctx context.Context, input voice.AudioInput) (VoiceConversationResult, error) {
	transcript, err := s.recognizer.Recognize(ctx, input)
	if err != nil {
		if input.WakeRequired && errors.Is(err, voice.ErrEmptyTranscript) {
			return VoiceConversationResult{Ignored: true}, nil
		}
		return VoiceConversationResult{}, err
	}

	transcript = strings.TrimSpace(transcript)
	message := transcript
	wakeDetected := false
	if input.WakeRequired {
		words := input.WakeWords
		if len(words) == 0 {
			words = s.wakeWords
		}
		var ok bool
		message, ok = stripWakeWord(transcript, words)
		if !ok || strings.TrimSpace(message) == "" {
			return VoiceConversationResult{
				Transcript:       transcript,
				Ignored:          true,
				WakeWordDetected: ok,
			}, nil
		}
		wakeDetected = true
	}

	response, err := s.chat.SendMessage(ctx, message)
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
		WakeWordDetected: wakeDetected,
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
		PrivacyMode:  "local",
	}
}

func providerAvailability(provider any) (bool, string) {
	checker, ok := provider.(availabilityChecker)
	if !ok {
		return true, ""
	}

	return checker.Available(), checker.Command()
}

func stripWakeWord(transcript string, wakeWords []string) (string, bool) {
	normalizedTranscript := strings.ToLower(transcript)
	for _, word := range wakeWords {
		normalizedWord := strings.ToLower(strings.TrimSpace(word))
		if normalizedWord == "" {
			continue
		}
		index := strings.Index(normalizedTranscript, normalizedWord)
		if index < 0 {
			continue
		}

		before := strings.TrimSpace(transcript[:min(index, len(transcript))])
		afterStart := min(index+len(normalizedWord), len(transcript))
		after := strings.TrimLeft(transcript[afterStart:], " ,.;:!?-")
		if before == "" {
			return after, true
		}
		return strings.TrimSpace(before + " " + after), true
	}
	return transcript, false
}
