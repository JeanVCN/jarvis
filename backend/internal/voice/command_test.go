package voice

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestCommandSpeechRecognizerUsesConfiguredArgs(t *testing.T) {
	t.Parallel()

	runner := &fakeVoiceRunner{output: []byte("texto reconhecido")}
	recognizer := NewCommandSpeechRecognizer("faster-whisper", ParseArgs("{input} --language {language} --model {model} --output_dir {output_dir}"), "small", "pt")
	recognizer.runner = runner

	transcript, err := recognizer.Recognize(context.Background(), AudioInput{
		Data:   []byte("audio"),
		Format: "audio/webm",
	})
	if err != nil {
		t.Fatalf("Recognize returned error: %v", err)
	}
	if transcript != "texto reconhecido" {
		t.Fatalf("unexpected transcript: %s", transcript)
	}
	if runner.command != "faster-whisper" {
		t.Fatalf("unexpected command: %s", runner.command)
	}
	if !containsArg(runner.args, "--language") || !containsArg(runner.args, "pt") {
		t.Fatalf("expected language args, got %#v", runner.args)
	}
}

func TestCommandVoiceProviderWritesConfiguredOutput(t *testing.T) {
	t.Parallel()

	runner := &fakeVoiceRunner{writeOutput: true}
	provider := NewCommandVoiceProvider("piper", ParseArgs("--model {voice} --output_file {output}"), "pt_BR.onnx")
	provider.runner = runner

	output, err := provider.Synthesize(context.Background(), "Ola")
	if err != nil {
		t.Fatalf("Synthesize returned error: %v", err)
	}
	if output.ContentType != "audio/wav" {
		t.Fatalf("unexpected content type: %s", output.ContentType)
	}
	if string(output.Audio) != "wav-bytes" {
		t.Fatalf("unexpected audio: %s", string(output.Audio))
	}
	if string(runner.stdin) != "Ola" {
		t.Fatalf("unexpected stdin: %s", string(runner.stdin))
	}
}

type fakeVoiceRunner struct {
	command     string
	args        []string
	stdin       []byte
	output      []byte
	writeOutput bool
}

func (r *fakeVoiceRunner) Run(ctx context.Context, command string, args []string, stdin []byte) ([]byte, error) {
	r.command = command
	r.args = args
	r.stdin = stdin

	if r.writeOutput {
		for i, arg := range args {
			if arg == "--output_file" && i+1 < len(args) {
				if err := os.WriteFile(args[i+1], []byte("wav-bytes"), 0o600); err != nil {
					return nil, err
				}
			}
		}
	}

	return r.output, nil
}

func containsArg(args []string, expected string) bool {
	for _, arg := range args {
		if strings.Contains(arg, expected) {
			return true
		}
	}
	return false
}
