package telegram

import (
	"context"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func sendTextMessage(ctx context.Context, b *bot.Bot, chatID int64, text string, parseMode models.ParseMode) {
	_, _ = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: parseMode,
	})
}