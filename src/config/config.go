package config

import "os"

type (

	Config struct {
		OpenRouterAPIKey    string
		TelegramBotToken    string
		DatabasePath        string
		StripeSecretKey     string
		StripeWebhookSecret string
		TelegramURL         string
	}
)



func LoadConfig() *Config {
	return &Config{
		OpenRouterAPIKey:    os.Getenv("OPENROUTER_API_KEY"),
		TelegramBotToken:    os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabasePath:        "./usuarios.sqlite",
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		TelegramURL:         os.Getenv("TELEGRAM_URL"),
	}
}
