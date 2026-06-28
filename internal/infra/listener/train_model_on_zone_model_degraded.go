package listener

import (
	"context"
	"fmt"

	"github.com/bruli/go-core/event"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
)

type TrainModelOnZoneModelDegraded struct {
	ch chan struct{ Zone string }
}

func (t TrainModelOnZoneModelDegraded) Listen(ctx context.Context, ev event.Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:

	}
	zmd, ok := ev.(*ml.ZoneModelDegradedEvent)
	if !ok {
		return fmt.Errorf("invalid event type: %T", ev)
	}
	t.ch <- struct{ Zone string }{Zone: zmd.Zone}
	return nil
}

func NewTrainModelOnZoneModelDegraded(ch chan struct{ Zone string }) *TrainModelOnZoneModelDegraded {
	return &TrainModelOnZoneModelDegraded{ch: ch}
}
