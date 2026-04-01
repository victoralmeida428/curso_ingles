package telegram

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"curso/src/database"
	"curso/src/openrouter"
	"curso/src/openrouter/models"
	"curso/src/openrouter/types"

	"github.com/go-telegram/bot"
	bot_models "github.com/go-telegram/bot/models"
)

const MaxDailyAudios = 30

// processIncomingMessage gerencia a conversa com a IA e popula o rawMsg com a resposta.
func processIncomingMessage(ctx context.Context, rawMsg *RawTelegramMessage, client openrouter.IClient, user database.UserData, b *bot.Bot, cache *HistoryCache) (err error) {
	msgInfo, err := ClassifyMessage(rawMsg)
	if err != nil {
		return err
	}

	// Defer seguro para capturar panics ou erros
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic na IA: %v", r)
			rawMsg.Text = "Desculpe, ocorreu um erro interno."
		} else if err != nil {
			log.Printf("Erro na IA: %v", err)
			rawMsg.Text = "Desculpe, ocorreu um erro de comunicação com a IA."
		}
	}()

	// 1. SALVA NA FILA: A mensagem atual do usuário
	textoUsuario := msgInfo.Text
	if msgInfo.Type == Audio {
		textoUsuario = "[O usuário enviou uma mensagem de áudio.]"
	}
	cache.AddMessage(rawMsg.ChatID, "user", textoUsuario)

	// Lógica do agente classificador
	var classificadorInput string
	if msgInfo.Type == Text {
		classificadorInput = msgInfo.Text
	} else {
		classificadorInput = "O usuário enviou uma mensagem de áudio. Por favor, decida se devo responder em áudio ou texto."
	}

	responderEmTexto := isTextAnswer(ctx, classificadorInput, client)

	// 🔒 TRAVA 1: Bloqueia resposta em áudio se não for PRO
	if user.PriceID != "PRO" {
		responderEmTexto = true
	}

	// 🔒 TRAVA 2: Bloqueia resposta em áudio se já atingiu o limite de 30
	if !responderEmTexto && user.DailyAudioCount >= MaxDailyAudios {
		responderEmTexto = true
	}

	// Prepara a base com o Prompt de Sistema
	messages := []types.Message{
		{
			Role:    "system",
			Content: BuildGlobalSystemPrompt(user),
		},
	}

	// 2. RECUPERA DA FILA: Busca as últimas interações do cache
	historico := cache.GetMessages(rawMsg.ChatID)
	if len(historico) > 0 {
		messages = append(messages, historico...)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 180*time.Second)
	defer cancel()

	// 3. Monta a requisição final
	chat := types.ChatCompletionRequest{Messages: messages}

	switch msgInfo.Type {
	case Text:
		// Se for texto, não precisamos dar append novamente.

	case Audio:
		// 🔒 TRAVA DA ENTRADA: Evita processar a entrada do usuário se já estourou o limite
		if user.DailyAudioCount >= MaxDailyAudios {
			// Alterado de 20 para MaxDailyAudios (30)
			rawMsg.Text = fmt.Sprintf("⚠️ Você atingiu seu limite diário de áudios (%d/dia). Para continuar praticando agora, use mensagens de texto!", MaxDailyAudios)
			return nil
		}

		if len(chat.Messages) > 0 {
			chat.Messages = chat.Messages[:len(chat.Messages)-1]
		}

		var base64Str string
		base64Str, err = fetchAndConvertAudio(reqCtx, b, rawMsg.Voice.FileID)
		if err != nil {
			return err
		}
		promptText := "Please listen to this audio and respond accordingly."
		chat.Messages = append(chat.Messages, types.Message{
			Role: "user",
			Content: []types.ContentPart{
				{Type: "text", Text: &promptText},
				{Type: "input_audio", InputAudio: &types.InputAudio{Data: base64Str, Format: "mp3"}},
			},
		})

	case Unknown:
		rawMsg.Text = "Desculpe, só entendo texto e áudio."
		return nil
	}

	// Configuração do modelo baseado na decisão do agente (com as travas já aplicadas)
	if responderEmTexto {
		chat.Model = models.TEXT
		chat.Modalities = []string{"text"}
	} else {
		chat.Model = models.AUDIO
		chat.Modalities = []string{"text", "audio"}
		chat.Audio = &types.AudioConfig{
			Voice:  "alloy",
			Format: "pcm16",
		}
	}

	// Inicia o streaming
	streamChan, err := client.ChatCompleteStream(reqCtx, &chat)
	if err != nil {
		return fmt.Errorf("erro ao iniciar stream: %w", err)
	}

	var fullText strings.Builder
	var fullAudio strings.Builder
	var messageID int

	// Se for texto, envia uma mensagem de "placeholder" para ir atualizando depois
	if responderEmTexto {
		msg, errSend := b.SendMessage(reqCtx, &bot.SendMessageParams{
			ChatID:    rawMsg.ChatID,
			Text:      "⏳ <i>Digitando...</i>",
			ParseMode: bot_models.ParseModeHTML,
		})
		if errSend == nil {
			messageID = msg.ID
		}
	}

	// Temporizador para evitar Rate Limit do Telegram
	lastEditTime := time.Now()
	tickerPeriod := 800 * time.Millisecond

	// Consome o canal de eventos
	for event := range streamChan {
		if event.Error != nil {
			log.Printf("Erro no meio do stream: %v", event.Error)
			break
		}

		if event.Transcript != "" {
			fullText.WriteString(event.Transcript)
		}

		if event.Audio != nil && event.Audio.Data != "" {
			fullAudio.WriteString(event.Audio.Data)
		}

		if responderEmTexto && messageID != 0 {
			if time.Since(lastEditTime) > tickerPeriod {
				currentText := fullText.String()
				if currentText != "" {
					_, errEdit := b.EditMessageText(reqCtx, &bot.EditMessageTextParams{
						ChatID:    rawMsg.ChatID,
						MessageID: messageID,
						Text:      currentText + " ✍️",
					})

					if errEdit != nil && !strings.Contains(errEdit.Error(), "message is not modified") {
						log.Printf("Aviso ao editar mensagem de stream: %v", errEdit)
					}
					lastEditTime = time.Now()
				}
			}
		}
	}

	textoFinal := strings.TrimSpace(fullText.String())
	rawMsg.Text = textoFinal

	// 4. SALVA NA FILA: A resposta final da IA
	if textoFinal != "" {
		cache.AddMessage(rawMsg.ChatID, "assistant", textoFinal)
	}

	if responderEmTexto {
		if messageID != 0 {
			if textoFinal != "" {
				_, errFinal := b.EditMessageText(reqCtx, &bot.EditMessageTextParams{
					ChatID:    rawMsg.ChatID,
					MessageID: messageID,
					Text:      textoFinal,
					ParseMode: bot_models.ParseModeHTML,
				})
				if errFinal != nil && !strings.Contains(errFinal.Error(), "message is not modified") {
					log.Printf("Erro na edição FINAL da mensagem: %v", errFinal)
				}
			} else {
				_, _ = b.EditMessageText(reqCtx, &bot.EditMessageTextParams{
					ChatID:    rawMsg.ChatID,
					MessageID: messageID,
					Text:      "❌ A IA não retornou nenhum texto.",
				})
			}
		}
	} else {
		// INÍCIO DO TRATAMENTO DE ÁUDIO
		base64Total := fullAudio.String()
		if base64Total != "" {
			if m := len(base64Total) % 4; m != 0 {
				base64Total += strings.Repeat("=", 4-m)
			}
			decodedBytes, errDecode := base64.StdEncoding.DecodeString(base64Total)
			if errDecode == nil {
				rawMsg.ResponseAudioBytes = decodedBytes

				// 💰 INCREMENTA O CONTADOR APÓS GERAR O ÁUDIO COM SUCESSO
				errInc := database.IncrementAudioUsage(rawMsg.ChatID)
				if errInc != nil {
					log.Printf("Aviso: Falha ao incrementar o contador de áudio para o chatID %d: %v", rawMsg.ChatID, errInc)
				}

			} else {
				log.Printf("Erro ao decodificar Base64: %v", errDecode)
			}
		} else {
			log.Printf("Aviso: Formato era áudio, mas Base64 chegou vazio.")
			sendTextMessage(reqCtx, b, rawMsg.ChatID, "Poderia repetir de alguma outra forma?", bot_models.ParseModeHTML)
		}
	}

	return nil
}

// isTextAnswer utiliza um LLM leve para decidir o formato de resposta com base na entrada do usuário.
func isTextAnswer(ctx context.Context, input string, client openrouter.IClient) bool {
	promt := "Você é um classificador para saber se o professor vai responder por texto ou audio. Responda apenas com 'true' (por texto) ou 'false' (por audio)."
	messages := []types.Message{
		{Role: "system", Content: promt},
		{Role: "user", Content: input},
	}
	chat := types.ChatCompletionRequest{
		Model:      models.TEXT,
		Messages:   messages,
		Modalities: []string{"text"},
	}
	resp, err := client.ChatComplete(ctx, &chat)
	if err != nil || len(resp.Choices) == 0 {
		return true // Fallback de segurança: sempre responde por texto se falhar
	}

	textoIA := strings.TrimSpace(strings.ToLower(resp.Choices[0].Message.Content))
	return strings.Contains(textoIA, "true")
}
