package main

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
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

func TestRedactedJIDLogDoesNotExposeFullUser(t *testing.T) {
	jid := types.NewJID("15550001234", types.DefaultUserServer)
	got := redactedJIDForLog(jid)
	if strings.Contains(got, jid.User) {
		t.Fatalf("log contains full JID user: %q", got)
	}
	if !strings.Contains(got, "user_len=11") || !strings.Contains(got, "user_suffix=34") || !strings.Contains(got, "server=\"s.whatsapp.net\"") {
		t.Fatalf("log is missing useful redacted metadata: %q", got)
	}
}

func TestQuotedWhatsAppReplyReferencesOriginalSelfChatMessage(t *testing.T) {
	chat := types.NewJID("15550001234", types.DefaultUserServer)
	sender := types.NewJID("15550001234", types.DefaultUserServer)
	event := &events.Message{
		Info: types.MessageInfo{
			MessageSource: types.MessageSource{Chat: chat, Sender: sender, IsFromMe: true},
			ID:            types.MessageID("original-message"),
		},
		Message: &waE2E.Message{Conversation: proto.String("pergunta original")},
	}

	reply := quotedWhatsAppReply("resposta do Jarvis", event)
	if reply.GetExtendedTextMessage() == nil {
		t.Fatal("quoted reply should use an extended text message")
	}
	if reply.GetExtendedTextMessage().GetText() != "resposta do Jarvis" {
		t.Fatalf("reply text = %q", reply.GetExtendedTextMessage().GetText())
	}
	contextInfo := reply.GetExtendedTextMessage().GetContextInfo()
	if contextInfo.GetStanzaID() != "original-message" {
		t.Fatalf("quoted stanza ID = %q", contextInfo.GetStanzaID())
	}
	if contextInfo.GetRemoteJID() != chat.String() {
		t.Fatalf("quoted remote JID = %q", contextInfo.GetRemoteJID())
	}
	if contextInfo.GetParticipant() != sender.String() {
		t.Fatalf("quoted participant = %q", contextInfo.GetParticipant())
	}
	if contextInfo.GetQuotedMessage().GetConversation() != "pergunta original" {
		t.Fatalf("quoted message = %q", contextInfo.GetQuotedMessage().GetConversation())
	}
}

func TestQuotedWhatsAppReplyFallsBackWithoutMessageReference(t *testing.T) {
	for _, event := range []*events.Message{
		nil,
		{},
		{Message: &waE2E.Message{Conversation: proto.String("original")}},
	} {
		reply := quotedWhatsAppReply("resposta", event)
		if reply.GetConversation() != "resposta" || reply.GetExtendedTextMessage() != nil {
			t.Fatalf("reply without quote reference = %#v", reply)
		}
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
