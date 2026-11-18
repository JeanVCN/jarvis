package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
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

type imageCodeBridge interface {
	PromptImage(context.Context, string, string, string, string, string, []byte) (opencode.PromptAccepted, error)
}

type interactiveCodeBridge interface {
	ReplyPermission(context.Context, string, string, string, string, string) error
	ReplyForm(context.Context, string, string, string, string, map[string]any) error
}

type imageAttachment struct {
	mime string
	data []byte
}

type messageRouter struct {
	db        *sql.DB
	project   string
	bridge    codeBridge
	chat      func(context.Context, string) string
	imageChat func(context.Context, string, string, []byte) string
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
		CREATE TABLE IF NOT EXISTS jarvis_assistant_message_ids (
			message_id TEXT PRIMARY KEY
		);
		CREATE TABLE IF NOT EXISTS jarvis_code_interactions (
			kind TEXT NOT NULL CHECK (kind IN ('permission', 'form')),
			position INTEGER NOT NULL,
			session_id TEXT NOT NULL,
			item_id TEXT NOT NULL,
			PRIMARY KEY (kind, position)
		);
	`)
	if err != nil {
		return err
	}
	return initializeTOTP(ctx, r.db)
}

func (r messageRouter) codeMode(ctx context.Context) (bool, error) {
	var code bool
	err := r.db.QueryRowContext(ctx, "SELECT code FROM jarvis_route WHERE id = 1").Scan(&code)
	return code, err
}

func isAssistantMessageID(ctx context.Context, db *sql.DB, id string) (bool, error) {
	var exists int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM jarvis_assistant_message_ids WHERE message_id = ?", id).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (r messageRouter) rememberCodeInteractions(ctx context.Context, kind, sessionID string, ids []string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "DELETE FROM jarvis_code_interactions WHERE kind = ?", kind); err != nil {
		return err
	}
	for i, id := range ids {
		if _, err := tx.ExecContext(ctx, "INSERT INTO jarvis_code_interactions (kind, position, session_id, item_id) VALUES (?, ?, ?, ?)", kind, i+1, sessionID, id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r messageRouter) codeInteractionID(ctx context.Context, kind, sessionID string, position int) (string, error) {
	var id string
	err := r.db.QueryRowContext(ctx, "SELECT item_id FROM jarvis_code_interactions WHERE kind = ? AND position = ? AND session_id = ?", kind, position, sessionID).Scan(&id)
	return id, err
}

func (r messageRouter) clearCodeInteractions(ctx context.Context, kind string) error {
	_, err := r.db.ExecContext(ctx, "DELETE FROM jarvis_code_interactions WHERE kind = ?", kind)
	return err
}

func (r messageRouter) reply(ctx context.Context, id, text string) (string, error) {
	return r.replyWithModePolicy(ctx, id, text, true, nil)
}

// A transcript follows the current route but can never switch it. Only a
// typed message may toggle /code.
func (r messageRouter) replyTranscription(ctx context.Context, id, text string) (string, error) {
	return r.replyWithModePolicy(ctx, id, text, false, nil)
}

func (r messageRouter) replyImage(ctx context.Context, id, caption, mime string, data []byte) (string, error) {
	if caption == "" {
		caption = "Analise a imagem enviada."
	}
	return r.replyWithModePolicy(ctx, id, caption, false, &imageAttachment{mime: mime, data: data})
}

func (r messageRouter) replyWithModePolicy(ctx context.Context, id, text string, allowModeToggle bool, image *imageAttachment) (string, error) {
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
	var currentCode bool
	if err := tx.QueryRowContext(ctx, "SELECT code FROM jarvis_route WHERE id = 1").Scan(&currentCode); err != nil {
		return "", err
	}
	var challengePending bool
	if err := tx.QueryRowContext(ctx, "SELECT challenge_pending FROM jarvis_totp WHERE id = 1").Scan(&challengePending); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if isTOTPCommand(text) {
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return r.handleTOTPCommand(allowModeToggle), nil
	}
	toggle := allowModeToggle && text == "/code"
	if toggle && currentCode {
		if _, err := tx.ExecContext(ctx, "UPDATE jarvis_route SET code=0 WHERE id=1"); err != nil {
			return "", err
		}
		_, _ = tx.ExecContext(ctx, "UPDATE jarvis_totp SET challenge_pending=0,authorized_until=0 WHERE id=1")
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return "Modo OpenCode desativado. As próximas mensagens vão para o GPT normal.", nil
	}
	if toggle {
		configured, _, err := totpAuthorization(ctx, tx)
		if err != nil {
			return "", err
		}
		if !configured {
			if err := tx.Commit(); err != nil {
				return "", err
			}
			return "O autenticador ainda não foi configurado. Pare o serviço e execute `JARVIS_WHATSAPP_OWNER=... go run . totp-setup` no terminal local.", nil
		}
		if _, err := tx.ExecContext(ctx, "UPDATE jarvis_totp SET challenge_pending=1,authorized_until=0 WHERE id=1"); err != nil {
			return "", err
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return "Autenticação necessária para entrar no `/code`. Responda agora somente com os seis dígitos do Google Authenticator.", nil
	}
	if challengePending {
		if err := tx.Commit(); err != nil {
			return "", err
		}
		if !allowModeToggle {
			return "A autenticação está pendente. Envie o código de seis dígitos como mensagem digitada; não encaminhei esse conteúdo.", nil
		}
		if !totpCodePattern.MatchString(strings.TrimSpace(text)) {
			return "Autenticação pendente. Responda somente com os seis dígitos do Google Authenticator.", nil
		}
		message, valid, err := r.verifyTOTP(ctx, strings.TrimSpace(text), time.Now())
		if err != nil {
			return "Não consegui verificar o autenticador. Nenhuma ação foi enviada ao OpenCode.", nil
		}
		if !valid {
			return message, nil
		}
		activate, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			return "", err
		}
		defer activate.Rollback()
		if _, err := activate.ExecContext(ctx, "UPDATE jarvis_route SET code=1 WHERE id=1"); err != nil {
			return "", err
		}
		if _, err := activate.ExecContext(ctx, "UPDATE jarvis_totp SET challenge_pending=0 WHERE id=1"); err != nil {
			return "", err
		}
		if err := activate.Commit(); err != nil {
			return "", err
		}
		return message + " Modo OpenCode ativado; pode enviar a tarefa.", nil
	}
	if currentCode {
		configured, authorized, err := totpAuthorization(ctx, tx)
		if err != nil {
			return "", err
		}
		if !configured {
			if err := tx.Commit(); err != nil {
				return "", err
			}
			return "O autenticador ainda não foi configurado. Pare o serviço e execute `JARVIS_WHATSAPP_OWNER=... go run . totp-setup` no terminal local. Envie `/code` para sair do modo OpenCode.", nil
		}
		if !authorized {
			if _, err := tx.ExecContext(ctx, "UPDATE jarvis_totp SET challenge_pending=1 WHERE id=1"); err != nil {
				return "", err
			}
			if err := tx.Commit(); err != nil {
				return "", err
			}
			return "A autorização do `/code` expirou. Responda agora somente com os seis dígitos do Google Authenticator para continuar.", nil
		}
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	if currentCode {
		if image != nil {
			return r.codeImageReply(ctx, text, image.mime, image.data), nil
		}
		return r.codeReply(ctx, text), nil
	}
	if image != nil {
		if r.imageChat == nil {
			return "O encaminhamento de imagens ao GPT não está disponível nesta configuração.", nil
		}
		return r.imageChat(ctx, text, image.mime, image.data), nil
	}
	return r.chat(ctx, text), nil
}

func (r messageRouter) codeReply(ctx context.Context, text string) string {
	return r.codeReplyWithImage(ctx, text, "", nil)
}

func (r messageRouter) codeImageReply(ctx context.Context, text, mime string, image []byte) string {
	return r.codeReplyWithImage(ctx, text, mime, image)
}

func (r messageRouter) codeReplyWithImage(ctx context.Context, text, mime string, image []byte) string {
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
	decision, permissionIndex, permissionCommand, validPermissionCommand := parsePermissionCommand(text)
	if permissionCommand {
		if !validPermissionCommand {
			return "Use /aprovar ou /rejeitar. Se houver vários pedidos, informe também o número, por exemplo: /aprovar 2."
		}
		if len(detail.Permissions) == 0 {
			_ = r.clearCodeInteractions(ctx, "permission")
			return "Não há pedido de permissão pendente nesta sessão."
		}
		if permissionIndex == 0 {
			if len(detail.Permissions) != 1 {
				return r.showPendingPermissions(ctx, sessionID, detail.Permissions)
			}
			permissionIndex = 1
		}
		if permissionIndex < 1 || permissionIndex > len(detail.Permissions) {
			return r.showPendingPermissions(ctx, sessionID, detail.Permissions)
		}
		permissionID, err := r.codeInteractionID(ctx, "permission", sessionID, permissionIndex)
		if err != nil {
			return r.showPendingPermissions(ctx, sessionID, detail.Permissions)
		}
		var pending *opencode.Permission
		for i := range detail.Permissions {
			if detail.Permissions[i].ID == permissionID {
				pending = &detail.Permissions[i]
				break
			}
		}
		if pending == nil {
			return "A lista de pedidos mudou desde o último aviso.\n\n" + r.showPendingPermissions(ctx, sessionID, detail.Permissions)
		}
		bridge, ok := r.bridge.(interactiveCodeBridge)
		if !ok {
			return "Este bridge não permite responder a pedidos do OpenCode pelo WhatsApp."
		}
		previous := latestAssistantMessage(detail.Messages)
		if err := bridge.ReplyPermission(ctx, r.project, sessionID, pending.ID, operationID(), decision); err != nil {
			return "Não consegui confirmar a decisão. Confira o estado da sessão antes de tentar novamente."
		}
		_ = r.clearCodeInteractions(ctx, "permission")
		return r.waitForCodeResponse(ctx, sessionID, previous, pending.ID)
	}
	if len(detail.Permissions) > 0 {
		return r.showPendingPermissions(ctx, sessionID, detail.Permissions)
	}
	if len(detail.Forms) > 0 {
		formID, err := r.codeInteractionID(ctx, "form", sessionID, 1)
		if err != nil {
			return r.showPendingForm(ctx, sessionID, detail.Forms[0], latestAssistantContent(detail.Messages))
		}
		var form *opencode.Form
		for i := range detail.Forms {
			if detail.Forms[i].ID == formID {
				form = &detail.Forms[i]
				break
			}
		}
		if form == nil {
			_ = r.clearCodeInteractions(ctx, "form")
			return r.showPendingForm(ctx, sessionID, detail.Forms[0], latestAssistantContent(detail.Messages))
		}
		answer, err := answerOpenCodeForm(*form, text)
		if err != nil {
			return formatOpenCodeForm(*form, latestAssistantContent(detail.Messages)) + "\n\n" + err.Error()
		}
		bridge, ok := r.bridge.(interactiveCodeBridge)
		if !ok {
			return "Este bridge não permite responder às perguntas do OpenCode pelo WhatsApp."
		}
		previous := latestAssistantMessage(detail.Messages)
		if err := bridge.ReplyForm(ctx, r.project, sessionID, form.ID, operationID(), answer); err != nil {
			return "Não consegui enviar essa resposta ao OpenCode.\n\n" + formatOpenCodeForm(*form, latestAssistantContent(detail.Messages))
		}
		_ = r.clearCodeInteractions(ctx, "form")
		return r.waitForCodeResponse(ctx, sessionID, previous, form.ID)
	}
	if detail.State == "running" {
		return "A sessão do OpenCode ainda está trabalhando. Aguarde a resposta antes de enviar outra mensagem."
	}

	operation := operationID()
	var accepted opencode.PromptAccepted
	if len(image) > 0 {
		imageBridge, ok := r.bridge.(imageCodeBridge)
		if !ok {
			return "O bridge do OpenCode não aceita imagens. A imagem não foi encaminhada."
		}
		accepted, err = imageBridge.PromptImage(ctx, r.project, sessionID, operation, text, mime, image)
	} else {
		accepted, err = r.bridge.Prompt(ctx, r.project, sessionID, operation, text)
	}
	if err != nil || accepted.InboxID == "" {
		return "Não consegui confirmar se o OpenCode recebeu a tarefa. Confira a sessão local antes de reenviar."
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		detail, err := r.bridge.Session(ctx, r.project, sessionID)
		if err == nil {
			if len(detail.Permissions) > 0 {
				return r.showPendingPermissions(ctx, sessionID, detail.Permissions)
			}
			if len(detail.Forms) > 0 {
				return r.showPendingForm(ctx, sessionID, detail.Forms[0], latestAssistantContent(detail.Messages))
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

func (r messageRouter) showPendingPermissions(ctx context.Context, sessionID string, items []opencode.Permission) string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	if err := r.rememberCodeInteractions(ctx, "permission", sessionID, ids); err != nil {
		return "Há um pedido de autorização, mas não consegui guardar a lista com segurança. Nenhuma decisão foi enviada."
	}
	return formatPendingPermissions(items)
}

func (r messageRouter) showPendingForm(ctx context.Context, sessionID string, item opencode.Form, contextText string) string {
	if err := r.rememberCodeInteractions(ctx, "form", sessionID, []string{item.ID}); err != nil {
		return "Há uma pergunta do OpenCode, mas não consegui guardar seu identificador. Nenhuma resposta foi enviada."
	}
	return formatOpenCodeForm(item, contextText)
}

func (r messageRouter) waitForCodeResponse(ctx context.Context, sessionID string, previous opencode.Message, handledID string) string {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		detail, err := r.bridge.Session(ctx, r.project, sessionID)
		if err == nil {
			permissions := withoutPermission(detail.Permissions, handledID)
			if len(permissions) > 0 {
				return r.showPendingPermissions(ctx, sessionID, permissions)
			}
			forms := withoutForm(detail.Forms, handledID)
			if len(forms) > 0 {
				return r.showPendingForm(ctx, sessionID, forms[0], latestAssistantContent(detail.Messages))
			}
			if detail.State == "idle" {
				latest := latestAssistantMessage(detail.Messages)
				if latest.ID != "" && (latest.ID != previous.ID || latest.Content != previous.Content) {
					return latest.Content
				}
			}
		}
		select {
		case <-ctx.Done():
			return "O OpenCode recebeu sua resposta, mas ainda não confirmou a conclusão. A sessão continua no mesmo contexto."
		case <-ticker.C:
		}
	}
}

func parsePermissionCommand(text string) (decision string, index int, matched, valid bool) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 {
		return "", 0, false, false
	}
	switch strings.ToLower(fields[0]) {
	case "/aprovar":
		decision = "once"
	case "/rejeitar":
		decision = "reject"
	default:
		return "", 0, false, false
	}
	if len(fields) == 1 {
		return decision, 0, true, true
	}
	if len(fields) != 2 {
		return decision, 0, true, false
	}
	index, err := strconv.Atoi(fields[1])
	return decision, index, true, err == nil && index > 0
}

func formatPendingPermissions(permissions []opencode.Permission) string {
	var message strings.Builder
	message.WriteString("O OpenCode pede autorização:\n")
	for i, permission := range permissions {
		message.WriteString(fmt.Sprintf("%d. %s", i+1, permission.Action))
		if permission.Message != "" {
			message.WriteString(" — " + permission.Message)
		}
		if len(permission.Resources) > 0 {
			message.WriteString("\n   Recursos: " + strings.Join(permission.Resources, ", "))
		}
		message.WriteByte('\n')
	}
	if len(permissions) == 1 {
		message.WriteString("Responda /aprovar para permitir somente esta ação ou /rejeitar para negar.")
	} else {
		message.WriteString("Responda /aprovar N ou /rejeitar N, usando o número do pedido.")
	}
	return strings.TrimSpace(message.String())
}

func formatOpenCodeForm(form opencode.Form, contextText string) string {
	var message strings.Builder
	message.WriteString("OpenCode pergunta")
	if form.Title != "" {
		message.WriteString(": " + form.Title)
	}
	if contextText = strings.TrimSpace(contextText); contextText != "" {
		message.WriteString("\n\n" + contextText)
	}
	for fieldIndex, field := range form.Fields {
		message.WriteByte('\n')
		if field.Title != "" {
			message.WriteString(field.Title)
		} else if field.Description != "" {
			message.WriteString(field.Description)
		} else {
			message.WriteString("Resposta")
		}
		if field.Title != "" && field.Description != "" {
			message.WriteString(" — " + field.Description)
		}
		for optionIndex, option := range field.Options {
			message.WriteString(fmt.Sprintf("\n%d) %s", optionIndex+1, option.Label))
			if option.Description != "" {
				message.WriteString(" — " + option.Description)
			}
			if option.Recommended && !strings.Contains(strings.ToLower(option.Label+" "+option.Description), "recommended") {
				message.WriteString(" (Recommended)")
			}
		}
		if field.Custom || len(field.Options) == 0 {
			message.WriteString("\nVocê também pode responder com texto personalizado.")
		}
		if len(form.Fields) > 1 && fieldIndex == 0 {
			message.WriteString("\nPara preencher vários campos, responda na ordem, separando os valores por ponto e vírgula.")
		}
	}
	return message.String()
}

func answerOpenCodeForm(form opencode.Form, text string) (map[string]any, error) {
	if len(form.Fields) == 0 {
		return nil, fmt.Errorf("o formulário não contém campos respondíveis")
	}
	values := []string{strings.TrimSpace(text)}
	if len(form.Fields) > 1 {
		values = strings.Split(text, ";")
		if len(values) != len(form.Fields) {
			return nil, fmt.Errorf("responda os %d campos na ordem, separados por ponto e vírgula", len(form.Fields))
		}
	}
	answer := make(map[string]any, len(form.Fields))
	for i, field := range form.Fields {
		value := strings.TrimSpace(values[i])
		if value == "" {
			return nil, fmt.Errorf("a resposta do campo %q está vazia", field.Title)
		}
		if number, err := strconv.Atoi(value); err == nil && len(field.Options) > 0 {
			if number < 1 || number > len(field.Options) {
				return nil, fmt.Errorf("escolha um número entre 1 e %d para %q", len(field.Options), field.Title)
			}
			option := field.Options[number-1]
			switch field.Type {
			case "multiselect":
				answer[field.Key] = []string{option.Value}
			default:
				answer[field.Key] = option.Value
			}
			continue
		}
		switch field.Type {
		case "string":
			answer[field.Key] = value
		case "multiselect":
			if !field.Custom {
				return nil, fmt.Errorf("escolha o número de uma opção para %q", field.Title)
			}
			answer[field.Key] = []string{value}
		case "integer":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("responda %q com um número inteiro", field.Title)
			}
			answer[field.Key] = parsed
		case "number":
			parsed, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return nil, fmt.Errorf("responda %q com um número", field.Title)
			}
			answer[field.Key] = parsed
		case "boolean":
			switch strings.ToLower(value) {
			case "sim", "s", "yes", "true", "1":
				answer[field.Key] = true
			case "não", "nao", "n", "no", "false", "0":
				answer[field.Key] = false
			default:
				return nil, fmt.Errorf("responda %q com sim ou não", field.Title)
			}
		default:
			return nil, fmt.Errorf("o campo %q não pode ser respondido pelo WhatsApp", field.Title)
		}
	}
	return answer, nil
}

func latestAssistantMessage(messages []opencode.Message) opencode.Message {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" && messages[i].Completed {
			return messages[i]
		}
	}
	return opencode.Message{}
}

func latestAssistantContent(messages []opencode.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "assistant" && strings.TrimSpace(messages[i].Content) != "" {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func withoutPermission(items []opencode.Permission, id string) []opencode.Permission {
	filtered := make([]opencode.Permission, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func withoutForm(items []opencode.Form, id string) []opencode.Form {
	filtered := make([]opencode.Form, 0, len(items))
	for _, item := range items {
		if item.ID != id {
			filtered = append(filtered, item)
		}
	}
	return filtered
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
