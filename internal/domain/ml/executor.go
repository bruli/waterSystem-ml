package ml

import "context"

type TrainExecutor interface {
	Run(ctx context.Context) error
}
