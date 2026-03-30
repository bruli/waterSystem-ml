package ml

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	pythonEnv   = "/opt/venv/bin/python"
	trainFile   = "python/train.py"
	predictFile = "python/predict.py"
)

func RunTraining(ctx context.Context, log *slog.Logger) error {
	output, err := runCommand(ctx, 30*time.Minute, pythonEnv, trainFile)
	if err != nil {
		return fmt.Errorf("error running training: %s", output)
	}
	if output != "" {
		log.InfoContext(ctx, "Training output", slog.String("output", output))
	}
	return nil
}
