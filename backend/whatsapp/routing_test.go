package main

import (
	"context"
	"database/sql"
	"encoding/base32"
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
		if _, err := db.ExecContext(ctx, `INSERT INTO jarvis_totp(id,secret,authorized_until)
			VALUES(1,'JBSWY3DPEHPK3PXP',?) ON CONFLICT(id) DO UPDATE SET authorized_until=excluded.authorized_until,last_step=-1`, time.Now().Add(time.Hour).Unix()); err != nil {
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
	check(first, "msg1", "/code", "Autenticação necessária")
	check(first, "msg1-auth", testTOTPCode(), "Modo OpenCode ativado")
	if err := first.db.Close(); err != nil {
		t.Fatal(err)
	}

	first = open("first")
	check(first, "msg1", "/code", "")
	check(first, "msg2", "conteúdo do projeto", "A autorização do `/code` expirou")
	check(first, "msg2-auth", testTOTPCode(), "Modo OpenCode ativado")
	check(first, "msg2-task", "conteúdo do projeto", "OpenCode indisponível")
	check(open("second"), "msg2", "oi", "GPT: oi")
	check(first, "msg3", "/code", "Modo OpenCode desativado")
	check(first, "msg2", "conteúdo do projeto", "")
	check(first, "msg4", "olá", "GPT: olá")
	if normal != 2 {
		t.Fatalf("GPT recebeu %d mensagens, esperado 2", normal)
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
	if _, err := db.ExecContext(ctx, `INSERT INTO jarvis_totp(id,secret,authorized_until) VALUES(1,'JBSWY3DPEHPK3PXP',?)`, time.Now().Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
	for _, msg := range []struct{ id, input, prefix string }{
		{"1", "/code", "Autenticação necessária"},
		{"1-auth", testTOTPCode(), "Modo OpenCode ativado"},
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

func testTOTPCode() string {
	secret, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString("JBSWY3DPEHPK3PXP")
	now := time.Now()
	return totpValue(secret, now.Unix()/totpPeriodSeconds)
}
