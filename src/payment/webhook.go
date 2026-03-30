package payment

import (
	"encoding/json"
	"io"
	"log" // Importação necessária para os logs
	"net/http"
	"strconv"

	"curso/src/config"
	"curso/src/database"

	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/webhook"
)

func HandleWebhook(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("📩 Webhook recebido do Stripe...")

		const MaxBodyBytes = int64(65536)
		payload, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes))
		if err != nil {
			log.Printf("❌ Erro ao ler corpo da requisição: %v", err)
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Verifica a assinatura do Stripe
		event, err := webhook.ConstructEventWithOptions(
			payload,
			r.Header.Get("Stripe-Signature"),
			cfg.StripeWebhookSecret,
			webhook.ConstructEventOptions{
				IgnoreAPIVersionMismatch: true,
			},
		)
		if err != nil {
			// Este é o log mais importante para diagnosticar o erro 400
			log.Printf("⚠️ Falha na verificação da assinatura: %v", err)
			log.Printf("DICA: Verifique se o STRIPE_WEBHOOK_SECRET no .env é igual ao da CLI.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}


		if event.Type == "checkout.session.completed" {
			var session stripe.CheckoutSession
			err := json.Unmarshal(event.Data.Raw, &session)
			if err != nil {
				log.Printf("❌ Erro ao decodificar JSON da sessão: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			// Debug dos Metadados
			chatIDStr := session.Metadata["chat_id"]
			log.Printf("🔍 Metadata chat_id: %s", chatIDStr)

			chatID, errConvert := strconv.ParseInt(chatIDStr, 10, 64)
			if errConvert != nil {
				log.Printf("❌ Erro ao converter chat_id para int64: %v", errConvert)
			}

			// Identifica o preço/plano
			priceID := session.Metadata["price_id"]
			log.Printf("🔍 Price ID recebido: %s", priceID)

			days := 0
			switch priceID {
			case cfg.PriceMonthlyID:
				days = 31
			case cfg.PriceSemiannualID:
				days = 183
			case cfg.PriceAnnualID:
				days = 366
			default:
				log.Printf("⚠️ Price ID não reconhecido: %s", priceID)
			}

			if days > 0 && chatID != 0 {
				log.Printf("⚙️ Atualizando assinatura: ChatID %d por %d dias", chatID, days)
				errDb := database.UpdateSubscription(chatID, days)
				if errDb != nil {
					log.Printf("❌ Erro ao atualizar banco de dados: %v", errDb)
				} 
			}
		}

		// Responde 200 para o Stripe não tentar reenviar o mesmo evento
		w.WriteHeader(http.StatusOK)
	}
}
