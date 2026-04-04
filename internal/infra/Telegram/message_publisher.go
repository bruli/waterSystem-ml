package telegram

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type MessagePublisher struct {
	chatID int
	bot    *tgbotapi.BotAPI
}

func (e MessagePublisher) Publish(ctx context.Context, message string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		msg := tgbotapi.NewMessage(int64(e.chatID), message)
		_, err := e.bot.Send(msg)
		return err
	}
}

func NewMessagePublisher(token string, chatID int) (*MessagePublisher, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	pub := MessagePublisher{chatID: chatID, bot: bot}
	return &pub, nil
}
