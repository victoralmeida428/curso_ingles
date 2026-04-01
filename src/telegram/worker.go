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
	isAudio := update.Message != nil && update.Message.Audio != nil

	if !isText && !isVoice && !isAudio {
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

			// LÓGICA INTELIGENTE: Verifica o status atual da assinatura do usuário
			if user.IsSubscribed() {
				switch user.PriceID {
				case "PRO":
					rawMsg.Text = "✅ <b>Você já possui o Plano Pro ativo!</b>\n\nSeu acesso total está liberado. Não é necessário assinar novamente."
					sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
					return
				case "BASIC":
					// Oferece apenas o Upgrade para o PRO
					linkPro, _ := payment.CreateCheckoutSession(chatID, string(assinatura.PlanProMonthly), cfg)
					rawMsg.Text = fmt.Sprintf(
						"✅ <b>Você possui o Plano Basic ativo!</b>\n\nDeseja fazer um <b>Upgrade para o Plano Pro</b> e liberar a conversação real por voz (30 áudios/dia)?\n\n<a href='%s'>👉 Fazer Upgrade para o Pro</a>",
						linkPro,
					)
					sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
					return
				}
			}

			// Se não for inscrito ou não tiver plano definido, mostra as duas opções
			linkBasic, _ := payment.CreateCheckoutSession(chatID, string(assinatura.PlanBasicMonthly), cfg) // Mapeado para o de R$ 10
			linkPro, _ := payment.CreateCheckoutSession(chatID, string(assinatura.PlanProMonthly), cfg)     // Mapeado para o de R$ 60

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

		case strings.HasPrefix(msgText, "/tipo "):
			user.Tipo = strings.TrimPrefix(msgText, "/tipo ")
			_ = database.SaveUser(user)
			rawMsg.Text = fmt.Sprintf("✅ Tipo definido para: %s", user.Tipo)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return

		case msgText == "/perfil":
			usoAudio := fmt.Sprintf("%d/30", user.DailyAudioCount)
			rawMsg.Text = fmt.Sprintf("👤 <b>Seu Perfil:</b>\nPlano Atual: <b>%s</b>\nIdioma: %s\nNível: %s\n\n🎙 <b>Uso de Voz hoje:</b> %s",
				fallback(user.PriceID), fallback(user.Lang), fallback(user.Nivel), usoAudio)
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
	} else if isVoice || isAudio {
		// CORREÇÃO DA TRAVA: Agora verifica a string "PRO"
		if user.PriceID != "PRO" {
			msgBlock := "🎙️ <b>Recurso Premium</b>\n\nO envio e recebimento de mensagens de voz é exclusivo do <b>Plano Pro</b>.\n\nUse /assinar para fazer o upgrade e liberar a conversação real!"
			sendTextMessage(ctx, b, chatID, msgBlock, models.ParseModeHTML)
			return // Para a execução aqui, não gasta API
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
