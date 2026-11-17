// Command whatsapp runs the optional, text-only WhatsApp channel.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/jean/jarvis/backend/internal/conversation"
	"github.com/jean/jarvis/backend/internal/opencode"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

const replyPrefix = "Jarvis: "

var phoneNumber = regexp.MustCompile(`^[1-9][0-9]{7,14}$`)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	number := strings.TrimPrefix(strings.TrimSpace(os.Getenv("JARVIS_WHATSAPP_OWNER")), "+")
	if !phoneNumber.MatchString(number) {
		return fmt.Errorf("defina JARVIS_WHATSAPP_OWNER com o número autorizado, incluindo país e DDD")
	}
	owner := types.NewJID(number, types.DefaultUserServer)

	provider, err := conversation.NewCodexText(os.Getenv("JARVIS_WHATSAPP_MODEL"))
	if err != nil {
		return fmt.Errorf("configure o perfil autenticado do Codex no Jarvis: %w", err)
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "jarvis", "whatsapp")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	dbPath, err := accountStorePath(ctx, dir, owner)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	chmodErr := file.Chmod(0600)
	closeErr := file.Close()
	if chmodErr != nil {
		return chmodErr
	}
	if closeErr != nil {
		return closeErr
	}

	dsn := url.URL{Scheme: "file", Path: dbPath, RawQuery: "_foreign_keys=on&_busy_timeout=5000"}
	store, err := sqlstore.New(ctx, "sqlite3", dsn.String(), nil)
	if err != nil {
		return fmt.Errorf("open WhatsApp device store: %w", err)
	}
	defer store.Close()
	routesDB, err := sql.Open("sqlite3", dsn.String())
	if err != nil {
		return err
	}
	defer routesDB.Close()
	router := messageRouter{
		db:      routesDB,
		project: strings.TrimSpace(os.Getenv("JARVIS_WHATSAPP_OPENCODE_PROJECT")),
		chat:    func(ctx context.Context, prompt string) string { return generateReply(ctx, provider, prompt) },
	}
	if router.project != "" {
		bridgeURL := os.Getenv("JARVIS_OPENCODE_BRIDGE_URL")
		if bridgeURL == "" {
			bridgeURL = "http://127.0.0.1:18110"
		}
		bridge, err := opencode.NewClient(bridgeURL, os.Getenv("JARVIS_OPENCODE_BRIDGE_TOKEN"))
		if err != nil {
			return fmt.Errorf("configurar bridge do OpenCode: %w", err)
		}
		router.bridge = bridge
	}
	if err := router.init(ctx); err != nil {
		return fmt.Errorf("carregar roteamento do WhatsApp: %w", err)
	}

	device, err := store.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("load WhatsApp device: %w", err)
	}
	if device.ID != nil && device.GetJID().ToNonAD() != owner {
		return fmt.Errorf("a conta vinculada não corresponde a JARVIS_WHATSAPP_OWNER")
	}
	client := whatsmeow.NewClient(device, nil)
	client.EnableAutoReconnect = true
	client.ManualHistorySyncDownload = true
	client.PrePairCallback = func(jid types.JID, _, _ string) bool {
		if jid.ToNonAD() == owner {
			return true
		}
		log.Print("QR recusado: a conta não corresponde a JARVIS_WHATSAPP_OWNER")
		return false
	}
	defer client.Disconnect()

	ctx, stopRouting := context.WithCancelCause(ctx)
	defer stopRouting(nil)
	incoming := make(chan *events.Message, 16)
	loggedOut := make(chan struct{}, 1)
	client.AddEventHandler(func(event any) {
		switch event := event.(type) {
		case *events.Connected:
			log.Print("WhatsApp conectado")
		case *events.LoggedOut:
			select {
			case loggedOut <- struct{}{}:
			default:
			}
		case *events.Message:
			if incomingText(event) == "" {
				return
			}
			select {
			case incoming <- event:
			default:
				stopRouting(errors.New("fila do WhatsApp cheia; roteamento interrompido"))
			}
		}
	})

	if device.ID == nil {
		qr, err := client.GetQRChannel(ctx)
		if err != nil {
			return err
		}
		if err := client.ConnectContext(ctx); err != nil {
			return err
		}
		for item := range qr {
			switch item.Event {
			case whatsmeow.QRChannelEventCode:
				fmt.Println("Na conta escolhida, abra Aparelhos conectados e escaneie este QR:")
				qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, os.Stdout)
			case "success":
				if device.GetJID().ToNonAD() != owner {
					return fmt.Errorf("a conta vinculada não corresponde a JARVIS_WHATSAPP_OWNER")
				}
				log.Print("Conta do Jarvis vinculada")
			default:
				return fmt.Errorf("pareamento do WhatsApp: %s", item.Event)
			}
		}
	} else if err := client.ConnectContext(ctx); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return context.Cause(ctx)
		case <-loggedOut:
			return fmt.Errorf("o WhatsApp desvinculou o Jarvis; inicie o processo novamente para parear")
		case event := <-incoming:
			if ctx.Err() != nil {
				return context.Cause(ctx)
			}
			prompt := selfPrompt(event.Info.MessageSource, owner, device.GetJID(), device.GetLID(), incomingText(event))
			if prompt == "" {
				continue
			}
			replyCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			reply, err := router.reply(replyCtx, string(event.Info.ID), prompt)
			cancel()
			if err != nil {
				return fmt.Errorf("roteamento do WhatsApp interrompido: %w", err)
			}
			if reply == "" {
				continue
			}
			sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			_, err = client.SendMessage(sendCtx, event.Info.Chat.ToNonAD(), &waE2E.Message{Conversation: proto.String(replyPrefix + reply)})
			cancel()
			if err != nil {
				if prompt == "/code" {
					return fmt.Errorf("não consegui confirmar o modo pelo WhatsApp; roteamento interrompido")
				}
				log.Print("Falha ao enviar resposta no WhatsApp")
			}
		}
	}
}

// Reuse an existing single-account database only for its original owner.
func accountStorePath(ctx context.Context, dir string, owner types.JID) (string, error) {
	accountPath := filepath.Join(dir, "device-"+owner.User+".db")
	if _, err := os.Stat(accountPath); !errors.Is(err, os.ErrNotExist) {
		return accountPath, err
	}

	legacyPath := filepath.Join(dir, "device.db")
	if _, err := os.Stat(legacyPath); errors.Is(err, os.ErrNotExist) {
		return accountPath, nil
	} else if err != nil {
		return "", err
	}

	dsn := url.URL{Scheme: "file", Path: legacyPath, RawQuery: "mode=ro"}
	db, err := sql.Open("sqlite3", dsn.String())
	if err != nil {
		return "", err
	}
	defer db.Close()

	var schemaExists bool
	err = db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type='table' AND name='whatsmeow_device')").Scan(&schemaExists)
	if err != nil || !schemaExists {
		return accountPath, err
	}

	var jid string
	err = db.QueryRowContext(ctx, "SELECT jid FROM whatsmeow_device LIMIT 1").Scan(&jid)
	if errors.Is(err, sql.ErrNoRows) {
		return accountPath, nil
	}
	if err != nil {
		return "", fmt.Errorf("verificar pareamento anterior: %w", err)
	}
	linked, err := types.ParseJID(jid)
	if err != nil {
		return "", fmt.Errorf("ler conta do pareamento anterior: %w", err)
	}
	if linked.ToNonAD() == owner {
		return legacyPath, nil
	}
	return accountPath, nil
}

func incomingText(event *events.Message) string {
	info := event.Info
	if info.ID == "" || info.IsGroup || event.IsEdit || event.SourceWebMsg != nil ||
		(info.Chat.Server != types.DefaultUserServer && info.Chat.Server != types.HiddenUserServer) {
		return ""
	}
	body := strings.TrimSpace(event.Message.GetConversation())
	if body == "" {
		body = strings.TrimSpace(event.Message.GetExtendedTextMessage().GetText())
	}
	return body
}

func selfPrompt(source types.MessageSource, owner, ownPhone, ownLID types.JID, body string) string {
	if !source.IsFromMe || source.IsGroup || ownPhone.ToNonAD() != owner {
		return ""
	}
	chat := source.Chat.ToNonAD()
	if chat.IsEmpty() || (chat != ownPhone.ToNonAD() && chat != ownLID.ToNonAD()) {
		return ""
	}
	// Replies can also arrive as events from the linked account. Never feed
	// our own output back into inference, including after a restart.
	if strings.HasPrefix(body, replyPrefix) {
		return ""
	}
	return body
}

func generateReply(ctx context.Context, provider conversation.Provider, prompt string) string {
	var response strings.Builder
	calls, err := provider.Stream(ctx, []conversation.Message{{Role: "user", Content: prompt}}, func(delta string) error {
		response.WriteString(delta)
		return nil
	})
	reply := strings.TrimSpace(response.String())
	if err == nil && len(calls) == 0 && reply != "" {
		return reply
	}
	if err == nil && len(calls) > 0 {
		return "Por enquanto, só respondo por texto neste chat."
	}
	if failure, ok := err.(*conversation.Failure); ok {
		switch failure.Code {
		case "chatgpt_login_required", "authentication_required":
			return "O GPT precisa que você entre novamente no ChatGPT no perfil Codex do Jarvis."
		case "quota_exhausted":
			return "O limite de uso do GPT foi atingido. Tente mais tarde."
		case "model_unavailable":
			return "O modelo GPT configurado está indisponível para esta conta."
		}
	}
	log.Print("GPT indisponível para responder à mensagem do WhatsApp")
	return "Não consegui obter uma resposta do GPT agora. Tente novamente daqui a pouco."
}
