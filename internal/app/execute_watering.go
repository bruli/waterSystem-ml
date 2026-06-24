package app

import (
	"context"

	"github.com/bruli/go-core/cqs"
	"github.com/bruli/go-core/event"
	"github.com/bruli/watersystem-ml/internal/domain/watering"
)

const ExecuteWateringCommandName = "execute_watering"

type ExecuteWateringCommand struct {
	Zone    string
	Seconds int
}

func (e ExecuteWateringCommand) Name() string {
	return ExecuteWateringCommandName
}

type ExecuteWatering struct {
	svc *watering.Execute
}

func (e ExecuteWatering) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	co, ok := cmd.(ExecuteWateringCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(ExecuteWateringCommandName, cmd.Name())
	}
	return nil, e.svc.Execute(ctx, watering.New(co.Zone, co.Seconds))
}

func NewExecuteWatering(svc *watering.Execute) *ExecuteWatering {
	return &ExecuteWatering{svc: svc}
}
