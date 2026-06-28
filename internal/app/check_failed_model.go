package app

import (
	"context"

	"github.com/bruli/go-core/cqs"
	"github.com/bruli/go-core/event"
	"github.com/bruli/watersystem-ml/internal/domain/ml"
)

const CheckFailedModelCommandName = "check_failed_model"

type CheckFailedModelCommand struct {
	Zone string
}

func (c CheckFailedModelCommand) Name() string {
	return CheckFailedModelCommandName
}

type CheckFailedModel struct {
	svc *ml.CheckModel
}

func (c CheckFailedModel) Handle(ctx context.Context, cmd cqs.Command) ([]event.Event, error) {
	co, ok := cmd.(CheckFailedModelCommand)
	if !ok {
		return nil, cqs.NewInvalidCommandError(CheckFailedModelCommandName, cmd.Name())
	}
	mh, err := c.svc.Check(ctx, co.Zone)
	if err != nil {
		return nil, err
	}
	return mh.Events(), nil
}

func NewCheckFailedModel(svc *ml.CheckModel) *CheckFailedModel {
	return &CheckFailedModel{svc: svc}
}
