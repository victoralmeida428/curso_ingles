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

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

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
		{Command: "start", Description: "Inicia o bot e mostra as instruções"},
		{Command: "perfil", Description: "Mostra o perfil atual"},
		{Command: "lang", Description: "Define o idioma"},
		{Command: "tipo", Description: "Define o vocabulário"},
		{Command: "nivel", Description: "Define seu nível atual"},
		{Command: "nivelar", Description: "Faz um teste guiado para descobrir o seu nível de inglês"},
		{Command: "assinar", Description: "Assina o Zellang"},
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
	if !isText && !isVoice {
		return
	}

	chatID := update.Message.Chat.ID
	msgText := update.Message.Text
	user := database.GetUser(chatID)
	if user.ID == 0 {
		_ = database.SaveUser(user)
	}

	rawMsg := &RawTelegramMessage{
		ChatID: chatID,
	}

	// Verifica se o utilizador é subscritor antes de permitir conversa
	if !user.IsSubscribed() && !strings.HasPrefix(msgText, "/assinar") && !strings.HasPrefix(msgText, "/start") {
		rawMsg.Text = "Você não possui uma assinatura ativa. Use /assinar para escolher um plano e começar a praticar."
		sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
		return
	}

	if isText {
		switch {
		case strings.HasPrefix(msgText, "/start"):
			rawMsg.Text = `🚀 <b>Bem-vindo ao Zellang!</b> 🤖

                Eu sou seu mentor pessoal de idiomas. Vamos acelerar a sua fluência!

                🌟 <b>Nossos Planos:</b>
                • <b>Basic (R$ 10):</b> Pratique via texto de forma ilimitada.
                • <b>Pro (R$ 60):</b> Conversação real por voz (até 30 áudios/dia).

                🛠 <b>Como começar?</b>
                1. /lang - Defina o idioma que quer praticar.
                2. /nivel - Defina o seu nível.
                3. /assinar - Escolha o seu plano.

                <i>Diga "Hello" ou envie um texto para começarmos!</i>`
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		case strings.HasPrefix(msgText, "/assinar"):
			cfg := config.LoadConfig()

			// Links para os novos produtos (Certifique-se de atualizar os IDs no seu .env ou config)
			linkBasic, _ := payment.CreateCheckoutSession(chatID, string(payment.PlanBasicMonthly), cfg) // Mapeado para o de R$ 10
			linkPro, _ := payment.CreateCheckoutSession(chatID, string(payment.PlanProMonthly), cfg)     // Reaproveite um ID ou crie um novo para R$ 60

			rawMsg.Text = fmt.Sprintf(
				"💎 <b>Escolha o seu plano Zellang:</b>\n\n"+
					"🔹 <b>Plano Basic - R$ 10,00/mês</b>\n"+
					"• Conversas ilimitadas por <u>texto</u>.\n"+
					"<a href='%s'>👉 Assinar Basic</a>\n\n"+
					"🔸 <b>Plano Pro - R$ 60,00/mês</b>\n"+
					"• Conversas por <u>texto e voz</u>.\n"+
					"• Até 30 mensagens de áudio por dia.\n"+
					"<a href='%s'>👉 Assinar Pro</a>",
				linkBasic, linkPro,
			)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

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

		case msgText == "/perfil":
			// Aqui você pode adicionar a lógica de mostrar qual o plano do user
			usoAudio := fmt.Sprintf("%d/30", user.DailyAudioCount)
			rawMsg.Text = fmt.Sprintf("👤 <b>Seu Perfil:</b>\nIdioma: %s\nNível: %s\n\n🎙 <b>Uso de Voz hoje:</b> %s",
				fallback(user.Lang), fallback(user.Nivel), usoAudio)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case msgText == "/creditos":
			credits, err := client.GetCredits(ctx)
			if err != nil {
				rawMsg.Text = "Erro ao obter créditos"
				sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
				return
			}
			rawMsg.Text = fmt.Sprintf("💰 <b>Seus Créditos:</b>\nCréditos: %.2f", credits.Remaining)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		default:
			rawMsg.Text = msgText
			_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
		}
	} else if isVoice {
		if user.PriceID != string(payment.PlanProMonthly) {
			msgBlock := "🎙️ <b>Recurso Premium</b>\n\nO envio e recebimento de mensagens de voz é exclusivo do <b>Plano Pro</b>.\n\nUse /assinar para fazer o upgrade e liberar a conversação real!"
			sendTextMessage(ctx, b, chatID, msgBlock, models.ParseModeHTML)
			return // Para a execução aqui, não gasta API
		}
		rawMsg.Voice = &VoicePayload{FileID: update.Message.Voice.FileID}
		_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
	}

	// [O restante do código de conversão FFmpeg permanece igual]
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
