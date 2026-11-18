package main

import "testing"

func TestFormatWhatsAppAssistantReplyUsesBoldNameOnOwnLine(t *testing.T) {
	got := formatWhatsAppAssistantReply(" Jarvis ", "Olá **mundo**")
	want := "*Jarvis*\n\nOlá *mundo*"
	if got != want {
		t.Fatalf("formatted reply = %q, want %q", got, want)
	}
}

func TestFormatWhatsAppMarkdownSupportsLongerFenceDelimiters(t *testing.T) {
	input := "````\n```go\nfmt.Println(\"Olá\")\n```\n````\n**fora**"
	want := "````\n```go\nfmt.Println(\"Olá\")\n```\n````\n*fora*"
	if got := formatWhatsAppMarkdown(input); got != want {
		t.Fatalf("formatted markdown = %q, want %q", got, want)
	}
}
