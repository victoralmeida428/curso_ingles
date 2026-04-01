package payment

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"

	"curso/src/config"
	"curso/src/database"
	"curso/src/payment/assinatura"

	"github.com/go-telegram/bot" // Import necessário para o tipo *bot.Bot
	"github.com/go-telegram/bot/models"
	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
)

// HandleWebhook agora recebe o ponteiro do bot para enviar mensagens
func HandleWebhook(cfg *config.Config, b *bot.Bot) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const MaxBodyBytes = int64(65536)
		payload, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes))
		if err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		event, err := webhook.ConstructEventWithOptions(
			payload,
			r.Header.Get("Stripe-Signature"),
			cfg.StripeWebhookSecret,
			webhook.ConstructEventOptions{
				IgnoreAPIVersionMismatch: true,
			},
		)
		if err != nil {
			log.Printf("⚠️ [WEBHOOK] Falha na assinatura. Segredo usado: %s | Erro: %v", cfg.StripeWebhookSecret, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if event.Type == "checkout.session.completed" {
			var session stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			chatIDStr := session.Metadata["chat_id"]
			chatID, _ := strconv.ParseInt(chatIDStr, 10, 64)
			priceID := session.Metadata["price_id"]

			days := 0
			if session.AmountTotal == 0 {
				days = 3
			} else {
				switch assinatura.Plan(priceID) {
				case assinatura.PlanBasicMonthly, assinatura.PlanProMonthly:
					days = 31
				case assinatura.PlanBasicSemiannual, assinatura.PlanProSemiannual:
					days = 183
				case assinatura.PlanBasicAnnual, assinatura.PlanProAnnual:
					days = 366
				}
			}

			log.Printf("⚙️ Atualizando assinatura: ChatID %d por %d dias", chatID, days)
			if days > 0 && chatID != 0 {
				errDb := database.UpdateSubscription(chatID, days, priceID)

				if errDb == nil {
					// ENVIA NOTIFICAÇÃO DE SUCESSO PARA O USUÁRIO NO TELEGRAM
					textoSucesso := "<b>🎉 Pagamento Confirmado!</b>\n\nObrigado por assinar o <b>Zellang</b>. Seu acesso foi liberado e você já pode continuar praticando seu inglês via texto ou áudio! 🚀"

					_, errMsg := b.SendMessage(r.Context(), &bot.SendMessageParams{
						ChatID:    chatID,
						Text:      textoSucesso,
						ParseMode: models.ParseModeHTML,
					})

					if errMsg != nil {
						log.Printf("❌ Erro ao enviar mensagem de confirmação: %v", errMsg)
					}
				} else {
					log.Printf("❌ Erro ao atualizar banco de dados: %v", errDb)
				}
			}
		}

		w.WriteHeader(http.StatusOK)
	}
}
