package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jean/jarvis/backend/internal/opencode"
)

func TestRoutingPersistsPerAccountAndDoesNotFallBack(t *testing.T) {
	ctx := context.Background()
	normal := 0
	chat := func(_ context.Context, text string) string {
		normal++
		return "GPT: " + text
	}
	dir := t.TempDir()
	open := func(account string) messageRouter {
		t.Helper()
		db, err := sql.Open("sqlite3", filepath.Join(dir, account+".db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { db.Close() })
		router := messageRouter{db: db, chat: chat}
		if err := router.init(ctx); err != nil {
			t.Fatal(err)
		}
		return router
	}
	check := func(router messageRouter, id, input, want string) {
		t.Helper()
		reply, err := router.reply(ctx, id, input)
		if err != nil || !strings.HasPrefix(reply, want) || (want == "" && reply != "") {
			t.Fatalf("reply = %q, %v; want prefix %q", reply, err, want)
		}
	}
	first := open("first")
	check(first, "msg1", "/code", "Modo OpenCode ativado")
	if err := first.db.Close(); err != nil {
		t.Fatal(err)
	}

	first = open("first")
	check(first, "msg1", "/code", "")
	check(first, "msg2", "conteúdo do projeto", "OpenCode indisponível")
	check(open("second"), "msg2", "oi", "GPT: oi")
	check(first, "msg3", "/code", "Modo OpenCode desativado")
	check(first, "msg2", "conteúdo do projeto", "")
	check(first, "msg4", "olá", "GPT: olá")
	if normal != 2 {
		t.Fatalf("GPT recebeu %d mensagens, esperado 2", normal)
	}
}

func TestAcknowledgementClaimIsDeduplicatedWithoutMessageContent(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "device.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	router := messageRouter{db: db}
	ctx := context.Background()
	if err := router.init(ctx); err != nil {
		t.Fatal(err)
	}
	first, err := router.claimAcknowledgement(ctx, "incoming-message-id")
	if err != nil || !first {
		t.Fatalf("first claim = %t, %v; want true, nil", first, err)
	}
	second, err := router.claimAcknowledgement(ctx, "incoming-message-id")
	if err != nil || second {
		t.Fatalf("duplicate claim = %t, %v; want false, nil", second, err)
	}
	var stored int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM jarvis_acknowledgements").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 1 {
		t.Fatalf("stored acknowledgement IDs = %d, want 1", stored)
	}
}

func TestCodeRoutingUsesLocalBridgeAndReusesSession(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "device.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const sessionID = "ses_local"
	const token = "0123456789abcdef0123456789abcdef"
	created, prompts := 0, 0
	inboxID := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer "+token {
			t.Error("missing local credentials")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var response any
		switch req.Method + " " + req.URL.Path {
		case "GET /v1/status":
			response = opencode.Status{Status: "connected", WriteEnabled: true}
		case "POST /v1/projects/jarvis/sessions":
			created++
			response = opencode.SessionCreated{ID: sessionID}
		case "POST /v1/projects/jarvis/sessions/ses_local/prompts":
			prompts++
			var input map[string]string
			if err := json.NewDecoder(req.Body).Decode(&input); err != nil || input["text"] != "tarefa local" || len(input["operation_id"]) != 32 {
				t.Errorf("unexpected prompt: %v (%v)", input, err)
			}
			inboxID = input["operation_id"]
			response = opencode.PromptAccepted{InboxID: inboxID}
		case "GET /v1/projects/jarvis/sessions/ses_local":
			response = opencode.SessionDetail{State: "idle", Messages: []opencode.Message{
				{Role: "user", ID: inboxID},
				{Role: "assistant", Completed: true, Content: "resposta direta"},
			}}
		default:
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	bridge, err := opencode.NewClient(server.URL, token)
	if err != nil {
		t.Fatal(err)
	}

	router := messageRouter{db: db, project: "jarvis", bridge: bridge, chat: func(context.Context, string) string {
		t.Error("GPT normal chamado em modo OpenCode")
		return ""
	}}
	if err := router.init(ctx); err != nil {
		t.Fatal(err)
	}
	for _, msg := range []struct{ id, input, prefix string }{
		{"1", "/code", "Modo OpenCode ativado"},
		{"2", "tarefa local", "resposta direta"},
		{"3", "tarefa local", "resposta direta"},
	} {
		reply, err := router.reply(ctx, msg.id, msg.input)
		if err != nil || !strings.HasPrefix(reply, msg.prefix) {
			t.Fatalf("reply = %q, %v", reply, err)
		}
	}
	if created != 1 || prompts != 2 {
		t.Fatalf("created %d sessions and sent %d prompts", created, prompts)
	}
}

func TestWaitForCodeResponseSchedulesFollowUpAfterDeadline(t *testing.T) {
	const token = "0123456789abcdef0123456789abcdef"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer "+token {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if req.Method != http.MethodGet || req.URL.Path != "/v1/projects/jarvis/sessions/ses_local" {
			t.Errorf("unexpected request: %s %s", req.Method, req.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(opencode.SessionDetail{State: "running"})
	}))
	defer server.Close()
	bridge, err := opencode.NewClient(server.URL, token)
	if err != nil {
		t.Fatal(err)
	}
	var watched bool
	router := messageRouter{
		project: "jarvis",
		bridge:  bridge,
		watchCode: func(sessionID string, previous opencode.Message, handledID string) {
			watched = sessionID == "ses_local" && previous.ID == "msg_previous" && handledID == "permission-id"
		},
	}
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	reply := router.waitForCodeResponse(ctx, "ses_local", opencode.Message{ID: "msg_previous"}, "permission-id")
	if !watched {
		t.Fatal("timed-out code response did not schedule a follow-up watcher")
	}
	if !strings.Contains(reply, "Vou avisar por aqui quando concluir") {
		t.Fatalf("timeout reply = %q", reply)
	}
}
