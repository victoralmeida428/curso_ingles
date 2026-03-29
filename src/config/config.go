package config

import "os"

type ( 
	Config struct {
		OpenRouterAPIKey string
		TelegramBotToken string
		DatabasePath string
	}
)

func LoadConfig() *Config {
	return &Config{
		OpenRouterAPIKey: os.Getenv("OPENROUTER_API_KEY"),
		TelegramBotToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		DatabasePath: "./usuarios.sqlite",
	}
}