package main

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestSelfPromptOnlyFromOwnedChatWithPrefix(t *testing.T) {
	ownPhone := types.NewJID("5544999999999", types.DefaultUserServer)
	ownLID := types.NewJID("123456", types.HiddenUserServer)
	otherPhone := types.NewJID("5544888888888", types.DefaultUserServer)

	for _, test := range []struct {
		name    string
		chat    types.JID
		fromMe  bool
		group   bool
		message string
		want    string
	}{
		{"own phone chat", ownPhone, true, false, "/jarvis  Olá", "Olá"},
		{"own LID chat", ownLID, true, false, "/jarvis Olá", "Olá"},
		{"ordinary note", ownPhone, true, false, "Comprar pão", ""},
		{"bot reply", ownPhone, true, false, "Jarvis: /jarvis Olá", ""},
		{"empty prompt", ownPhone, true, false, "/jarvis ", ""},
		{"outbound to someone else", otherPhone, true, false, "/jarvis Olá", ""},
		{"message from another account", ownPhone, false, false, "/jarvis Olá", ""},
		{"group", ownPhone, true, true, "/jarvis Olá", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := types.MessageSource{Chat: test.chat, IsFromMe: test.fromMe, IsGroup: test.group}
			if got := selfPrompt(source, ownPhone, ownLID, test.message); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
