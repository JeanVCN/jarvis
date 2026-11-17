package main

import (
	"testing"

	"go.mau.fi/whatsmeow/types"
)

func TestSelfPromptOnlyFromConfiguredAccountChat(t *testing.T) {
	ownPhone := types.NewJID("5544999999999", types.DefaultUserServer)
	ownLID := types.NewJID("123456", types.HiddenUserServer)
	otherPhone := types.NewJID("5544888888888", types.DefaultUserServer)

	for _, test := range []struct {
		name    string
		chat    types.JID
		linked  types.JID
		fromMe  bool
		group   bool
		message string
		want    string
	}{
		{"own phone chat", ownPhone, ownPhone, true, false, "Olá", "Olá"},
		{"own LID chat", ownLID, ownPhone, true, false, "Olá", "Olá"},
		{"bot reply", ownPhone, ownPhone, true, false, replyPrefix + "Olá", ""},
		{"outbound to someone else", otherPhone, ownPhone, true, false, "Olá", ""},
		{"message from another account", ownPhone, ownPhone, false, false, "Olá", ""},
		{"wrong linked account", otherPhone, otherPhone, true, false, "Olá", ""},
		{"wrong linked account with LID", ownLID, otherPhone, true, false, "Olá", ""},
		{"group", ownPhone, ownPhone, true, true, "Olá", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			source := types.MessageSource{Chat: test.chat, IsFromMe: test.fromMe, IsGroup: test.group}
			if got := selfPrompt(source, ownPhone, test.linked, ownLID, test.message); got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}
