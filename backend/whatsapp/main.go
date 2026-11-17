// Command whatsapp runs the optional, text-only WhatsApp channel.
package main

import (
	"context"
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

	"github.com/jean/jarvis/backend/internal/ai"
	"github.com/jean/jarvis/backend/internal/config"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	"google.golang.org/protobuf/proto"
)

var phoneNumber = regexp.MustCompile(`^[1-9][0-9]{7,14}$`)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil && ctx.Err() == nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	mode := os.Getenv("JARVIS_WHATSAPP_MODE")
	var owner types.JID
	switch mode {
	case "self":
		// The linked account is the only identity allowed in this mode.
	case "contact":
		number := strings.TrimPrefix(strings.TrimSpace(os.Getenv("JARVIS_WHATSAPP_OWNER")), "+")
		if !phoneNumber.MatchString(number) {
			return fmt.Errorf("defina JARVIS_WHATSAPP_OWNER com o número autorizado, incluindo país e DDD")
		}
		owner = types.NewJID(number, types.DefaultUserServer)
	default:
		return fmt.Errorf("defina JARVIS_WHATSAPP_MODE=self ou contact")
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "jarvis", "whatsapp")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	dbPath := filepath.Join(dir, "device.db")
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
	device, err := store.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("load WhatsApp device: %w", err)
	}
	client := whatsmeow.NewClient(device, nil)
	client.EnableAutoReconnect = true
	client.ManualHistorySyncDownload = true
	defer client.Disconnect()

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
				log.Print("fila do WhatsApp cheia; mensagem descartada")
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
				log.Print("Conta do Jarvis vinculada")
			default:
				return fmt.Errorf("pareamento do WhatsApp: %s", item.Event)
			}
		}
	} else if err := client.ConnectContext(ctx); err != nil {
		return err
	}

	cfg := config.Load()
	llm := ai.NewOllamaProvider(cfg.OllamaURL, cfg.OllamaModel)
	seen := make(map[types.MessageID]struct{})
	recent := make([]types.MessageID, 0, 512)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-loggedOut:
			return fmt.Errorf("o WhatsApp desvinculou o Jarvis; inicie o processo novamente para parear")
		case event := <-incoming:
			source := event.Info.MessageSource
			prompt := incomingText(event)
			if mode == "self" {
				prompt = selfPrompt(source, device.GetJID(), device.GetLID(), prompt)
			} else if source.IsFromMe || !authorized(ctx, device.LIDs, source, owner) {
				continue
			}
			if prompt == "" {
				continue
			}
			id := event.Info.ID
			if _, duplicate := seen[id]; duplicate {
				continue
			}
			if len(recent) == cap(recent) {
				delete(seen, recent[0])
				copy(recent, recent[1:])
				recent = recent[:len(recent)-1]
			}
			seen[id] = struct{}{}
			recent = append(recent, id)

			replyCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			reply, err := llm.Chat(replyCtx, prompt)
			cancel()
			if err != nil {
				log.Print("Ollama não respondeu à mensagem do WhatsApp")
				reply = "Não consegui gerar uma resposta agora. Tente novamente daqui a pouco."
			}
			if mode == "self" {
				reply = "Jarvis: " + reply
			}
			sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
			_, err = client.SendMessage(sendCtx, event.Info.Chat.ToNonAD(), &waE2E.Message{Conversation: proto.String(reply)})
			cancel()
			if err != nil {
				log.Print("Falha ao enviar resposta no WhatsApp")
			}
		}
	}
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

func selfPrompt(source types.MessageSource, ownPhone, ownLID types.JID, body string) string {
	if !source.IsFromMe || source.IsGroup {
		return ""
	}
	chat := source.Chat.ToNonAD()
	if chat.IsEmpty() || (chat != ownPhone.ToNonAD() && chat != ownLID.ToNonAD()) {
		return ""
	}
	const prefix = "/jarvis "
	if !strings.HasPrefix(body, prefix) {
		return ""
	}
	return strings.TrimSpace(strings.TrimPrefix(body, prefix))
}

type lidLookup interface {
	GetPNForLID(context.Context, types.JID) (types.JID, error)
}

func authorized(ctx context.Context, lookup lidLookup, source types.MessageSource, owner types.JID) bool {
	sender := source.Sender.ToNonAD()
	if source.Chat.ToNonAD() != sender {
		return false
	}
	switch sender.Server {
	case types.DefaultUserServer:
		return sender == owner
	case types.HiddenUserServer:
		if source.SenderAlt.ToNonAD() == owner {
			return true
		}
		pn, err := lookup.GetPNForLID(ctx, sender)
		return err == nil && pn.ToNonAD() == owner
	default:
		return false
	}
}
