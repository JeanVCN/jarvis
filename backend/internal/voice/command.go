package voice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var ErrVoiceProviderUnavailable = errors.New("voice provider unavailable")

type CommandSpeechRecognizer struct {
	command  string
	args     []string
	model    string
	language string
	runner   commandRunner
}

type CommandVoiceProvider struct {
	command     string
	args        []string
	voice       string
	contentType string
	runner      commandRunner
}

type commandRunner interface {
	Run(ctx context.Context, command string, args []string, stdin []byte) ([]byte, error)
}

type execRunner struct{}

func NewCommandSpeechRecognizer(command string, args []string, model string, language string) *CommandSpeechRecognizer {
	return &CommandSpeechRecognizer{
		command:  strings.TrimSpace(command),
		args:     args,
		model:    strings.TrimSpace(model),
		language: strings.TrimSpace(language),
		runner:   execRunner{},
	}
}

func NewCommandVoiceProvider(command string, args []string, voice string) *CommandVoiceProvider {
	return &CommandVoiceProvider{
		command:     strings.TrimSpace(command),
		args:        args,
		voice:       strings.TrimSpace(voice),
		contentType: "audio/wav",
		runner:      execRunner{},
	}
}

func (r *CommandSpeechRecognizer) Recognize(ctx context.Context, input AudioInput) (string, error) {
	if r.command == "" {
		return "", ErrVoiceProviderUnavailable
	}

	dir, err := os.MkdirTemp("", "jarvis-stt-*")
	if err != nil {
		return "", fmt.Errorf("create stt temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	inputPath := filepath.Join(dir, "input."+audioExtension(input.Format))
	if err := os.WriteFile(inputPath, input.Data, 0o600); err != nil {
		return "", fmt.Errorf("write stt input: %w", err)
	}

	language := strings.TrimSpace(input.Language)
	if language == "" {
		language = r.language
	}

	values := map[string]string{
		"input":      inputPath,
		"output_dir": dir,
		"model":      r.model,
		"language":   language,
	}
	args := expandArgs(r.args, values)

	output, err := r.runner.Run(ctx, r.command, args, nil)
	if err != nil {
		return "", fmt.Errorf("%w: stt command failed: %v: %s", ErrVoiceProviderUnavailable, err, strings.TrimSpace(string(output)))
	}

	transcript, err := readTranscript(dir, output)
	if err != nil {
		return "", err
	}

	transcript = strings.TrimSpace(transcript)
	if transcript == "" {
		return "", fmt.Errorf("stt returned empty transcript")
	}

	return transcript, nil
}

func (r *CommandSpeechRecognizer) Available() bool {
	if r.command == "" {
		return false
	}
	_, err := exec.LookPath(r.command)
	return err == nil
}

func (r *CommandSpeechRecognizer) Command() string {
	return r.command
}

func (p *CommandVoiceProvider) Synthesize(ctx context.Context, text string) (SpeechOutput, error) {
	if p.command == "" {
		return SpeechOutput{}, ErrVoiceProviderUnavailable
	}

	dir, err := os.MkdirTemp("", "jarvis-tts-*")
	if err != nil {
		return SpeechOutput{}, fmt.Errorf("create tts temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	outputPath := filepath.Join(dir, "speech.wav")
	values := map[string]string{
		"output": outputPath,
		"voice":  p.voice,
	}
	args := expandArgs(p.args, values)

	output, err := p.runner.Run(ctx, p.command, args, []byte(strings.TrimSpace(text)))
	if err != nil {
		return SpeechOutput{}, fmt.Errorf("%w: tts command failed: %v: %s", ErrVoiceProviderUnavailable, err, strings.TrimSpace(string(output)))
	}

	audio, err := os.ReadFile(outputPath)
	if err != nil {
		return SpeechOutput{}, fmt.Errorf("read tts output: %w", err)
	}
	if len(audio) == 0 {
		return SpeechOutput{}, fmt.Errorf("tts returned empty audio")
	}

	return SpeechOutput{Audio: audio, ContentType: p.contentType}, nil
}

func (p *CommandVoiceProvider) Available() bool {
	if p.command == "" {
		return false
	}
	_, err := exec.LookPath(p.command)
	return err == nil
}

func (p *CommandVoiceProvider) Command() string {
	return p.command
}

func ParseArgs(raw string) []string {
	return strings.Fields(raw)
}

func expandArgs(args []string, values map[string]string) []string {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		for key, value := range values {
			arg = strings.ReplaceAll(arg, "{"+key+"}", value)
		}
		if arg != "" {
			expanded = append(expanded, arg)
		}
	}
	return expanded
}

func readTranscript(dir string, commandOutput []byte) (string, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return "", fmt.Errorf("find stt transcript: %w", err)
	}
	if len(files) > 0 {
		data, err := os.ReadFile(files[0])
		if err != nil {
			return "", fmt.Errorf("read stt transcript: %w", err)
		}
		return string(data), nil
	}

	return string(commandOutput), nil
}

func audioExtension(format string) string {
	format = strings.ToLower(format)
	switch {
	case strings.Contains(format, "wav"):
		return "wav"
	case strings.Contains(format, "mpeg"), strings.Contains(format, "mp3"):
		return "mp3"
	case strings.Contains(format, "ogg"):
		return "ogg"
	default:
		return "webm"
	}
}

func (r execRunner) Run(ctx context.Context, command string, args []string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	if len(stdin) > 0 {
		cmd.Stdin = bytes.NewReader(stdin)
	}
	return cmd.CombinedOutput()
}
