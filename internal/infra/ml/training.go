package ml

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

const (
	trainFile   = "python/train.py"
	predictFile = "python/predict.py"
)

func RunTraining(ctx context.Context, log *slog.Logger, pythonPath string) error {
	output, err := runCommand(ctx, 30*time.Minute, pythonPath, trainFile)
	if err != nil {
		return fmt.Errorf("error running training: %s", output)
	}
	if output != "" {
		log.InfoContext(ctx, "Training output", slog.String("output", output))
	}
	return nil
}
