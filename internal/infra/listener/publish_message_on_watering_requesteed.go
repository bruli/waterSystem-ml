package listener

import (
	"context"
	"fmt"

	"github.com/bruli/go-core/cqs"
	"github.com/bruli/go-core/event"
	"github.com/bruli/go-core/ptr"
	"github.com/bruli/watersystem-ml/internal/app"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
)

type PublishMessageOnWateringRequested struct {
	ch cqs.CommandHandler
}

func (p PublishMessageOnWateringRequested) Listen(ctx context.Context, ev event.Event) error {
	wr, ok := ev.(*ml.WateringRequestedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: %T", ev)
	}
	message := "WaterSystem\nML Prediction\n"
	message += fmt.Sprintf("Watering zone: %s\n", wr.Zone)
	message += fmt.Sprintf("seconds: %v\n", wr.Seconds)
	message += fmt.Sprintf("reason: %s\n", ptr.FromPointer(wr.DecisionReason))
	message += fmt.Sprintf("soil moisture: %v\n", wr.MoistureBefore)
	if _, err := p.ch.Handle(ctx, app.PublishMessageCommand{Message: message}); err != nil {
		return fmt.Errorf("error publishing message: %w", err)
	}
	return nil
}

func NewPublishMessageOnWateringRequested(ch cqs.CommandHandler) *PublishMessageOnWateringRequested {
	return &PublishMessageOnWateringRequested{ch: ch}
}
