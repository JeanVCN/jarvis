package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/jean/jarvis/backend/internal/opencode"
)

type codeBridge interface {
	Status(context.Context) (opencode.Status, error)
	CreateSession(context.Context, string, string) (opencode.SessionCreated, error)
	FindCreatedSession(context.Context, string, string) (*opencode.SessionCreated, error)
	Session(context.Context, string, string) (opencode.SessionDetail, error)
	Prompt(context.Context, string, string, string, string) (opencode.PromptAccepted, error)
}

type messageRouter struct {
	db      *sql.DB
	project string
	bridge  codeBridge
	chat    func(context.Context, string) string
}

func (r messageRouter) init(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS jarvis_route (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			code INTEGER NOT NULL DEFAULT 0
		);
		INSERT OR IGNORE INTO jarvis_route (id) VALUES (1);
		CREATE TABLE IF NOT EXISTS jarvis_route_events (
			message_id TEXT PRIMARY KEY
		);
		CREATE TABLE IF NOT EXISTS jarvis_code_sessions (
			project TEXT PRIMARY KEY,
			create_id TEXT NOT NULL,
			session_id TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE IF NOT EXISTS jarvis_codex_thread (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			thread_id TEXT NOT NULL
		);
	`)
	return err
}

func (r messageRouter) reply(ctx context.Context, id, text string) (string, error) {
	return r.replyWithModePolicy(ctx, id, text, true)
}

// A transcript follows the current route but can never switch it. Only a
// typed message may toggle /code.
func (r messageRouter) replyTranscription(ctx context.Context, id, text string) (string, error) {
	return r.replyWithModePolicy(ctx, id, text, false)
}

func (r messageRouter) replyWithModePolicy(ctx context.Context, id, text string, allowModeToggle bool) (string, error) {
	// Commit mode changes and deduplication together. Persist only routing
	// metadata, so a replay cannot forward a code prompt to GPT.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO jarvis_route_events VALUES (?)", id)
	if err != nil {
		return "", err
	}
	inserted, err := result.RowsAffected()
	if err != nil || inserted == 0 {
		return "", err
	}
	toggle := allowModeToggle && text == "/code"
	if toggle {
		if _, err := tx.ExecContext(ctx, "UPDATE jarvis_route SET code = NOT code WHERE id = 1"); err != nil {
			return "", err
		}
	}
	var code bool
	if err := tx.QueryRowContext(ctx, "SELECT code FROM jarvis_route WHERE id = 1").Scan(&code); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	if toggle {
		if code {
			return "Modo OpenCode ativado. Envie /code para voltar ao chat normal.", nil
		}
		return "Modo OpenCode desativado. As próximas mensagens vão para o GPT normal.", nil
	}
	if code {
		return r.codeReply(ctx, text), nil
	}
	return r.chat(ctx, text), nil
}

func (r messageRouter) codeReply(ctx context.Context, text string) string {
	const unavailable = "OpenCode indisponível. A mensagem não foi enviada ao GPT normal."
	if r.bridge == nil || r.project == "" {
		return unavailable
	}
	status, err := r.bridge.Status(ctx)
	if err != nil || status.Status != "connected" || !status.WriteEnabled {
		return unavailable
	}
	if len(text) > 8000 {
		return "A mensagem excede o limite de 8.000 bytes do OpenCode."
	}

	sessionID, err := r.codeSession(ctx)
	if err != nil {
		return "Não consegui confirmar a sessão do OpenCode. Nenhuma tarefa foi reenviada."
	}
	detail, err := r.bridge.Session(ctx, r.project, sessionID)
	if err != nil {
		return unavailable
	}
	if detail.State == "running" || len(detail.Permissions) > 0 {
		return "A sessão do OpenCode está ocupada. Veja a tarefa ou a aprovação pendente no OpenCode local."
	}

	accepted, err := r.bridge.Prompt(ctx, r.project, sessionID, operationID(), text)
	if err != nil || accepted.InboxID == "" {
		return "Não consegui confirmar se o OpenCode recebeu a tarefa. Confira a sessão local antes de reenviar."
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		detail, err := r.bridge.Session(ctx, r.project, sessionID)
		if err == nil {
			if len(detail.Permissions) > 0 {
				return "O OpenCode aguarda aprovação. Confira o pedido no OpenCode local."
			}
			if detail.State == "idle" {
				if reply := codeAnswer(detail.Messages, accepted.InboxID); reply != "" {
					return reply
				}
			}
		}
		select {
		case <-ctx.Done():
			return "O OpenCode recebeu a tarefa, mas a resposta ainda não foi confirmada. Confira a sessão local antes de reenviar."
		case <-ticker.C:
		}
	}
}

func (r messageRouter) codeSession(ctx context.Context) (string, error) {
	createID := operationID()
	result, err := r.db.ExecContext(ctx, "INSERT OR IGNORE INTO jarvis_code_sessions (project, create_id) VALUES (?, ?)", r.project, createID)
	if err != nil {
		return "", err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	var session *opencode.SessionCreated
	if inserted == 1 {
		created, err := r.bridge.CreateSession(ctx, r.project, createID)
		if err != nil {
			return "", err
		}
		session = &created
	} else {
		var sessionID string
		if err := r.db.QueryRowContext(ctx, "SELECT create_id, session_id FROM jarvis_code_sessions WHERE project = ?", r.project).Scan(&createID, &sessionID); err != nil {
			return "", err
		}
		if sessionID != "" {
			return sessionID, nil
		}
		session, err = r.bridge.FindCreatedSession(ctx, r.project, createID)
		if err != nil {
			return "", err
		}
	}
	if session == nil || session.ID == "" {
		return "", fmt.Errorf("sessão do OpenCode sem vínculo confirmado")
	}
	_, err = r.db.ExecContext(ctx, "UPDATE jarvis_code_sessions SET session_id = ? WHERE project = ?", session.ID, r.project)
	return session.ID, err
}

func codeAnswer(messages []opencode.Message, inboxID string) string {
	found := false
	answer := ""
	for _, message := range messages {
		if message.Role == "user" {
			if found {
				break
			}
			found = message.ID == inboxID
		}
		if found && message.Role == "assistant" && message.Completed {
			if message.Finish == "error" {
				return "O OpenCode informou uma falha durante a tarefa."
			}
			answer = strings.TrimSpace(message.Content)
		}
	}
	return answer
}

func operationID() string {
	var id [16]byte
	rand.Read(id[:])
	return hex.EncodeToString(id[:])
}
