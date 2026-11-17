package main

import (
	"context"
	"database/sql"
	"path/filepath"
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
		{"assistant-looking text is ordinary input", ownPhone, ownPhone, true, false, "Jarvis: Olá", "Jarvis: Olá"},
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

func TestAccountStoreSeparatesNumbersAndKeepsLegacyPairing(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()
	first := types.NewJID("5544999999999", types.DefaultUserServer)
	second := types.NewJID("5544888888888", types.DefaultUserServer)
	legacy := filepath.Join(dir, "device.db")
	db, err := sql.Open("sqlite3", legacy)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	check := func(owner types.JID, want string) {
		t.Helper()
		got, err := accountStorePath(ctx, dir, owner)
		if err != nil || got != filepath.Join(dir, want) {
			t.Fatalf("got %q, %v; want %q", got, err, want)
		}
	}
	check(first, "device-"+first.User+".db")
	if _, err := db.Exec("CREATE TABLE whatsmeow_device (jid TEXT PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO whatsmeow_device (jid) VALUES (?)", first.String()); err != nil {
		t.Fatal(err)
	}
	check(first, "device.db")
	check(second, "device-"+second.User+".db")
	check(first, "device.db")
}
