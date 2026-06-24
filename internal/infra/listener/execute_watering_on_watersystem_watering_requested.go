package listener

import (
	"context"
	"fmt"

	"github.com/bruli/go-core/cqs"
	"github.com/bruli/go-core/event"
	"github.com/bruli/watersystem-ml/internal/app"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
)

type ExecuteWateringOnWatersystemWateringRequested struct {
	ch cqs.CommandHandler
}

func (e ExecuteWateringOnWatersystemWateringRequested) Listen(ctx context.Context, ev event.Event) error {
	swr, ok := ev.(*ml.WateringRequestedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: %T", ev)
	}
	if _, err := e.ch.Handle(ctx, app.ExecuteWateringCommand{
		Zone:    swr.Zone,
		Seconds: int(swr.Seconds),
	}); err != nil {
		return fmt.Errorf("error executing watering: %w", err)
	}
	return nil
}

func NewExecuteWateringOnWatersystemWateringRequested(ch cqs.CommandHandler) *ExecuteWateringOnWatersystemWateringRequested {
	return &ExecuteWateringOnWatersystemWateringRequested{ch: ch}
}
