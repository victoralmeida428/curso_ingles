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

	if !user.IsSubscribed() && !strings.HasPrefix(msgText, "/assinar") && !strings.HasPrefix(msgText, "/start") {
		rawMsg.Text = "Você não é assinante. Use /assinar para assinar."
		sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
		return
	}

	if isText {
		switch {
		case strings.HasPrefix(msgText, "/start"):
			rawMsg.Text = `🚀 <b>Bem-vindo ao Zellang!</b> 🤖

				Eu sou seu mentor pessoal de inglês alimentado por Inteligência Artificial. Meu objetivo é ajudar você a alcançar a fluência de forma natural e dinâmica.

				🌟 <b>O que eu posso fazer?</b>
				• <b>Conversa Real:</b> Pratique inglês via texto ou <b>mensagens de voz</b>. Eu respondo você com áudio de alta fidelidade!
				• <b>Personalização:</b> Adaptado ao seu idioma nativo, nível de fluência e tipo de vocabulário (formal, técnico ou casual).
				• <b>Nivelamento:</b> Posso avaliar suas habilidades e definir seu nível CEFR (A1 a C2).

				🛠 <b>Como começar?</b>
				Primeiro, configure seu perfil para que eu possa ajustar minhas respostas:
				1. /lang - Defina seu idioma nativo (ex: <code>/lang Português</code>)
				2. /tipo - Escolha o foco (ex: <code>/tipo técnico</code>)
				3. /nivel - Defina seu nível (ex: <code>/nivel intermediário</code>)
				4. /nivelar - Inicie um teste guiado de 4 perguntas.

				👤 <b>Seu Perfil:</b> Use /perfil para ver suas configurações.
				💎 <b>Assinatura:</b> Use /assinar para liberar acesso total às conversas.

				<i>Diga "Hello" ou envie um áudio para começarmos a praticar!</i>`
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case strings.HasPrefix(msgText, "/assinar"):
			cfg := config.LoadConfig()
			// Gerar links para os 3 planos
			linkMensal, _ := payment.CreateCheckoutSession(chatID, cfg.PriceMonthlyID, cfg)
			linkSemestral, _ := payment.CreateCheckoutSession(chatID, cfg.PriceSemiannualID, cfg)
			linkAnual, _ := payment.CreateCheckoutSession(chatID, cfg.PriceAnnualID, cfg)

			rawMsg.Text = fmt.Sprintf(
				"💎 <b>Escolha seu plano Zellang:</b>\n\n"+
					"• <b>Mensal:</b> R$ 10,00\n<a href='%s'>👉 Assinar Mensal</a>\n\n"+
					"• <b>Semestral:</b> R$ 55,00 (Economia de R$ 5)\n<a href='%s'>👉 Assinar Semestral</a>\n\n"+
					"• <b>Anual:</b> R$ 100,00 (Melhor preço! R$ 8,33/mês)\n<a href='%s'>👉 Assinar Anual</a>",
				linkMensal, linkSemestral, linkAnual,
			)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case strings.HasPrefix(msgText, "/lang "):
			user.Lang = strings.TrimPrefix(msgText, "/lang ")
			_ = database.SaveUser(user)
			rawMsg.Text = fmt.Sprintf("✅ Idioma definido para: %s", user.Lang)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case strings.HasPrefix(msgText, "/tipo "):
			user.Tipo = strings.TrimPrefix(msgText, "/tipo ")
			_ = database.SaveUser(user)
			rawMsg.Text = fmt.Sprintf("✅ Tipo definido para: %s", user.Tipo)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case strings.HasPrefix(msgText, "/nivel "):
			user.Nivel = strings.TrimPrefix(msgText, "/nivel ")
			_ = database.SaveUser(user)
			rawMsg.Text = fmt.Sprintf("✅ Nível definido para: %s", user.Nivel)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case msgText == "/perfil":
			usoAudio := fmt.Sprintf("%d/%d", user.DailyAudioCount, MaxDailyAudios)
			rawMsg.Text = fmt.Sprintf("👤 <b>Seu Perfil:</b>\nIdioma: %s\nNível: %s\n\n🎙 <b>Uso de Voz hoje:</b> %s",
				fallback(user.Lang), fallback(user.Nivel), usoAudio)
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		case strings.HasPrefix(msgText, "/nivelar"):
			rawMsg.Text = "Você é agora meu avaliador de inglês. Inicie o meu teste de nivelamento para descobrir meu nível CEFR (A1 a C2). O teste deve avaliar as seguintes 4 habilidades: Vocabulário, Escrita, Pronúncia (exija que eu mande áudios testando a fala/leitura de textos que você enviar) e Leitura.\n\nRegras CRÍTICAS do teste:\n1. Faça EXATAMENTE UMA pergunta para cada habilidade (Total de 4 perguntas no teste inteiro).\n2. Faça apenas UMA pergunta por vez, esperando minha resposta antes de ir para a próxima habilidade.\n3. Após eu responder a 4ª e última pergunta, encerre o teste, me dê um feedback geral e o meu nível CEFR final (A1, A2, B1, B2, C1 ou C2).\n4. Faça a primeira pergunta do teste agora."
			_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
		case strings.HasPrefix(msgText, "/credits"):
			credit, err := client.GetCredits(ctx)
			if err != nil {
				rawMsg.Text = fmt.Sprintf("Erro ao obter créditos: %v", err)
			} else {
				rawMsg.Text = fmt.Sprintf("Créditos: %.2f", credit.Remaining)
			}
			sendTextMessage(ctx, b, chatID, rawMsg.Text, models.ParseModeHTML)
			return
		default:
			rawMsg.Text = msgText
			_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
		}
	} else if isVoice {
		rawMsg.Voice = &VoicePayload{FileID: update.Message.Voice.FileID}
		_ = processIncomingMessage(ctx, rawMsg, client, user, b, cache)
	}

	// =========================================================
	// ROTEAMENTO DE SAÍDA: ENVIA ÁUDIO NATIVO (VOICE) 100% EM RAM
	// =========================================================

	if len(rawMsg.ResponseAudioBytes) > 0 {
		// 1. Prepara o leitor com os bytes em MP3 recebidos da IA
		inputReader := bytes.NewReader(rawMsg.ResponseAudioBytes)

		// 2. Prepara os buffers de memória para receber a saída e os erros
		var oggBuffer bytes.Buffer
		var errLog bytes.Buffer

		// 3. Executa o FFmpeg na RAM (lê do stdin, converte para Opus, escreve no stdout)
		// O "-f ogg" é obrigatório aqui porque não temos um nome de arquivo para o FFmpeg adivinhar o formato
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-f", "s16le", // Formato de entrada: PCM 16-bit little-endian
			"-ar", "24000", // Sample rate: 24 kHz (padrão da OpenAI)
			"-ac", "1", // Canais: 1 (Mono)
			"-i", "pipe:0", // Origem: Memória RAM
			"-c:a", "libopus", // Codec de saída: Opus (Obrigatório pro Telegram)
			"-b:a", "48k", // Bitrate: 48 kbps
			"-f", "ogg", // Container: OGG
			"pipe:1", // Destino: Memória RAM
		)

		cmd.Stdin = inputReader
		cmd.Stdout = &oggBuffer
		cmd.Stderr = &errLog

		// 4. Inicia a conversão
		if errCmd := cmd.Run(); errCmd == nil {
			caption := rawMsg.Text
			if len(caption) > 1024 {
				caption = caption[:1020] + "..."
			}

			// 5. O buffer "oggBuffer" agora tem os bytes em OGG Opus perfeitos.
			// Usamos bytes.NewReader para enviá-lo direto ao Telegram.
			_, err := b.SendVoice(ctx, &bot.SendVoiceParams{
				ChatID:    chatID,
				Voice:     &models.InputFileUpload{Filename: "resposta.ogg", Data: bytes.NewReader(oggBuffer.Bytes())},
				Caption:   caption,
				ParseMode: models.ParseModeHTML,
			})

			if err == nil {
				return // Sucesso, áudio enviado perfeitamente!
			}
			log.Printf("Erro ao enviar SendVoice: %v", err)
		} else {
			log.Printf("Erro no FFmpeg (MP3 -> OGG em RAM): %v | Log: %s", errCmd, errLog.String())
		}
	}
}

func fallback(value string) string {
	if value == "" {
		return "não definido"
	}
	return value
}
