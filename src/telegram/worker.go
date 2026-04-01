package telegram

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"curso/src/config"
	"curso/src/database"
	"curso/src/openrouter"
	"curso/src/payment"
	"curso/src/payment/assinatura"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// Constante para o funil de vendas (Isca gratuita)
const LimiteMensagensGratuitas = 5

type Worker struct {
	bot    *bot.Bot
	db     *sql.DB
	client openrouter.IClient
	cache  *HistoryCache
}

func (w *Worker) GetBot() *bot.Bot {
	return w.bot
}

func NewWorker(botToken string, db *sql.DB, client openrouter.IClient) *Worker {
	cache := NewHistoryCache(30)
	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {
			processMessage(ctx, b, update, client, cache)
		}),
	}
	b, err := bot.New(botToken, opts...)
	if err != nil {
		log.Fatalf("Erro ao inicializar o bot: %v", err)
	}
	return &Worker{
		bot:    b,
		db:     db,
		client: client,
		cache:  cache,
	}
}

func (w *Worker) StartWorker() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	comandos := []models.BotCommand{
		{Command: "start", Description: "Inicia o bot e ganhe mensagens grátis"},
		{Command: "perfil", Description: "Mostra o perfil atual"},
		{Command: "lang", Description: "Define o idioma"},
		{Command: "tipo", Description: "Define o vocabulário"},
		{Command: "nivel", Description: "Define seu nível atual"},
		{Command: "assinar", Description: "Destrave o acesso ilimitado"},
	}

	if _, err := w.bot.SetMyCommands(ctx, &bot.SetMyCommandsParams{Commands: comandos}); err != nil {
		log.Printf("Erro ao definir comandos: %v", err)
	}

	log.Println("🤖 Bot do Telegram em execução! Pressione Ctrl+C para parar.")
	w.bot.Start(ctx)
}

func processMessage(ctx context.Context, b *bot.Bot, update *models.Update, client openrouter.IClient, cache *HistoryCache) {
	isText := update.Message != nil && update.Message.Text != ""
	isVoice := update.Message != nil && update.Message.Voice != nil
	isAudio := update.Message != nil && update.Message.Audio != nil

	if !isText && !isVoice && !isAudio {
		return
	}

	chatID := update.Message.Chat.ID
	msgText := update.Message.Text
	user := database.GetUser(chatID)
	isElegivel := !user.TrialUsed

	if user.ID == 0 {
		_ = database.SaveUser(user)
	}

	rawMsg := &RawTelegramMessage{
		ChatID: chatID,
	}

	isCommand := strings.HasPrefix(msgText, "/")

	// ==========================================
	// 🚀 FUNIL DE VENDAS (PAYWALL INTELIGENTE)
	// ==========================================
	if !user.IsSubscribed() && !isCommand {
		if user.FreeUsed >= LimiteMensagensGratuitas {
			// Bateu no limite. Mostra o valor que ele perdeu e oferece o plano.
			rawMsg.Text = "⏳ <b>Seu teste gratuito chegou ao fim!</b>\n\nVocê mandou super bem nas respostas! Para continuar praticando e não perder o ritmo, escolha um dos nossos planos.\n\n👉 Digite /assinar para destravar o Zellang."
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		} else {
			// Ainda tem mensagens grátis. Incrementa o uso no banco de dados silenciosamente.
			_ = database.IncrementFreeUse(chatID)
		}
	}

	if isText {
		switch {
		case strings.HasPrefix(msgText, "/start"):
			rawMsg.Text = fmt.Sprintf(`🚀 <b>Bora destravar seu inglês?</b>

				Você pode treinar conversação real aqui comigo — sem professor te julgando, sem horário fixo.

				🎁 <b>Presente:</b> Você ganhou %d mensagens gratuitas para testar a IA agora mesmo!

				👇 <b>O que você quer treinar hoje?</b> (Responda com 1, 2 ou 3)

				1️⃣ Conversação do dia a dia  
				2️⃣ Inglês para o trabalho  
				3️⃣ Viagem e Aeroporto`, LimiteMensagensGratuitas)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		case strings.HasPrefix(msgText, "/assinar"):
			cfg := config.LoadConfig()

			if user.IsSubscribed() {
				switch user.PriceID {
				case "PRO":
					rawMsg.Text = "✅ <b>Você já é VIP! (Plano Pro Ativo)</b>\n\nSeu acesso total está liberado. Não é necessário assinar novamente. Pode mandar áudio ou texto!"
					sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
					return
				case "BASIC":
					linkPro, _ := payment.CreateCheckoutSession(chatID, string(assinatura.PlanProMonthly), cfg, isElegivel)
					rawMsg.Text = fmt.Sprintf(
						"✅ <b>Você já possui o Plano Basic!</b>\n\nDeseja dar o próximo passo na fluência?\n\nFaça o <b>Upgrade para o Plano Pro</b> e libere a conversação por voz (Até 30 áudios/dia).\n\n<a href='%s'>👉 Quero falar inglês (Upgrade Pro)</a>",
						linkPro,
					)
					sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
					return
				}
			}

			linkBasic, _ := payment.CreateCheckoutSession(chatID, string(assinatura.PlanBasicMonthly), cfg, isElegivel)
			linkPro, _ := payment.CreateCheckoutSession(chatID, string(assinatura.PlanProMonthly), cfg, isElegivel)

			rawMsg.Text = fmt.Sprintf(
				"💎 <b>Escolha como você quer ficar fluente:</b>\n\n"+
					"🔹 <b>Plano Basic - R$ 10,00/mês</b>\n"+
					"• Prática ilimitada por <u>texto</u>.\n"+
					"• Correções nativas na hora.\n"+
					"<a href='%s'>👉 Assinar Basic</a>\n\n"+
					"🔸 <b>Plano Pro - R$ 60,00/mês</b>\n"+
					"• Prática completa: <u>texto e VOZ</u>.\n"+
					"• Até 30 áudios por dia (Escute e Fale).\n"+
					"• Imersão real de intercâmbio.\n"+
					"<a href='%s'>👉 Assinar Pro (Mais Recomendado)</a>",
				linkBasic, linkPro,
			)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		// ... (comandos de lang, nivel, tipo mantidos)
		case strings.HasPrefix(msgText, "/lang "):
			user.Lang = strings.TrimPrefix(msgText, "/lang ")
			_ = database.SaveUser(user)
			rawMsg.Text = fmt.Sprintf("✅ Idioma definido para: %s", user.Lang)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		case strings.HasPrefix(msgText, "/nivel "):
			user.Nivel = strings.TrimPrefix(msgText, "/nivel ")
			_ = database.SaveUser(user)
			rawMsg.Text = fmt.Sprintf("✅ Nível definido para: %s", user.Nivel)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		case strings.HasPrefix(msgText, "/tipo "):
			user.Tipo = strings.TrimPrefix(msgText, "/tipo ")
			_ = database.SaveUser(user)
			rawMsg.Text = fmt.Sprintf("✅ Foco definido para: %s", user.Tipo)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		case msgText == "/perfil":
			usoAudio := fmt.Sprintf("%d/30", user.DailyAudioCount)
			statusAssinatura := user.PriceID
			if !user.IsSubscribed() {
				statusAssinatura = fmt.Sprintf("Grátis (%d/%d msgs)", user.FreeUsed, LimiteMensagensGratuitas)
			}

			rawMsg.Text = fmt.Sprintf("👤 <b>Seu Perfil:</b>\nPlano Atual: <b>%s</b>\nIdioma: %s\nNível: %s\n\n🎙 <b>Uso de Voz hoje:</b> %s",
				statusAssinatura, fallback(user.Lang), fallback(user.Nivel), usoAudio)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case msgText == "1" || msgText == "2" || msgText == "3":
			var novoTipo string
			var promptOculto string

			switch msgText {
			case "1":
				novoTipo = "Conversação do dia a dia"
				promptOculto = "Olá! Quero treinar conversação do dia a dia. Pode puxar assunto comigo de forma natural para começarmos?"
			case "2":
				novoTipo = "Inglês para o trabalho"
				promptOculto = "Olá! Quero treinar inglês focado no ambiente de trabalho. Pode simular uma situação de escritório ou reunião comigo?"
			case "3":
				novoTipo = "Viagem e Aeroporto"
				promptOculto = "Olá! Quero treinar vocabulário de viagem e aeroporto. Pode iniciar um roleplay (simulação) comigo como se eu estivesse viajando?"
			}

			// Salva a escolha no banco de dados para a IA lembrar depois
			user.Tipo = novoTipo
			_ = database.SaveUser(user)

			// Engana a IA: substitui o número isolado por um texto rico e explicativo
			rawMsg.Text = promptOculto

			// Manda para a IA processar a mensagem "oculta"
			_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
			return

		default:
			rawMsg.Text = msgText
			_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
		}
	} else if isVoice || isAudio {
		// UPSELL DE ÁUDIO MELHORADO
		if user.PriceID != "PRO" {
			msgBlock := "🎙️ <b>Gostaria de falar em vez de digitar?</b>\n\nA conversação real por voz é o que mais acelera a fluência, mas esse recurso é exclusivo do <b>Plano Pro</b>.\n\n👉 Digite /assinar para destravar o envio e recebimento de áudios!"
			sendTextMessage(ctx, b, chatID, msgBlock, models.ParseModeHTML)
			return
		}

		var fID string
		if isVoice {
			fID = update.Message.Voice.FileID
		} else {
			fID = update.Message.Audio.FileID
		}

		rawMsg.Voice = &VoicePayload{FileID: fID}
		_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
	}

	// [Conversão FFmpeg abaixo... mantida igual]
	if len(rawMsg.ResponseAudioBytes) > 0 {
		inputReader := bytes.NewReader(rawMsg.ResponseAudioBytes)
		var oggBuffer bytes.Buffer
		var errLog bytes.Buffer

		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-f", "s16le",
			"-ar", "24000",
			"-ac", "1",
			"-i", "pipe:0",
			"-c:a", "libopus",
			"-b:a", "48k",
			"-f", "ogg",
			"pipe:1",
		)

		cmd.Stdin = inputReader
		cmd.Stdout = &oggBuffer
		cmd.Stderr = &errLog

		if errCmd := cmd.Run(); errCmd == nil {
			caption := rawMsg.Text
			if len(caption) > 1024 {
				caption = caption[:1020] + "..."
			}

			_, err := b.SendVoice(ctx, &bot.SendVoiceParams{
				ChatID:    chatID,
				Voice:     &models.InputFileUpload{Filename: "resposta.ogg", Data: bytes.NewReader(oggBuffer.Bytes())},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})

			if err == nil {
				return
			}
			log.Printf("Erro ao enviar SendVoice: %v", err)
		} else {
			log.Printf("Erro no FFmpeg: %v | Log: %s", errCmd, errLog.String())
		}
	}
}

func fallback(value string) string {
	if value == "" {
		return "não definido"
	}
	return value
}
