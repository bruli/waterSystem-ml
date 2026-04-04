package python

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	trainFile   = "python/train.py"
	predictFile = "python/predict.py"
)

type TrainingExecutor struct {
	timeout    time.Duration
	tracer     trace.Tracer
	log        *slog.Logger
	pythonPath string
}

func (t TrainingExecutor) Run(ctx context.Context) error {
	ctx, span := t.tracer.Start(ctx, "Training executor")
	defer span.End()
	output, err := runCommand(ctx, t.timeout, t.pythonPath, trainFile)
	if err != nil {
		err = fmt.Errorf("error running training: %s", output)
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if output != "" {
		t.log.InfoContext(ctx, "Training output", slog.String("output", output))
	}
	span.SetStatus(codes.Ok, "OK")
	return nil
}

func NewTrainingExecutor(
	timeout time.Duration,
	pythonPath string,
	tracer trace.Tracer,
	log *slog.Logger,
) *TrainingExecutor {
	return &TrainingExecutor{
		timeout:    timeout,
		tracer:     tracer,
		log:        log,
		pythonPath: pythonPath,
	}
}
