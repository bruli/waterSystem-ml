package telegram

import (
	"context"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
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

func NewMessagePublisher(token string, chatID int, isProd bool) (*MessagePublisher, error) {
	var (
		bot *tgbotapi.BotAPI
		err error
	)
	switch {
	case isProd:
		bot, err = tgbotapi.NewBotAPI(token)
	default:
		bot, err = tgbotapi.NewBotAPIWithAPIEndpoint(token, "http://mockserver:1080/bot%s/%s")
	}
	if err != nil {
		return nil, err
	}
	pub := MessagePublisher{chatID: chatID, bot: bot}
	return &pub, nil
}
