package payment

import (
	"curso/src/config"
	"fmt"

	"github.com/stripe/stripe-go/v85"
	"github.com/stripe/stripe-go/v85/checkout/session"
)

func CreateCheckoutSession(chatID int64, priceID string, cfg *config.Config) (string, error) {
	stripe.Key = cfg.StripeSecretKey

	params := &stripe.CheckoutSessionParams{
		SuccessURL:          stripe.String(cfg.TelegramURL),
		Mode:                stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		AllowPromotionCodes: stripe.Bool(true),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Price:    stripe.String(priceID),
				Quantity: stripe.Int64(1),
			},
		},
		Metadata: map[string]string{
			"chat_id":  fmt.Sprintf("%d", chatID),
			"price_id": priceID,
		},
	}

	s, err := session.New(params)
	if err != nil {
		return "", err
	}
	return s.URL, nil
}
