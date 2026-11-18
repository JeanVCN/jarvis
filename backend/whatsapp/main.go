// Command whatsapp runs the optional WhatsApp chat and voice-note channel.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"mime"
	"net"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/jean/jarvis/backend/internal/config"
	"github.com/jean/jarvis/backend/internal/conversation"
	"github.com/jean/jarvis/backend/internal/opencode"
	"github.com/jean/jarvis/backend/internal/voice"
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
	if len(os.Args) > 1 {
		if len(os.Args) != 2 || (os.Args[1] != "totp-setup" && os.Args[1] != "totp-reset") {
			log.Fatal("uso: whatsapp [totp-setup|totp-reset]")
		}
		if err := manageTOTP(context.Background(), os.Args[1]); err != nil {
			log.Fatal(err)
		}
		return
	}
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
	assistantName := whatsappAssistantName(os.Getenv("JARVIS_WHATSAPP_NAME"))
	persona := strings.TrimSpace(os.Getenv("JARVIS_WHATSAPP_PERSONA"))
	if persona == "" {
		persona = fmt.Sprintf("Você é %s, assistente pessoal do usuário. Fale em português brasileiro, com tom natural, cordial e respostas concisas. Ajude com dúvidas e tarefas cotidianas. Considere o histórico desta conversa para manter continuidade. Não use ferramentas, arquivos, comandos ou rede; não afirme ter executado ações externas. Use Markdown padrão quando útil; a resposta será adaptada para o WhatsApp.", assistantName)
	} else {
		persona = "Seu nome é " + assistantName + ". " + persona
	}
	provider, err := conversation.NewCodexTextWithThreadStore(os.Getenv("JARVIS_WHATSAPP_MODEL"), persona, whatsappThreadStore{db: routesDB})
	if err != nil {
		return fmt.Errorf("configure o perfil autenticado do Codex no Jarvis: %w", err)
	}
	recognizer, err := whatsappSpeechRecognizer(config.Load())
	if err != nil {
		return err
	}
	router := messageRouter{
		db:      routesDB,
		project: strings.TrimSpace(os.Getenv("JARVIS_WHATSAPP_OPENCODE_PROJECT")),
		chat:    func(ctx context.Context, prompt string) string { return generateReply(ctx, provider, prompt) },
		imageChat: func(ctx context.Context, prompt, mimeType string, data []byte) string {
			return generateImageReply(ctx, provider, prompt, mimeType, data)
		},
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
			if !isSelfChatSource(event.Info.MessageSource, owner, device.GetJID(), device.GetLID()) {
				return
			}
			assistantMessage, err := isAssistantMessageID(ctx, routesDB, string(event.Info.ID))
			if err != nil {
				stopRouting(fmt.Errorf("verificar mensagem enviada pelo Jarvis: %w", err))
				return
			}
			if assistantMessage {
				return
			}
			if incomingText(event) == "" && !isVoiceNote(event) && !isImageMessage(event) {
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
			replyCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
			prompt := selfPrompt(event.Info.MessageSource, owner, device.GetJID(), device.GetLID(), incomingText(event))
			isImage := isImageMessage(event)
			isVoice := !isImage && prompt == "" && isVoiceNote(event)
			transcriptionFailed := false
			imageFailed := false
			imageMIME := ""
			var imageData []byte
			if isImage {
				image := event.Message.GetImageMessage()
				var imageErr error
				imageMIME, imageErr = supportedWhatsAppImageMIME(image.GetMimetype())
				if imageErr == nil && image.GetFileLength() > maxWhatsAppImageBytes {
					imageErr = errors.New("imagem excede o limite de 10 MiB")
				}
				if imageErr == nil {
					imageData, imageErr = client.Download(replyCtx, image)
				}
				if imageErr == nil && (len(imageData) == 0 || len(imageData) > maxWhatsAppImageBytes) {
					imageErr = errors.New("tamanho de imagem inválido")
				}
				if imageErr != nil {
					log.Print("Não foi possível receber a imagem do WhatsApp")
					imageFailed = true
				}
			}
			if isVoice {
				transcript, sttErr := transcribeVoice(replyCtx, client, event, recognizer)
				if sttErr != nil {
					log.Print("Não foi possível transcrever nota de voz com o STT local")
					prompt = "Não consegui transcrever essa nota de voz localmente. Tente um áudio mais curto ou envie texto."
					transcriptionFailed = true
				} else {
					prompt = transcript
				}
			}
			if prompt == "" && !isImage {
				cancel()
				continue
			}
			var reply string
			var routeErr error
			if imageFailed {
				reply = "Não consegui encaminhar essa imagem. Envie uma imagem JPEG, PNG ou WebP de até 10 MiB."
			} else if isImage {
				reply, routeErr = router.replyImage(replyCtx, string(event.Info.ID), prompt, imageMIME, imageData)
			} else if transcriptionFailed {
				reply = prompt
			} else if isVoice {
				reply, routeErr = router.replyTranscription(replyCtx, string(event.Info.ID), prompt)
			} else {
				reply, routeErr = router.reply(replyCtx, string(event.Info.ID), prompt)
			}
			modeCtx, cancelMode := context.WithTimeout(ctx, 5*time.Second)
			codeMode, modeErr := router.codeMode(modeCtx)
			cancelMode()
			if routeErr == nil && modeErr != nil {
				routeErr = modeErr
			}
			clear(imageData)
			cancel()
			if routeErr != nil {
				return fmt.Errorf("roteamento do WhatsApp interrompido: %w", routeErr)
			}
			if reply == "" {
				continue
			}
			parts := splitWhatsAppMessage(formatWhatsAppMarkdown(reply), 12*1024-160)
			if codeMode && len(parts) > 0 {
				parts[len(parts)-1] += "\n\n*OpenCode*"
			}
			var sendErr error
			for _, part := range parts {
				sendCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
				messageID := client.GenerateMessageID()
				_, sendErr = routesDB.ExecContext(sendCtx, "INSERT OR IGNORE INTO jarvis_assistant_message_ids (message_id) VALUES (?)", string(messageID))
				if sendErr == nil {
					_, sendErr = client.SendMessage(sendCtx, event.Info.Chat.ToNonAD(), &waE2E.Message{Conversation: proto.String(part)}, whatsmeow.SendRequestExtra{ID: messageID})
				}
				cancel()
				if sendErr != nil {
					break
				}
			}
			if sendErr != nil {
				if !isVoice && !isImage && prompt == "/code" {
					return fmt.Errorf("não consegui confirmar o modo pelo WhatsApp; roteamento interrompido")
				}
				log.Print("Falha ao enviar resposta no WhatsApp")
			}
		}
	}
}

type whatsappThreadStore struct{ db *sql.DB }

func (s whatsappThreadStore) Get(ctx context.Context) (string, error) {
	var id string
	err := s.db.QueryRowContext(ctx, "SELECT thread_id FROM jarvis_codex_thread WHERE id = 1").Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return id, err
}

func (s whatsappThreadStore) Set(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, "INSERT INTO jarvis_codex_thread (id, thread_id) VALUES (1, ?) ON CONFLICT(id) DO UPDATE SET thread_id = excluded.thread_id", id)
	return err
}

const maxWhatsAppVoiceBytes = 15 << 20
const maxWhatsAppImageBytes = 10 << 20

func whatsappSpeechRecognizer(cfg config.Config) (voice.SpeechRecognizer, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.VoiceSTTProvider)) {
	case "", "command":
		return voice.NewCommandSpeechRecognizer(cfg.VoiceSTTCommand, voice.ParseArgs(cfg.VoiceSTTArgs), cfg.VoiceSTTModel, cfg.VoiceSTTLanguage), nil
	case "http":
		if !isLoopbackURL(cfg.VoiceSTTURL) || (cfg.VoiceSTTStatusURL != "" && !isLoopbackURL(cfg.VoiceSTTStatusURL)) {
			return nil, errors.New("o STT HTTP do WhatsApp precisa usar somente um endereço loopback local")
		}
		return voice.NewHTTPSpeechRecognizer(cfg.VoiceSTTURL, cfg.VoiceSTTStatusURL, cfg.VoiceSTTLanguage), nil
	default:
		return nil, errors.New("provedor de STT do WhatsApp inválido; use command ou HTTP local")
	}
}

func isLoopbackURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isVoiceNote(event *events.Message) bool {
	return event != nil && event.Message != nil && event.Message.GetAudioMessage().GetPTT()
}

func isImageMessage(event *events.Message) bool {
	return event != nil && event.Message != nil && event.Message.GetImageMessage() != nil
}

func whatsappAssistantName(raw string) string {
	name := strings.NewReplacer("\r", " ", "\n", " ", ":", "").Replace(strings.TrimSpace(raw))
	runes := []rune(name)
	if len(runes) > 32 {
		name = string(runes[:32])
	}
	if strings.TrimSpace(name) == "" {
		return "Jarvis"
	}
	return name
}

func supportedWhatsAppImageMIME(raw string) (string, error) {
	mediaType, _, err := mime.ParseMediaType(raw)
	if err != nil {
		return "", errors.New("tipo de imagem inválido")
	}
	switch strings.ToLower(mediaType) {
	case "image/jpeg", "image/png", "image/webp":
		return strings.ToLower(mediaType), nil
	default:
		return "", errors.New("tipo de imagem não suportado")
	}
}

func transcribeVoice(ctx context.Context, client *whatsmeow.Client, event *events.Message, recognizer voice.SpeechRecognizer) (string, error) {
	audio := event.Message.GetAudioMessage()
	if audio.GetFileLength() > maxWhatsAppVoiceBytes || audio.GetSeconds() > 120 {
		return "", errors.New("nota de voz excede o limite de 15 MiB ou 120 segundos")
	}
	data, err := client.Download(ctx, audio)
	if err != nil {
		return "", errors.New("download da nota de voz falhou")
	}
	defer clear(data)
	if len(data) == 0 || len(data) > maxWhatsAppVoiceBytes {
		return "", errors.New("tamanho da nota de voz inválido")
	}
	transcript, err := recognizer.Recognize(ctx, voice.AudioInput{Data: data, Format: audio.GetMimetype(), Language: "pt"})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(transcript), nil
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
	if body == "" {
		body = strings.TrimSpace(event.Message.GetImageMessage().GetCaption())
	}
	return body
}

func selfPrompt(source types.MessageSource, owner, ownPhone, ownLID types.JID, body string) string {
	if !isSelfChatSource(source, owner, ownPhone, ownLID) {
		return ""
	}
	return body
}

func isSelfChatSource(source types.MessageSource, owner, ownPhone, ownLID types.JID) bool {
	if !source.IsFromMe || source.IsGroup || ownPhone.ToNonAD() != owner {
		return false
	}
	chat := source.Chat.ToNonAD()
	return !chat.IsEmpty() && (chat == ownPhone.ToNonAD() || chat == ownLID.ToNonAD())
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

func generateImageReply(ctx context.Context, provider conversation.Provider, prompt, mimeType string, data []byte) string {
	imageProvider, ok := provider.(conversation.ImageProvider)
	if !ok {
		return "O provedor Codex configurado não aceita imagens. A imagem não foi encaminhada."
	}
	if prompt == "" {
		prompt = "Analise a imagem enviada."
	}
	var response strings.Builder
	calls, err := imageProvider.StreamImage(ctx, []conversation.Message{{Role: "user", Content: prompt}}, mimeType, data, func(delta string) error {
		response.WriteString(delta)
		return nil
	})
	reply := strings.TrimSpace(response.String())
	if err == nil && len(calls) == 0 && reply != "" {
		return reply
	}
	if err == nil && len(calls) > 0 {
		return "O Codex pediu uma ferramenta, mas o chat de imagens não permite executar ferramentas."
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
	log.Print("GPT indisponível para analisar a imagem do WhatsApp")
	return "Não consegui obter uma resposta sobre a imagem agora. Tente novamente daqui a pouco."
}
